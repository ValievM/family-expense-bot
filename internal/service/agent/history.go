package agent

import (
	"sync"

	"github.com/ValievM/family-expense-bot/internal/ports"
)

type memoryHistory struct {
	mutex  sync.Mutex
	limit  int
	byUser map[int64][]ports.Message
}

// Принимает: limit — максимум сообщений на пользователя; не больше нуля = 8.
// Делает: создаёт потокобезопасную историю в памяти.
// Возвращает: ports.History.
func NewMemoryHistory(limit int) ports.History {
	if limit <= 0 {
		limit = 8
	}
	return &memoryHistory{limit: limit, byUser: make(map[int64][]ports.Message)}
}

// Принимает: userID.
// Делает: под блокировкой копирует сообщения, чтобы не отдавать внутреннее состояние.
// Возвращает: копию истории пользователя.
func (history *memoryHistory) Recent(userID int64) []ports.Message {
	history.mutex.Lock()
	defer history.mutex.Unlock()
	return append([]ports.Message(nil), history.byUser[userID]...)
}

// Принимает: userID; messages — новые сообщения по порядку.
// Делает: добавляет, обрезает до лимита, следит, чтобы история начиналась с сообщения пользователя.
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
