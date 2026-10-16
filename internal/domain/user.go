package domain

type UserID struct {
	value int64
}

// Принимает: value — Telegram ID.
// Делает: проверяет, что ID положителен.
// Возвращает: UserID либо ErrInvalidUser.
func NewUserID(value int64) (UserID, error) {
	if value <= 0 {
		return UserID{}, ErrInvalidUser
	}
	return UserID{value: value}, nil
}

func (id UserID) Int64() int64 {
	return id.value
}

func (id UserID) IsZero() bool {
	return id.value == 0
}
