package telegram

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/ValievM/family-expense-bot/internal/ports"
)

const (
	pollTimeoutSeconds = 30
	retryDelay         = 3 * time.Second
	handleTimeout      = 90 * time.Second
	maxMessageRunes    = 4096
	rememberTimeout    = 3 * time.Second
	drainTimeout       = 8 * time.Second

	failureReply   = "Что-то пошло не так, попробуйте ещё раз чуть позже."
	deniedReply    = "У вас нет доступа к этому боту."
	privateOnlyMsg = "Я работаю только в личных сообщениях."
)

type Gateway interface {
	GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error)
	SendMessage(ctx context.Context, chatID int64, text string) error
	SendTyping(ctx context.Context, chatID int64) error
	SendHTML(ctx context.Context, chatID int64, html string) error
	SetProfile(ctx context.Context, profile Profile) error
}

// Bot читает обновления, фильтрует доступ и отвечает; сообщения одного пользователя идут строго по порядку.
type Bot struct {
	gateway   Gateway
	responder ports.Responder
	access    ports.AccessPolicy
	directory ports.MemberDirectory
	logger    *slog.Logger
	retry     time.Duration
	drain     time.Duration
	queues    sync.Map // userID -> *userQueue
}

// userQueue — очередь сообщений одного пользователя; разбирает её не больше одной горутины.
type userQueue struct {
	mutex   sync.Mutex
	pending []Message
	running bool
}

type BotOption func(*Bot)

func WithLogger(logger *slog.Logger) BotOption {
	return func(bot *Bot) { bot.logger = logger }
}

func WithMemberDirectory(directory ports.MemberDirectory) BotOption {
	return func(bot *Bot) { bot.directory = directory }
}

func WithRetryDelay(delay time.Duration) BotOption {
	return func(bot *Bot) { bot.retry = delay }
}

func WithDrainTimeout(timeout time.Duration) BotOption {
	return func(bot *Bot) { bot.drain = timeout }
}

func NewBot(gateway Gateway, responder ports.Responder, access ports.AccessPolicy, options ...BotOption) *Bot {
	bot := &Bot{gateway: gateway, responder: responder, access: access, logger: slog.Default(), retry: retryDelay, drain: drainTimeout}
	for _, option := range options {
		option(bot)
	}
	return bot
}

// Принимает: ctx — контекст, отмена которого останавливает бота (сигнал завершения процесса).
// Делает: читает обновления и раздаёт сообщения по очередям пользователей. После остановки новые сообщения не берёт,
// очереди не разбирает дальше, а уже начатые ответы даёт довести до конца в пределах drain; затем отменяет остатки.
// Возвращает: nil после остановки.
func (bot *Bot) Run(ctx context.Context) error {
	// Работа над сообщениями не зависит от сигнала остановки: ответ, который уже начат, должен дойти до пользователя.
	work, abandon := context.WithCancel(context.WithoutCancel(ctx))
	defer abandon()
	var inFlight sync.WaitGroup
	defer bot.drainInFlight(&inFlight, abandon)

	if err := bot.gateway.SetProfile(ctx, defaultProfile); err != nil {
		bot.logger.WarnContext(ctx, "telegram bot profile was not updated", "error", err)
	}

	var offset int64
	for ctx.Err() == nil {
		updates, err := bot.gateway.GetUpdates(ctx, offset, pollTimeoutSeconds)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			bot.logger.ErrorContext(ctx, "telegram polling failed", "error", err)
			bot.wait(ctx, bot.retry)
			continue
		}
		for _, update := range updates {
			offset = update.UpdateID + 1
			if update.Message == nil {
				continue
			}
			bot.dispatch(ctx, work, &inFlight, *update.Message)
		}
	}
	return nil
}

// Принимает: inFlight — счётчик работающих обработчиков; abandon — отмена их контекста.
// Делает: ждёт завершения начатых ответов не дольше bot.drain, после чего отменяет оставшиеся.
// Возвращает: ничего.
func (bot *Bot) drainInFlight(inFlight *sync.WaitGroup, abandon context.CancelFunc) {
	done := make(chan struct{})
	go func() {
		inFlight.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(bot.drain):
		bot.logger.Warn("telegram handlers did not finish before the shutdown deadline", "deadline", bot.drain)
		abandon()
	}
}

func (bot *Bot) wait(ctx context.Context, delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

// Принимает: ctx; message — входящее сообщение.
// Делает: отбрасывает чужие типы, проверяет доступ, вызывает обработчик, отправляет ответ.
// Возвращает: ничего; сбои пишутся в лог, пользователь получает нейтральный ответ.
// Принимает: ctx — контекст бота; message — входящее сообщение.
// Делает: обрабатывает сообщение и перехватывает панику, чтобы одно сообщение не уронило весь процесс.
// Возвращает: ничего; при панике пользователь получает нейтральный ответ, а паника пишется в лог.
func (bot *Bot) safeHandle(ctx context.Context, message Message) {
	defer func() {
		if recovered := recover(); recovered != nil {
			bot.logger.ErrorContext(ctx, "panic while handling a message", "panic", recovered)
			bot.reply(ctx, message.Chat.ID, failureReply)
		}
	}()
	bot.handle(ctx, message)
}

func (bot *Bot) handle(ctx context.Context, message Message) {
	if message.From == nil || message.From.IsBot || strings.TrimSpace(message.Text) == "" {
		return
	}
	userID, chatID := message.From.ID, message.Chat.ID

	if message.Chat.Type != "private" {
		bot.reply(ctx, chatID, privateOnlyMsg)
		return
	}
	if !bot.access.IsUserAllowed(userID) {
		bot.logger.WarnContext(ctx, "telegram access denied", "user_id", userID)
		bot.reply(ctx, chatID, deniedReply)
		return
	}
	bot.rememberMember(ctx, message.From)
	if isHelpCommand(message.Text) {
		bot.replyHTML(ctx, chatID, greetingHTML)
		return
	}

	if err := bot.gateway.SendTyping(ctx, chatID); err != nil {
		bot.logger.WarnContext(ctx, "telegram typing indicator failed", "error", err)
	}
	handleCtx, cancel := context.WithTimeout(ctx, handleTimeout)
	defer cancel()

	answer, err := bot.responder.Handle(handleCtx, userID, message.Text)
	if err != nil {
		bot.logger.ErrorContext(ctx, "message handling failed", "user_id", userID, "error", err)
		answer = failureReply
	}
	bot.reply(ctx, chatID, answer)
}

// Принимает: ctx; user — автор сообщения, уже прошедший проверку доступа.
// Делает: если справочник подключён, запоминает имя и username из профиля Telegram с коротким таймаутом.
// Возвращает: ничего; сбой пишется в лог и не мешает ответу.
func (bot *Bot) rememberMember(ctx context.Context, user *User) {
	if bot.directory == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, rememberTimeout)
	defer cancel()
	name := strings.TrimSpace(user.FirstName + " " + user.LastName)
	if err := bot.directory.Remember(ctx, user.ID, name, user.Username); err != nil {
		bot.logger.WarnContext(ctx, "remember family member failed", "user_id", user.ID, "error", err)
	}
}

// Принимает: ctx; chatID — получатель; text — ответ.
// Делает: обрезает ответ до лимита Telegram и отправляет; ошибку отправки пишет в лог.
// Возвращает: ничего.
func (bot *Bot) replyHTML(ctx context.Context, chatID int64, html string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), requestTimeout)
	defer cancel()
	if err := bot.gateway.SendHTML(ctx, chatID, html); err != nil {
		bot.logger.ErrorContext(ctx, "telegram send failed", "chat_id", chatID, "error", err)
	}
}

func (bot *Bot) reply(ctx context.Context, chatID int64, text string) {
	// Ответ нужно доставить даже во время остановки, поэтому отмена бота на отправку не действует.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), requestTimeout)
	defer cancel()
	if err := bot.gateway.SendMessage(ctx, chatID, truncate(text, maxMessageRunes)); err != nil {
		bot.logger.ErrorContext(ctx, "telegram send failed", "chat_id", chatID, "error", err)
	}
}

// Принимает: ctx; inFlight — счётчик обработчиков; message — входящее сообщение.
// Делает: ставит сообщение в очередь автора и при нужде запускает её разбор; разные пользователи идут параллельно.
// Возвращает: ничего.
func (bot *Bot) dispatch(stop, work context.Context, inFlight *sync.WaitGroup, message Message) {
	var userID int64
	if message.From != nil {
		userID = message.From.ID
	}
	value, _ := bot.queues.LoadOrStore(userID, &userQueue{})
	queue := value.(*userQueue)

	queue.mutex.Lock()
	queue.pending = append(queue.pending, message)
	start := !queue.running
	queue.running = true
	queue.mutex.Unlock()

	if !start {
		return
	}
	inFlight.Add(1)
	go func() {
		defer inFlight.Done()
		for {
			queue.mutex.Lock()
			if len(queue.pending) == 0 {
				queue.running = false
				queue.mutex.Unlock()
				return
			}
			if stop.Err() != nil {
				// После остановки новые сообщения из очереди не берём: Telegram их уже подтвердил, но ответить на них мы не успеем.
				bot.logger.Warn("telegram messages dropped on shutdown", "user_id", userID, "count", len(queue.pending))
				queue.pending = nil
				queue.running = false
				queue.mutex.Unlock()
				return
			}
			next := queue.pending[0]
			queue.pending = queue.pending[1:]
			queue.mutex.Unlock()
			bot.safeHandle(work, next)
		}
	}()
}

// Принимает: text — текст сообщения.
// Делает: узнаёт «/start», «/start payload» и «/start@botname».
// Возвращает: true для команды /start.
func isHelpCommand(text string) bool {
	first, _, _ := strings.Cut(strings.TrimSpace(text), " ")
	command, _, _ := strings.Cut(first, "@")
	return command == "/start" || command == "/help"
}

// Принимает: text — исходный текст; limit — максимум символов.
// Делает: режет по символам, а не по байтам, чтобы не порвать кириллицу.
// Возвращает: исходный или укороченный текст.
func truncate(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	return string([]rune(text)[:limit])
}
