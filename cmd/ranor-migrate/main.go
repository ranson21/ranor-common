// cmd/ranor-migrate/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ranson21/ranor-common/pkg/database/adapter"
	"github.com/ranson21/ranor-common/pkg/database/config"
	"github.com/ranson21/ranor-common/pkg/database/connection"
	"github.com/ranson21/ranor-common/pkg/database/seeder"
)

func main() {
	generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)
	migrateCmd := flag.NewFlagSet("migrate", flag.ExitOnError)

	// Generate flags
	genType := generateCmd.String("type", "", "Type to generate (migration/seed)")
	name := generateCmd.String("name", "", "Name of migration/seed")

	// Migrate flags
	service := migrateCmd.String("service", "", "Service name")
	env := migrateCmd.String("env", "local", "Environment")
	migrationsDir := migrateCmd.String("migrations", "config/db/migrations", "Migrations path")
	seedsDir := migrateCmd.String("seeds", "config/db/seeds", "Seeds path")

	if len(os.Args) < 2 {
		fmt.Println("Expected 'generate' or 'migrate' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		generateCmd.Parse(os.Args[2:])
		if *genType == "" || *name == "" {
			generateCmd.PrintDefaults()
			os.Exit(1)
		}
		generate(*genType, *name)
	case "migrate":
		migrateCmd.Parse(os.Args[2:])
		if *service == "" {
			migrateCmd.PrintDefaults()
			os.Exit(1)
		}
		runMigrations(*service, *env, *migrationsDir, *seedsDir)
	default:
		fmt.Println("Expected 'generate' or 'migrate' subcommands")
		os.Exit(1)
	}
}

func generate(genType, name string) {
	switch strings.ToLower(genType) {
	case "migration":
		generateMigration(name)
	case "seed":
		generateSeed(name)
	default:
		log.Fatalf("Unknown type: %s", genType)
	}
}

func generateMigration(name string) {
	timestamp := time.Now().Format("20060102150405")
	baseDir := "config/db/migrations"

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatal(err)
	}

	// Create up migration
	upPath := filepath.Join(baseDir, fmt.Sprintf("%s_%s_up.sql", timestamp, name))
	if err := os.WriteFile(upPath, []byte("-- Write your up migration here\n"), 0644); err != nil {
		log.Fatal(err)
	}

	// Create down migration
	downPath := filepath.Join(baseDir, fmt.Sprintf("%s_%s_down.sql", timestamp, name))
	if err := os.WriteFile(downPath, []byte("-- Write your down migration here\n"), 0644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created migration files:\n%s\n%s\n", upPath, downPath)
}

func generateSeed(name string) {
	baseDir := "config/db/seeds"
	timestamp := time.Now().Format("20060102150405")

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatal(err)
	}

	path := filepath.Join(baseDir, fmt.Sprintf("%s_%s.sql", timestamp, name))
	if err := os.WriteFile(path, []byte("-- Write your seed data here\n"), 0644); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Created seed file: %s\n", path)
}

func runMigrations(service, env string, migrationsDir, seedsDir string) {
	cfg := config.NewDatabaseConfig(config.Environment(env), service)
	db, err := connection.NewDB(&cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	loader := seeder.NewFileLoader()

	migrations, err := loader.LoadMigrationsFromDir(migrationsDir)
	if err != nil {
		log.Fatal("loading migrations:", err)
	}

	seeds, err := loader.LoadSeedsFromDir(seedsDir)
	if err != nil {
		log.Fatal("loading seeds:", err)
	}

	manager := seeder.NewManager(adapter.NewPgxAdapter(db.GetPool()))
	for _, m := range migrations {
		manager.AddMigration(m)
	}
	for _, s := range seeds {
		manager.AddSeed(s)
	}

	ctx := context.Background()
	if err := manager.RunMigrations(ctx); err != nil {
		log.Fatal("running migrations:", err)
	}
	if err := manager.RunSeeds(ctx); err != nil {
		log.Fatal("running seeds:", err)
	}

	fmt.Println("Successfully ran all migrations and seeds")
}
