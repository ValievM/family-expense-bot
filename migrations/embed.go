// Package migrations embeds PostgreSQL schema changes for goose.
package migrations

import "embed"

// Files contains the SQL migrations used by the PostgreSQL adapter.
//
//go:embed *.sql
var Files embed.FS
