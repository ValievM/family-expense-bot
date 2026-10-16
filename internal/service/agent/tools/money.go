package tools

import (
	"fmt"
	"math"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// Верхняя граница суммы в рублях защищает от переполнения при переводе в копейки.
const maxRubles = 1_000_000_000

// Принимает: rubles — сумма в рублях от модели, возможно дробная (350 или 350.5).
// Делает: проверяет границы и переводит рубли в копейки с округлением до целой копейки.
// Возвращает: сумму в копейках либо domain.ErrInvalidAmount для NaN, бесконечности, нуля, отрицательных и слишком больших значений.
func rublesToKopecks(rubles float64) (int64, error) {
	if math.IsNaN(rubles) || math.IsInf(rubles, 0) || rubles <= 0 || rubles > maxRubles {
		return 0, domain.ErrInvalidAmount
	}
	kopecks := int64(math.Round(rubles * 100))
	if kopecks <= 0 {
		return 0, domain.ErrInvalidAmount
	}
	return kopecks, nil
}

// Принимает: kopecks — неотрицательная сумма в копейках.
// Делает: форматирует сумму как рубли с двумя знаками после точки, чтобы модель не считала сама.
// Возвращает: строку вида "350.00".
func formatKopecks(kopecks int64) string {
	return fmt.Sprintf("%d.%02d", kopecks/100, kopecks%100)
}
