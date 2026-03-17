package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrate applies all *.sql files in the given migrationsDir in
// lexicographic order. It is idempotent: it tracks applied migrations
// in a schema_migrations table and skips already-applied files.
//
// Usage in main.go:
//
//	pool, _ := pgxpool.New(ctx, os.Getenv("POSTGRES_URL"))
//	if err := db.Migrate(ctx, pool, "./migrations"); err != nil {
//	    log.Fatal("migration failed:", err)
//	}
func Migrate(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	// Ensure the tracking table exists.
	if err := ensureMigrationsTable(ctx, pool); err != nil {
		return fmt.Errorf("db.Migrate: create schema_migrations: %w", err)
	}

	// Read migration files.
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("db.Migrate: read dir %q: %w", migrationsDir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files) // lexicographic = chronological for 001_, 002_, …

	for _, file := range files {
		applied, err := isApplied(ctx, pool, file)
		if err != nil {
			return fmt.Errorf("db.Migrate: check %q: %w", file, err)
		}
		if applied {
			fmt.Printf("db.Migrate: %s — already applied, skipping\n", file)
			continue
		}

		sql, err := os.ReadFile(filepath.Join(migrationsDir, file))
		if err != nil {
			return fmt.Errorf("db.Migrate: read %q: %w", file, err)
		}

		fmt.Printf("db.Migrate: applying %s …\n", file)
		start := time.Now()

		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("db.Migrate: apply %q: %w", file, err)
		}

		if err := markApplied(ctx, pool, file); err != nil {
			return fmt.Errorf("db.Migrate: record %q: %w", file, err)
		}

		fmt.Printf("db.Migrate: %s — applied in %s\n", file, time.Since(start).Round(time.Millisecond))
	}

	return nil
}

// ── private helpers ───────────────────────────────────────────

func ensureMigrationsTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename   TEXT        PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func isApplied(ctx context.Context, pool *pgxpool.Pool, filename string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)`,
		filename,
	).Scan(&exists)
	return exists, err
}

func markApplied(ctx context.Context, pool *pgxpool.Pool, filename string) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING`,
		filename,
	)
	return err
}
