// Package telegram принимает сообщения через long polling Bot API и передаёт их обработчику, не зная про учёт трат.
package telegram

// Update — одно событие Bot API; нас интересуют только обычные сообщения.
type Update struct {
	UpdateID int64    `json:"update_id"`
	Message  *Message `json:"message"`
}

// Message — входящее сообщение пользователя.
type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

// User — автор сообщения; его ID считается доверенным владельцем данных.
type User struct {
	ID    int64 `json:"id"`
	IsBot bool  `json:"is_bot"`
}

// Chat — чат, из которого пришло сообщение.
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}
