package domain

import (
	"fmt"
	"math"
)

const (
	// maxRubles — верхняя граница суммы: защита от переполнения при переводе в копейки.
	maxRubles = 1_000_000_000
	// maxKopecks — та же граница в копейках.
	maxKopecks = maxRubles * 100
)

// Money — неизменяемое значение «сумма в рублях», хранится целым числом копеек, чтобы не терять точность.
type Money struct {
	kopecks int64
}

// Принимает: kopecks — сумма в копейках.
// Делает: создаёт сумму и проверяет, что она в допустимых границах (от нуля до 1 млрд рублей).
// Возвращает: Money либо ErrInvalidAmount для отрицательных и слишком больших значений.
func NewMoney(kopecks int64) (Money, error) {
	if kopecks < 0 || kopecks > maxKopecks {
		return Money{}, ErrInvalidAmount
	}
	return Money{kopecks: kopecks}, nil
}

// Принимает: rubles — положительная сумма в рублях, возможно дробная (350 или 99.9).
// Делает: переводит рубли в копейки с округлением до целой копейки.
// Возвращает: Money либо ErrInvalidAmount для NaN, бесконечности, нуля, отрицательных и слишком больших значений.
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

// Принимает: ничего.
// Делает: читает сумму в копейках.
// Возвращает: целое число копеек.
func (money Money) Kopecks() int64 {
	return money.kopecks
}

// Принимает: ничего.
// Делает: проверяет, что сумма больше нуля.
// Возвращает: true для положительной суммы.
func (money Money) IsPositive() bool {
	return money.kopecks > 0
}

// Принимает: other — слагаемое.
// Делает: складывает суммы, не изменяя исходные значения.
// Возвращает: новую сумму.
func (money Money) Add(other Money) Money {
	return Money{kopecks: money.kopecks + other.kopecks}
}

// Принимает: ничего.
// Делает: форматирует сумму как рубли с двумя знаками после точки.
// Возвращает: строку вида "350.00".
func (money Money) String() string {
	return fmt.Sprintf("%d.%02d", money.kopecks/100, money.kopecks%100)
}
