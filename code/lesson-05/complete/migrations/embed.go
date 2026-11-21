// Package migrations holds the database schema as numbered .sql files.
// They are embedded into the binary, so the program can set up its own tables.
package migrations

import "embed"

// FS holds every *.sql file in this folder, applied in name order.
//
//go:embed *.sql
var FS embed.FS
