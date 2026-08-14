package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// RunMigrations runs all pending migrations using goose
func RunMigrations(ctx context.Context, dsn string) error {
	// Create a sql.DB connection for goose
	sqlDB := stdlib.OpenDB(*mustParseConfig(dsn))
	defer sqlDB.Close()

	// Set base FS to our embedded migrations
	goose.SetBaseFS(migrationFS)

	// Run migrations from the embedded FS
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// RunMigrationsWithDir runs migrations from a specific directory in the embedded FS
func RunMigrationsWithDir(ctx context.Context, dsn string, dir string) error {
	sqlDB := stdlib.OpenDB(*mustParseConfig(dsn))
	defer sqlDB.Close()

	goose.SetBaseFS(migrationFS)

	if err := goose.Up(sqlDB, dir); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// mustParseConfig parses DSN into config, panics on error
func mustParseConfig(dsn string) *sql.DB {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		panic(err)
	}
	return db
}

// Alternative migration runner that doesn't use goose
func RunMigrationsManual(ctx context.Context, dsn string) error {
	pool, err := New(ctx, dsn, 5)
	if err != nil {
		return fmt.Errorf("create pool for migrations: %w", err)
	}
	defer pool.Close()

	conn, err := pool.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection: %w", err)
	}
	defer conn.Release()

	// Create goose version table if not exists
	_, err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS goose_db_version (
			id serial PRIMARY KEY,
			version_id bigint NOT NULL UNIQUE,
			is_applied boolean NOT NULL,
			tstamp timestamp DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("create goose table: %w", err)
	}

	// Check current version
	var currentVersion int64
	row := conn.QueryRow(ctx, "SELECT COALESCE(MAX(version_id), 0) FROM goose_db_version WHERE is_applied = true")
	if err := row.Scan(&currentVersion); err != nil {
		return fmt.Errorf("get current version: %w", err)
	}

	// Get list of migration files
	var migrations []string
	err = fs.WalkDir(migrationFS, "migrations", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".up.sql") {
			return nil
		}
		migrations = append(migrations, path)
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk migrations: %w", err)
	}

	sort.Strings(migrations)

	// Apply pending migrations
	for _, migrationPath := range migrations {
		// Extract version number from filename (e.g., 000001_init.up.sql -> 1)
		base := filepath.Base(migrationPath)
		parts := strings.SplitN(base, "_", 2)
		if len(parts) < 2 {
			continue
		}
		
		var version int64
		fmt.Sscanf(parts[0], "%d", &version)

		if version <= currentVersion {
			continue
		}

		// Read migration content
		content, err := migrationFS.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", migrationPath, err)
		}

		// Execute migration in transaction
		tx, err := conn.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin transaction: %w", err)
		}

		// Split by statements and execute
		statements := strings.Split(string(content), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" || strings.HasPrefix(stmt, "--") {
				continue
			}
			if _, err := tx.Exec(ctx, stmt); err != nil {
				tx.Rollback(ctx)
				return fmt.Errorf("execute statement in %s: %w", migrationPath, err)
			}
		}

		// Record migration
		_, err = tx.Exec(ctx, 
			"INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, true)",
			version)
		if err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("record migration: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit migration: %w", err)
		}

		currentVersion = version
	}

	return nil
}
