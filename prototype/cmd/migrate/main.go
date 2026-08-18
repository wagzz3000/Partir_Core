package main

import (
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: migrate <up|down|force N|version>")
	}

	// Check PARTIR_DB_URL first, then DATABASE_URL, then fallback to dev default
	dbURL := os.Getenv("PARTIR_DB_URL")
	if dbURL == "" {
		dbURL = os.Getenv("DATABASE_URL")
	}
	if dbURL == "" {
		dbURL = "postgres://partir:partirpass@127.0.0.1:5432/partir?sslmode=disable"
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
	case "force":
		if len(os.Args) < 3 {
			log.Fatal("expected version number after 'force'")
		}
		version, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("invalid version number: %v", err)
		}
		if err := m.Force(version); err != nil {
			log.Fatalf("failed to force version: %v", err)
		}
		log.Printf("Forced version to %d\n", version)
	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("failed to get version: %v", err)
		}
		log.Printf("Version: %d, Dirty: %v\n", version, dirty)
	default:
		log.Fatalf("unknown command: %s", cmd)
	}
}
