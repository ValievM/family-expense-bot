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

// Таймауты, лимиты и тексты ответов бота.
const (
	pollTimeoutSeconds = 30
	retryDelay         = 3 * time.Second
	handleTimeout      = 90 * time.Second
	maxMessageRunes    = 4096

	greeting       = "Привет! Пишите траты обычным текстом: «кофе 350», «такси 1200 вчера». Можно спросить «сколько потратил на еду в этом месяце», изменить «последнее было не 350, а 450» или удалить «удали такси»."
	failureReply   = "Что-то пошло не так, попробуйте ещё раз чуть позже."
	deniedReply    = "У вас нет доступа к этому боту."
	privateOnlyMsg = "Я работаю только в личных сообщениях."
)

type Gateway interface {
	GetUpdates(ctx context.Context, offset int64, timeoutSeconds int) ([]Update, error)
	SendMessage(ctx context.Context, chatID int64, text string) error
	SendTyping(ctx context.Context, chatID int64) error
}

// Bot читает обновления, ограничивает доступ и отвечает; сообщения одного пользователя обрабатываются строго в порядке получения.
type Bot struct {
	gateway   Gateway
	responder ports.Responder
	access    ports.AccessPolicy
	logger    *slog.Logger
	retry     time.Duration
	queues    sync.Map // userID -> *userQueue
}

// userQueue хранит сообщения одного пользователя в порядке получения; одновременно их разбирает не больше одной горутины.
type userQueue struct {
	mutex   sync.Mutex
	pending []Message
	running bool
}

// BotOption настраивает Bot.
type BotOption func(*Bot)

// Принимает: logger — логгер бота.
// Делает: заменяет логгер.
// Возвращает: BotOption.
func WithLogger(logger *slog.Logger) BotOption {
	return func(bot *Bot) { bot.logger = logger }
}

// Принимает: delay — пауза после ошибки получения обновлений.
// Делает: меняет паузу; тесты ставят её почти нулевой.
// Возвращает: BotOption.
func WithRetryDelay(delay time.Duration) BotOption {
	return func(bot *Bot) { bot.retry = delay }
}

// Принимает: gateway — доступ к Telegram; responder — обработчик текста; access — политика доступа; options — настройки.
// Делает: собирает бота.
// Возвращает: готового Bot.
func NewBot(gateway Gateway, responder ports.Responder, access ports.AccessPolicy, options ...BotOption) *Bot {
	bot := &Bot{gateway: gateway, responder: responder, access: access, logger: slog.Default(), retry: retryDelay}
	for _, option := range options {
		option(bot)
	}
	return bot
}

// Принимает: ctx — контекст, отмена которого останавливает бота.
// Делает: в цикле получает обновления и обрабатывает каждое сообщение в своей горутине; при остановке дожидается текущих обработчиков.
// Возвращает: nil после остановки по контексту.
func (bot *Bot) Run(ctx context.Context) error {
	var inFlight sync.WaitGroup
	defer inFlight.Wait()

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
			bot.dispatch(ctx, &inFlight, *update.Message)
		}
	}
	return nil
}

// Принимает: ctx — контекст остановки; delay — сколько ждать.
// Делает: ждёт паузу, но прерывается при остановке бота.
// Возвращает: ничего.
func (bot *Bot) wait(ctx context.Context, delay time.Duration) {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

// Принимает: ctx — контекст бота; message — входящее сообщение.
// Делает: отбрасывает чужие типы сообщений, проверяет доступ, вызывает обработчик и отправляет ответ.
// Возвращает: ничего; сбои пишутся в лог, а пользователь получает нейтральный ответ.
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
	if isStartCommand(message.Text) {
		bot.reply(ctx, chatID, greeting)
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

// Принимает: ctx — контекст отправки; chatID — получатель; text — текст ответа.
// Делает: обрезает ответ до лимита Telegram и отправляет его, записывая ошибку отправки в лог.
// Возвращает: ничего.
func (bot *Bot) reply(ctx context.Context, chatID int64, text string) {
	// Ответ нужно доставить даже во время остановки, поэтому отмена бота на отправку не действует.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), requestTimeout)
	defer cancel()
	if err := bot.gateway.SendMessage(ctx, chatID, truncate(text, maxMessageRunes)); err != nil {
		bot.logger.ErrorContext(ctx, "telegram send failed", "chat_id", chatID, "error", err)
	}
}

// Принимает: ctx — контекст бота; inFlight — счётчик работающих обработчиков; message — входящее сообщение.
// Делает: ставит сообщение в очередь его автора и при необходимости запускает горутину, которая разбирает очередь по порядку; разные пользователи обрабатываются параллельно.
// Возвращает: ничего.
func (bot *Bot) dispatch(ctx context.Context, inFlight *sync.WaitGroup, message Message) {
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
			next := queue.pending[0]
			queue.pending = queue.pending[1:]
			queue.mutex.Unlock()
			bot.handle(ctx, next)
		}
	}()
}

// Принимает: text — текст сообщения.
// Делает: узнаёт стартовое сообщение Telegram: «/start» или «/start payload», в том числе «/start@botname».
// Возвращает: true, если это команда /start.
func isStartCommand(text string) bool {
	first, _, _ := strings.Cut(strings.TrimSpace(text), " ")
	command, _, _ := strings.Cut(first, "@")
	return command == "/start"
}

// Принимает: text — исходный текст; limit — максимум символов.
// Делает: обрезает по символам, а не по байтам, чтобы не порвать русские буквы.
// Возвращает: исходный или укороченный текст.
func truncate(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	return string([]rune(text)[:limit])
}
