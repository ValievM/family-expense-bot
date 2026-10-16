package telegram_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/adapters/telegram"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

type sentMessage struct {
	chatID int64
	text   string
}

type fakeGateway struct {
	mutex   sync.Mutex
	batches [][]telegram.Update
	sent    chan sentMessage
	offsets []int64
}

func newFakeGateway(batches ...[]telegram.Update) *fakeGateway {
	return &fakeGateway{batches: batches, sent: make(chan sentMessage, 16)}
}

func (gateway *fakeGateway) GetUpdates(ctx context.Context, offset int64, _ int) ([]telegram.Update, error) {
	gateway.mutex.Lock()
	gateway.offsets = append(gateway.offsets, offset)
	if len(gateway.batches) > 0 {
		batch := gateway.batches[0]
		gateway.batches = gateway.batches[1:]
		gateway.mutex.Unlock()
		return batch, nil
	}
	gateway.mutex.Unlock()
	<-ctx.Done()
	return nil, ctx.Err()
}

func (gateway *fakeGateway) SendMessage(_ context.Context, chatID int64, text string) error {
	gateway.sent <- sentMessage{chatID: chatID, text: text}
	return nil
}

func (gateway *fakeGateway) SendTyping(context.Context, int64) error { return nil }

type responderFunc func(ctx context.Context, userID int64, text string) (string, error)

func (fn responderFunc) Handle(ctx context.Context, userID int64, text string) (string, error) {
	return fn(ctx, userID, text)
}

type allowList map[int64]bool

func (list allowList) IsUserAllowed(userID int64) bool { return list[userID] }

func textUpdate(id, userID int64, chatType, text string) telegram.Update {
	return telegram.Update{UpdateID: id, Message: &telegram.Message{
		From: &telegram.User{ID: userID}, Chat: telegram.Chat{ID: userID, Type: chatType}, Text: text,
	}}
}

func runBot(t *testing.T, gateway *fakeGateway, responder ports.Responder, access ports.AccessPolicy) (stop func()) {
	t.Helper()
	bot := telegram.NewBot(gateway, responder, access,
		telegram.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))), telegram.WithRetryDelay(time.Millisecond))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- bot.Run(ctx) }()
	var once sync.Once
	return func() {
		once.Do(func() {
			cancel()
			if err := <-done; err != nil {
				t.Errorf("Run() error = %v", err)
			}
		})
	}
}

func nextSent(t *testing.T, gateway *fakeGateway) sentMessage {
	t.Helper()
	select {
	case message := <-gateway.sent:
		return message
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for a bot reply")
		return sentMessage{}
	}
}

// Проверяет: текст разрешённого пользователя доходит до обработчика с доверенным ID автора, ответ уходит в его чат, offset растёт.
func TestBotRepliesWithTrustedAuthorID(t *testing.T) {
	gateway := newFakeGateway([]telegram.Update{textUpdate(10, 42, "private", "кофе 350")})
	var gotUser atomic.Int64
	stop := runBot(t, gateway, responderFunc(func(_ context.Context, userID int64, text string) (string, error) {
		gotUser.Store(userID)
		return "записал: " + text, nil
	}), allowList{42: true})
	defer stop()

	reply := nextSent(t, gateway)

	if reply.chatID != 42 || reply.text != "записал: кофе 350" || gotUser.Load() != 42 {
		t.Errorf("reply = %+v, handler user = %d", reply, gotUser.Load())
	}
	stop()
	gateway.mutex.Lock()
	defer gateway.mutex.Unlock()
	if len(gateway.offsets) < 2 || gateway.offsets[1] != 11 {
		t.Errorf("offsets = %v, want the second poll to ask from 11", gateway.offsets)
	}
}

// Проверяет: не разрешённые, групповые, от ботов и без текста сообщения не доходят до обработчика.
func TestBotFiltersMessages(t *testing.T) {
	var calls atomic.Int32
	updates := []telegram.Update{
		textUpdate(1, 99, "private", "привет"),
		textUpdate(2, 42, "group", "кофе 350"),
		{UpdateID: 3, Message: &telegram.Message{From: &telegram.User{ID: 42, IsBot: true}, Chat: telegram.Chat{ID: 42, Type: "private"}, Text: "bot"}},
		{UpdateID: 4, Message: &telegram.Message{From: &telegram.User{ID: 42}, Chat: telegram.Chat{ID: 42, Type: "private"}}},
		{UpdateID: 5},
	}
	gateway := newFakeGateway(updates)
	stop := runBot(t, gateway, responderFunc(func(context.Context, int64, string) (string, error) {
		calls.Add(1)
		return "x", nil
	}), allowList{42: true})

	// Сообщения обрабатываются параллельно, поэтому порядок ответов не гарантирован.
	replies := nextSent(t, gateway).text + "|" + nextSent(t, gateway).text
	stop()

	if !strings.Contains(replies, "нет доступа") || !strings.Contains(replies, "личных") {
		t.Errorf("replies = %q, want an access denial and a private-chat notice", replies)
	}
	if calls.Load() != 0 {
		t.Errorf("handler called %d times, want 0", calls.Load())
	}
}

// Проверяет: ошибка обработчика не раскрывается пользователю, /start отвечает приветствием без обработчика.
func TestBotHidesErrorsAndGreetsOnStart(t *testing.T) {
	var calls atomic.Int32
	gateway := newFakeGateway([]telegram.Update{
		textUpdate(1, 42, "private", "/start@family_bot"),
		textUpdate(2, 42, "private", "кофе"),
	})
	stop := runBot(t, gateway, responderFunc(func(context.Context, int64, string) (string, error) {
		calls.Add(1)
		return "", errors.New("secret database failure")
	}), allowList{42: true})

	// /start отвечает сразу, а второе сообщение идёт через очередь, поэтому порядок ответов не гарантирован.
	replies := nextSent(t, gateway).text + "|" + nextSent(t, gateway).text
	stop()

	if !strings.Contains(replies, "Привет") {
		t.Errorf("replies = %q, want a greeting", replies)
	}
	if strings.Contains(replies, "secret") || !strings.Contains(replies, "Что-то пошло не так") {
		t.Errorf("replies = %q, want a neutral failure text without internals", replies)
	}
	if calls.Load() != 1 {
		t.Errorf("handler calls = %d, want 1 (the /start must not reach it)", calls.Load())
	}
}

// Проверяет: два сообщения одного пользователя не выполняются одновременно.
func TestBotSerializesMessagesOfOneUser(t *testing.T) {
	var running, maxRunning atomic.Int32
	gateway := newFakeGateway([]telegram.Update{
		textUpdate(1, 42, "private", "a"),
		textUpdate(2, 42, "private", "b"),
		textUpdate(3, 42, "private", "c"),
	})
	stop := runBot(t, gateway, responderFunc(func(context.Context, int64, string) (string, error) {
		current := running.Add(1)
		for {
			seen := maxRunning.Load()
			if current <= seen || maxRunning.CompareAndSwap(seen, current) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		running.Add(-1)
		return "ok", nil
	}), allowList{42: true})

	for range 3 {
		nextSent(t, gateway)
	}
	stop()

	if maxRunning.Load() != 1 {
		t.Errorf("max concurrent handlers for one user = %d, want 1", maxRunning.Load())
	}
}

// Проверяет: очень длинный ответ обрезается до лимита Telegram по символам.
func TestBotTruncatesLongReplies(t *testing.T) {
	gateway := newFakeGateway([]telegram.Update{textUpdate(1, 42, "private", "много")})
	stop := runBot(t, gateway, responderFunc(func(context.Context, int64, string) (string, error) {
		return strings.Repeat("ё", 5000), nil
	}), allowList{42: true})

	reply := nextSent(t, gateway)
	stop()

	if got := len([]rune(reply.text)); got != 4096 {
		t.Errorf("reply length = %d runes, want 4096", got)
	}
}

// Проверяет: сообщения одного пользователя обрабатываются в порядке получения.
func TestBotKeepsMessageOrderPerUser(t *testing.T) {
	updates := make([]telegram.Update, 0, 20)
	for id := int64(1); id <= 20; id++ {
		updates = append(updates, textUpdate(id, 42, "private", strings.Repeat("x", int(id))))
	}
	gateway := newFakeGateway(updates)
	stop := runBot(t, gateway, responderFunc(func(_ context.Context, _ int64, text string) (string, error) {
		return text, nil
	}), allowList{42: true})

	for want := 1; want <= 20; want++ {
		if got := len(nextSent(t, gateway).text); got != want {
			t.Fatalf("reply #%d has length %d, want the messages in arrival order", want, got)
		}
	}
	stop()
}
