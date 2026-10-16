package domain

import (
	"fmt"
	"math"
)

const (
	// maxRubles — верхняя граница суммы: защита от переполнения при переводе в копейки.
	maxRubles  = 1_000_000_000
	maxKopecks = maxRubles * 100
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

// Принимает: rubles — сумма в рублях, возможно дробная (99.9).
// Делает: переводит в копейки с округлением до целой копейки.
// Возвращает: Money либо ErrInvalidAmount (NaN, Inf, ноль, отрицательные, слишком большие).
func MoneyFromRubles(rubles float64) (Money, error) {
	if math.IsNaN(rubles) || math.IsInf(rubles, 0) || rubles <= 0 || rubles > maxRubles {
		return Money{}, ErrInvalidAmount
	}
	kopecks := int64(math.Round(rubles * 100))
	if kopecks <= 0 {
		return Money{}, ErrInvalidAmount
	}
	return Money{kopecks: kopecks}, nil
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
