package telegram_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/ValievM/family-expense-bot/internal/adapters/telegram"
)

type rememberedCall struct {
	userID         int64
	name, username string
}

type fakeDirectory struct {
	mutex sync.Mutex
	calls []rememberedCall
	err   error
}

func (directory *fakeDirectory) Remember(_ context.Context, userID int64, name, username string) error {
	directory.mutex.Lock()
	defer directory.mutex.Unlock()
	directory.calls = append(directory.calls, rememberedCall{userID, name, username})
	return directory.err
}

func (directory *fakeDirectory) snapshot() []rememberedCall {
	directory.mutex.Lock()
	defer directory.mutex.Unlock()
	return append([]rememberedCall(nil), directory.calls...)
}

func profileUpdate(userID int64, chatType, text string, user telegram.User) telegram.Update {
	user.ID = userID
	return telegram.Update{UpdateID: 1, Message: &telegram.Message{
		From: &user, Chat: telegram.Chat{ID: userID, Type: chatType}, Text: text,
	}}
}

func runMemberBot(t *testing.T, update telegram.Update, directory *fakeDirectory, allowed allowList) sentMessage {
	t.Helper()
	gateway := newFakeGateway([]telegram.Update{update})
	options := []telegram.BotOption{
		telegram.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))), telegram.WithRetryDelay(time.Millisecond),
	}
	if directory != nil {
		options = append(options, telegram.WithMemberDirectory(directory))
	}
	responder := responderFunc(func(context.Context, int64, string) (string, error) { return "ok", nil })
	bot := telegram.NewBot(gateway, responder, allowed, options...)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- bot.Run(ctx) }()
	reply := nextSent(t, gateway)
	cancel()
	<-done
	return reply
}

// Проверяет: имя и username запоминаются на обычном сообщении и на /start, имя обрезается по краям.
func TestBotRemembersMemberProfile(t *testing.T) {
	tests := []struct {
		name string
		text string
		user telegram.User
		want rememberedCall
	}{
		{"message", "кофе 350", telegram.User{FirstName: "Анна", LastName: "Иванова", Username: "anna"},
			rememberedCall{42, "Анна Иванова", "anna"}},
		{"start", "/start", telegram.User{FirstName: "Анна", Username: "anna"}, rememberedCall{42, "Анна", "anna"}},
		{"trims name", "кофе 350", telegram.User{LastName: "Иванова"}, rememberedCall{42, "Иванова", ""}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := &fakeDirectory{}
			runMemberBot(t, profileUpdate(42, "private", test.text, test.user), directory, allowList{42: true})

			calls := directory.snapshot()
			if len(calls) != 1 || calls[0] != test.want {
				t.Errorf("Remember calls = %+v, want [%+v]", calls, test.want)
			}
		})
	}
}

// Проверяет: запрещённые пользователи и групповые чаты в справочник не попадают.
func TestBotDoesNotRememberDeniedOrGroup(t *testing.T) {
	tests := []struct {
		name     string
		chatType string
		allowed  allowList
	}{
		{"denied user", "private", allowList{}},
		{"group chat", "group", allowList{42: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := &fakeDirectory{}
			runMemberBot(t, profileUpdate(42, test.chatType, "кофе 350", telegram.User{FirstName: "Анна"}), directory, test.allowed)

			if calls := directory.snapshot(); len(calls) != 0 {
				t.Errorf("Remember calls = %+v, want none", calls)
			}
		})
	}
}

// Проверяет: ошибка справочника не мешает ответу, а без справочника бот работает как раньше.
func TestBotRepliesWhenDirectoryFailsOrMissing(t *testing.T) {
	update := profileUpdate(42, "private", "кофе 350", telegram.User{FirstName: "Анна"})

	failing := &fakeDirectory{err: errors.New("database is down")}
	if reply := runMemberBot(t, update, failing, allowList{42: true}); reply.text != "ok" {
		t.Errorf("reply with failing directory = %q, want ok", reply.text)
	}
	if reply := runMemberBot(t, update, nil, allowList{42: true}); reply.text != "ok" {
		t.Errorf("reply without directory = %q, want ok", reply.text)
	}
}
