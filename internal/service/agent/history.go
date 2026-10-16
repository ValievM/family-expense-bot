package agent

import (
	"sync"

	"github.com/ValievM/family-expense-bot/internal/ports"
)

// memoryHistory — потокобезопасная реализация ports.History в памяти процесса.
type memoryHistory struct {
	mutex  sync.Mutex
	limit  int
	byUser map[int64][]ports.Message
}

// Принимает: limit — максимум сообщений на пользователя; значение не больше нуля заменяется на 8.
// Делает: создаёт пустое хранилище истории в памяти.
// Возвращает: ports.History, безопасную для вызовов из нескольких горутин.
func NewMemoryHistory(limit int) ports.History {
	if limit <= 0 {
		limit = 8
	}
	return &memoryHistory{limit: limit, byUser: make(map[int64][]ports.Message)}
}

// Принимает: userID — Telegram ID владельца диалога.
// Делает: под блокировкой копирует сохранённые сообщения, чтобы вызывающий не менял внутреннее состояние.
// Возвращает: копию истории пользователя.
func (history *memoryHistory) Recent(userID int64) []ports.Message {
	history.mutex.Lock()
	defer history.mutex.Unlock()
	return append([]ports.Message(nil), history.byUser[userID]...)
}

// Принимает: userID — Telegram ID владельца диалога; messages — новые сообщения по порядку.
// Делает: добавляет сообщения, обрезает историю до лимита и следит, чтобы она начиналась с сообщения пользователя.
// Возвращает: ничего.
func (history *memoryHistory) Remember(userID int64, messages ...ports.Message) {
	history.mutex.Lock()
	defer history.mutex.Unlock()

	previous := history.byUser[userID]
	all := make([]ports.Message, 0, len(previous)+len(messages))
	all = append(all, previous...)
	all = append(all, messages...)
	if len(all) > history.limit {
		all = all[len(all)-history.limit:]
	}
	for len(all) > 0 && all[0].Role != ports.RoleUser {
		all = all[1:]
	}
	history.byUser[userID] = append([]ports.Message(nil), all...)
}
