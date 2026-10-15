package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ValievM/family-expense-bot/migrations"
	"github.com/pressly/goose/v3"
)

// Принимает: ctx — контекст выполнения; database — подключение к PostgreSQL.
// Делает: применяет все ещё не выполненные встроенные миграции через goose.
// Возвращает: nil при успехе или ошибку настройки либо применения миграций.
func Migrate(ctx context.Context, database *sql.DB) error {
	provider, err := goose.NewProvider(goose.DialectPostgres, database, migrations.Files)
	if err != nil {
		return fmt.Errorf("configure PostgreSQL migrations: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply PostgreSQL migrations: %w", err)
	}
	return nil
}
