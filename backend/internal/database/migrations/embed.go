package migrations

import "embed"

// FS contains the database migration files.
//
//go:embed *.sql
var FS embed.FS