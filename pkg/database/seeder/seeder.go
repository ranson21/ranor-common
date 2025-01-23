// pkg/database/seeder/loader.go
package seeder

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Migration struct {
	Version     int
	Description string
	Up          string
	Down        string
}

type Rows interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close()
}

func NewManager(pool Pool) *Manager {
	return &Manager{pool: pool}
}

type Pool interface {
	Begin(ctx context.Context) (Tx, error)
	Query(ctx context.Context, sql string, args ...interface{}) (Rows, error)
}

type Tx interface {
	Exec(ctx context.Context, sql string, args ...interface{}) error
	QueryRow(ctx context.Context, sql string, args ...interface{}) Row
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Manager struct {
	pool       Pool
	migrations []Migration
	seeds      []Seed
}

type Seed struct {
	Name     string
	Priority int
	Run      func(ctx context.Context, tx Tx) error
}

type Row interface {
	Scan(dest ...interface{}) error
}

type FileLoader struct{}

func NewFileLoader() *FileLoader {
	return &FileLoader{}
}

func (m *Manager) AddMigration(mg Migration) {
	m.migrations = append(m.migrations, mg)
}

func (m *Manager) AddSeed(seed Seed) {
	m.seeds = append(m.seeds, seed)
}

func (m *Manager) RunMigrations(ctx context.Context) error {
	if err := m.createMigrationsTable(ctx); err != nil {
		return err
	}

	sort.Slice(m.migrations, func(i, j int) bool {
		return m.migrations[i].Version < m.migrations[j].Version
	})

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	for _, mg := range m.migrations {
		if !applied[mg.Version] {
			if err := m.runMigration(ctx, mg); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Manager) RunSeeds(ctx context.Context) error {
	if err := m.createSeedsTable(ctx); err != nil {
		return err
	}

	sort.Slice(m.seeds, func(i, j int) bool {
		return m.seeds[i].Priority < m.seeds[j].Priority
	})

	applied, err := m.getAppliedSeeds(ctx)
	if err != nil {
		return err
	}

	for _, seed := range m.seeds {
		if !applied[seed.Name] {
			if err := m.runSeed(ctx, seed); err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *FileLoader) LoadMigrationsFromDir(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	migrations := make(map[int]Migration)
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading file %s: %w", entry.Name(), err)
		}

		// Parse filename: 001_create_users_up.sql or 001_create_users_down.sql
		parts := strings.Split(strings.TrimSuffix(entry.Name(), ".sql"), "_")
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid migration filename: %s", entry.Name())
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid version in filename %s: %w", entry.Name(), err)
		}

		isUp := strings.HasSuffix(parts[len(parts)-1], "up")
		description := strings.Join(parts[1:len(parts)-1], "_")

		migration, exists := migrations[version]
		if !exists {
			migration = Migration{
				Version:     version,
				Description: description,
			}
		}

		if isUp {
			migration.Up = string(content)
		} else {
			migration.Down = string(content)
		}

		migrations[version] = migration
	}

	// Convert map to sorted slice
	result := make([]Migration, 0, len(migrations))
	for _, m := range migrations {
		result = append(result, m)
	}

	return result, nil
}

func (l *FileLoader) LoadSeedsFromDir(dir string) ([]Seed, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var seeds []Seed
	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading file %s: %w", entry.Name(), err)
		}

		// Parse filename: 001_admin_user.sql
		parts := strings.Split(strings.TrimSuffix(entry.Name(), ".sql"), "_")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid seed filename: %s", entry.Name())
		}

		priority, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid priority in filename %s: %w", entry.Name(), err)
		}

		name := strings.Join(parts[1:], "_")
		sqlContent := string(content)

		seeds = append(seeds, Seed{
			Name:     name,
			Priority: priority,
			Run: func(ctx context.Context, tx Tx) error {
				return tx.Exec(ctx, sqlContent)
			},
		})
	}

	return seeds, nil
}

func (m *Manager) createSeedsTable(ctx context.Context) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = tx.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS schema_seeds (
					name text PRIMARY KEY,
					applied_at timestamp with time zone DEFAULT current_timestamp
			);
	`)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (m *Manager) createMigrationsTable(ctx context.Context) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = tx.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS schema_migrations (
					version integer PRIMARY KEY,
					description text NOT NULL,
					applied_at timestamp with time zone DEFAULT current_timestamp
			);
	`)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (m *Manager) getAppliedMigrations(ctx context.Context) (map[int]bool, error) {
	rows, err := m.pool.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}
	return applied, nil
}

func (m *Manager) getAppliedSeeds(ctx context.Context) (map[string]bool, error) {
	rows, err := m.pool.Query(ctx, "SELECT name FROM schema_seeds")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, nil
}

func (m *Manager) runMigration(ctx context.Context, migration Migration) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := tx.Exec(ctx, migration.Up); err != nil {
		return fmt.Errorf("executing migration %d: %w", migration.Version, err)
	}

	if err := tx.Exec(ctx,
		"INSERT INTO schema_migrations (version, description) VALUES ($1, $2)",
		migration.Version, migration.Description); err != nil {
		return fmt.Errorf("recording migration %d: %w", migration.Version, err)
	}

	return tx.Commit(ctx)
}

func (m *Manager) runSeed(ctx context.Context, seed Seed) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := seed.Run(ctx, tx); err != nil {
		return fmt.Errorf("executing seed %s: %w", seed.Name, err)
	}

	if err := tx.Exec(ctx,
		"INSERT INTO schema_seeds (name) VALUES ($1)",
		seed.Name); err != nil {
		return fmt.Errorf("recording seed %s: %w", seed.Name, err)
	}

	return tx.Commit(ctx)
}
