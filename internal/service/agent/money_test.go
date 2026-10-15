package agent

import (
	"errors"
	"math"
	"testing"

	"github.com/ValievM/family-expense-bot/internal/domain"
)

// Принимает: t — тестовый контекст.
// Делает: проверяет перевод рублей в копейки: округление, границы и недопустимые значения.
// Возвращает: ничего.
func TestRublesToKopecks(t *testing.T) {
	tests := []struct {
		name    string
		rubles  float64
		want    int64
		wantErr bool
	}{
		{name: "whole rubles", rubles: 350, want: 35000},
		{name: "fraction", rubles: 99.9, want: 9990},
		{name: "float noise is rounded", rubles: 0.1 + 0.2, want: 30},
		{name: "one kopeck", rubles: 0.01, want: 1},
		{name: "below one kopeck", rubles: 0.004, wantErr: true},
		{name: "zero", rubles: 0, wantErr: true},
		{name: "negative", rubles: -1, wantErr: true},
		{name: "NaN", rubles: math.NaN(), wantErr: true},
		{name: "infinity", rubles: math.Inf(1), wantErr: true},
		{name: "too large", rubles: 2e9, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := rublesToKopecks(tt.rubles)

			if tt.wantErr {
				if !errors.Is(err, domain.ErrInvalidAmount) {
					t.Errorf("error = %v, want ErrInvalidAmount", err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("rublesToKopecks(%v) = %d, %v; want %d", tt.rubles, got, err, tt.want)
			}
		})
	}
}

// Принимает: t — тестовый контекст.
// Делает: проверяет форматирование копеек в рубли с двумя знаками.
// Возвращает: ничего.
func TestFormatKopecks(t *testing.T) {
	tests := map[int64]string{0: "0.00", 5: "0.05", 35000: "350.00", 9990: "99.90", 123456789: "1234567.89"}

	for kopecks, want := range tests {
		if got := formatKopecks(kopecks); got != want {
			t.Errorf("formatKopecks(%d) = %q, want %q", kopecks, got, want)
		}
	}
}
