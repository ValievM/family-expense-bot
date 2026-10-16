package domain

// UserID — неизменяемое значение «владелец расхода», Telegram ID члена семьи.
type UserID struct {
	value int64
}

// Принимает: value — Telegram ID.
// Делает: создаёт идентификатор и проверяет, что он положителен.
// Возвращает: UserID либо ErrInvalidUser.
func NewUserID(value int64) (UserID, error) {
	if value <= 0 {
		return UserID{}, ErrInvalidUser
	}
	return UserID{value: value}, nil
}

// Принимает: ничего.
// Делает: читает числовое значение.
// Возвращает: Telegram ID.
func (id UserID) Int64() int64 {
	return id.value
}

// Принимает: ничего.
// Делает: проверяет, что идентификатор не задан (нулевое значение).
// Возвращает: true для нулевого UserID.
func (id UserID) IsZero() bool {
	return id.value == 0
}
