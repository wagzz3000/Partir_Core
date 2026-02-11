package main

import (
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("expected 'up' or 'down' subcommands")
	}

	// Default to dev config
	dbURL := "postgres://partir:partir@localhost:5432/partir?sslmode=disable"
	if url := os.Getenv("DATABASE_URL"); url != "" {
		dbURL = url
	}

	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		log.Fatalf("failed to create migrate instance: %v", err)
	}

	cmd := os.Args[1]
	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("failed to run up migrations: %v", err)
		}
		log.Println("Migrations applied successfully")
	case "down":
		if err := m.Down(); err != nil {
			log.Fatalf("failed to run down migrations: %v", err)
		}
		log.Println("Migrations rolled back successfully")
	default:
		log.Fatalf("unknown command: %s", cmd)
	}
}
