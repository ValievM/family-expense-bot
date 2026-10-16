package domain

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// maxRubles — верхняя граница суммы: защита от переполнения при переводе в копейки.
	maxRubles      = 1_000_000_000
	maxKopecks     = maxRubles * 100
	maxWholeDigits = 10 // длина maxRubles, чтобы целая часть не переполнила int64
)

type Money struct {
	kopecks int64
}

// Принимает: kopecks — сумма в копейках.
// Делает: проверяет границы суммы (от нуля до 1 млрд рублей).
// Возвращает: Money либо ErrInvalidAmount.
func NewMoney(kopecks int64) (Money, error) {
	if kopecks < 0 || kopecks > maxKopecks {
		return Money{}, ErrInvalidAmount
	}
	return Money{kopecks: kopecks}, nil
}

// Принимает: text — положительная сумма в рублях десятичной записью: "350", "99.9", "99,90".
// Делает: разбирает рубли и копейки целыми числами, без float; больше двух знаков после запятой не принимается.
// Возвращает: Money либо ErrInvalidAmount (пусто, знак, экспонента, ноль, слишком много знаков, слишком большая сумма).
func ParseRubles(text string) (Money, error) {
	whole, fraction, hasFraction := strings.Cut(strings.ReplaceAll(strings.TrimSpace(text), ",", "."), ".")
	if !isDigits(whole) || len(whole) > maxWholeDigits {
		return Money{}, ErrInvalidAmount
	}
	if !hasFraction {
		fraction = "00"
	}
	if !isDigits(fraction) || len(fraction) > 2 {
		return Money{}, ErrInvalidAmount
	}
	fraction += strings.Repeat("0", 2-len(fraction))

	rubles, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return Money{}, ErrInvalidAmount
	}
	kopecks, err := strconv.ParseInt(fraction, 10, 64)
	if err != nil {
		return Money{}, ErrInvalidAmount
	}
	money, err := NewMoney(rubles*100 + kopecks)
	if err != nil || !money.IsPositive() {
		return Money{}, ErrInvalidAmount
	}
	return money, nil
}

func isDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, char := range text {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func (money Money) Kopecks() int64 {
	return money.kopecks
}

func (money Money) IsPositive() bool {
	return money.kopecks > 0
}

func (money Money) Add(other Money) Money {
	return Money{kopecks: money.kopecks + other.kopecks}
}

func (money Money) String() string {
	return fmt.Sprintf("%d.%02d", money.kopecks/100, money.kopecks%100)
}
