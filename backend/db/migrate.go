package db

import (
	"context"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schema string

// RunMigrations applies the embedded schema.sql to the database.
// Uses go:embed so the path is resolved at compile time — works with
// `go run`, `go build`, and any working directory.
func (d *DB) RunMigrations() error {
	if _, err := d.pool.Exec(context.Background(), schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}
