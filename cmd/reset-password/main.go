package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/supersuit-tech/permission-slip/auth"
	"github.com/supersuit-tech/permission-slip/db"
)

const minPasswordLen = 8

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: reset-password <email> <new-password>")
		os.Exit(1)
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		log.Fatal("DATABASE_PATH is required")
	}

	email := strings.ToLower(strings.TrimSpace(os.Args[1]))
	password := os.Args[2]
	if len(password) < minPasswordLen {
		log.Fatalf("password must be at least %d characters", minPasswordLen)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, dbPath)
	if err != nil {
		log.Fatalf("connect database: %v", err)
	}
	defer pool.Close()

	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	found, err := db.UpdatePasswordHash(ctx, pool, email, hash)
	if err != nil {
		log.Fatalf("update password: %v", err)
	}
	if !found {
		log.Fatalf("no user found with email %s", email)
	}

	fmt.Printf("password updated for %s\n", email)
}
