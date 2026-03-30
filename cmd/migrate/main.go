package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/pressly/goose/v3"
	"github.com/supersuit-tech/permission-slip/db"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: migrate <up|down|status>")
		os.Exit(1)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	conn, err := db.OpenMigrationDB(dbURL)
	if err != nil {
		log.Fatalf("failed to open migration database: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	command := os.Args[1]

	switch command {
	case "up":
		if err := goose.UpContext(ctx, conn, "migrations"); err != nil {
			log.Fatalf("migration up failed: %v", err)
		}
	case "down":
		if err := goose.DownContext(ctx, conn, "migrations"); err != nil {
			log.Fatalf("migration down failed: %v", err)
		}
	case "status":
		if err := goose.StatusContext(ctx, conn, "migrations"); err != nil {
			log.Fatalf("migration status failed: %v", err)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s (expected up, down, or status)\n", command)
		os.Exit(1)
	}
}
