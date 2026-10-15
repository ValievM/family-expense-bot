// Package migrations — встроенные SQL-миграции PostgreSQL для goose.
package migrations

import "embed"

// Files — встроенные SQL-миграции для адаптера PostgreSQL.
//
//go:embed *.sql
var Files embed.FS
