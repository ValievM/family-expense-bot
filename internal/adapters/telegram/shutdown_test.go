package telegram_test

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/adapters/telegram"
	"github.com/ValievM/family-expense-bot/internal/ports"
)

func newShutdownBot(gateway *fakeGateway, responder ports.Responder, drain time.Duration) *telegram.Bot {
	return telegram.NewBot(gateway, responder, allowList{42: true},
		telegram.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))),
		telegram.WithRetryDelay(time.Millisecond), telegram.WithDrainTimeout(drain))
}

// Проверяет: сообщение, которое уже обрабатывается в момент остановки, доводится до ответа пользователю, а Run ждёт его.
func TestBotFinishesInFlightMessageOnShutdown(t *testing.T) {
	gateway := newFakeGateway([]telegram.Update{textUpdate(1, 42, "private", "кофе 350")})
	started, release := make(chan struct{}), make(chan struct{})
	bot := newShutdownBot(gateway, responderFunc(func(ctx context.Context, _ int64, _ string) (string, error) {
		close(started)
		<-release
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "записал", nil
	}), 5*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- bot.Run(ctx) }()

	<-started
	cancel()
	select {
	case <-done:
		t.Fatal("Run returned while a message was still being handled")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)

	if reply := nextSent(t, gateway); reply.text != "записал" {
		t.Errorf("reply = %q, want the answer of the in-flight message", reply.text)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after the in-flight message was answered")
	}
}

// Проверяет: сообщения, которые ещё стоят в очереди пользователя, после остановки не обрабатываются.
func TestBotDropsQueuedMessagesOnShutdown(t *testing.T) {
	gateway := newFakeGateway([]telegram.Update{
		textUpdate(1, 42, "private", "первое"),
		textUpdate(2, 42, "private", "второе"),
	})
	var calls atomic.Int32
	started, release := make(chan struct{}), make(chan struct{})
	bot := newShutdownBot(gateway, responderFunc(func(context.Context, int64, string) (string, error) {
		if calls.Add(1) == 1 {
			close(started)
			<-release
		}
		return "ok", nil
	}), 5*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- bot.Run(ctx) }()

	<-started
	time.Sleep(50 * time.Millisecond) // второе сообщение уже в очереди
	cancel()
	close(release)

	if reply := nextSent(t, gateway); reply.text != "ok" {
		t.Errorf("first reply = %q", reply.text)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return")
	}
	if calls.Load() != 1 {
		t.Errorf("responder calls = %d, want 1: the queued message must be dropped", calls.Load())
	}
}

// Проверяет: зависший обработчик не держит остановку дольше срока drain, его контекст отменяется.
func TestBotShutdownIsBoundedByDrainTimeout(t *testing.T) {
	gateway := newFakeGateway([]telegram.Update{textUpdate(1, 42, "private", "повисло")})
	started := make(chan struct{})
	cancelled := make(chan struct{})
	bot := newShutdownBot(gateway, responderFunc(func(ctx context.Context, _ int64, _ string) (string, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return "", ctx.Err()
	}), 100*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- bot.Run(ctx) }()

	<-started
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop within the drain timeout")
	}
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Error("the stuck handler was not cancelled after the deadline")
	}
}
