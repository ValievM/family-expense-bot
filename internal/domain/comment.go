package domain

import (
	"strings"
	"unicode/utf8"
)

// maxCommentRunes — максимальная длина комментария в символах.
const maxCommentRunes = 500

// Comment — неизменяемое необязательное пояснение к расходу; пробелы по краям отбрасываются.
type Comment struct {
	text string
}

// Принимает: text — произвольный текст.
// Делает: обрезает пробелы по краям и проверяет длину.
// Возвращает: Comment (пустой допустим) либо ErrCommentTooLong.
func NewComment(text string) (Comment, error) {
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > maxCommentRunes {
		return Comment{}, ErrCommentTooLong
	}
	return Comment{text: text}, nil
}

// Принимает: ничего.
// Делает: читает текст.
// Возвращает: строку комментария.
func (comment Comment) String() string {
	return comment.text
}
