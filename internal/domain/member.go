package domain

import (
	"strings"
	"unicode/utf8"
)

const (
	maxMemberNameRunes     = 80
	maxMemberUsernameRunes = 64
)

// Member — член семьи, известный по сообщениям боту: Telegram ID плюс отображаемые данные из профиля.
type Member struct {
	id       UserID
	name     string
	username string
}

// Принимает: id — Telegram ID; name — имя из профиля; username — ник без «@».
// Делает: создаёт участника; пробелы по краям и «@» отбрасываются, слишком длинные значения обрезаются по символам,
// потому что внешние данные профиля не должны ломать работу бота.
// Возвращает: Member либо ErrInvalidUser для нулевого ID.
func NewMember(id UserID, name, username string) (Member, error) {
	if id.IsZero() {
		return Member{}, ErrInvalidUser
	}
	return Member{
		id:       id,
		name:     truncateRunes(strings.TrimSpace(name), maxMemberNameRunes),
		username: truncateRunes(strings.TrimPrefix(strings.TrimSpace(username), "@"), maxMemberUsernameRunes),
	}, nil
}

func (member Member) ID() UserID { return member.id }

func (member Member) Name() string { return member.name }

func (member Member) Username() string { return member.username }

func truncateRunes(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}
	return string([]rune(text)[:limit])
}
