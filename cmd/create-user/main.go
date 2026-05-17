package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/supersuit-tech/permission-slip/auth"
	"github.com/supersuit-tech/permission-slip/db"
)

const minPasswordLen = 8

func main() {
	usernameFlag := flag.String("username", "", "profile username (default: derived from email local part)")
	flag.Parse()
	args := flag.Args()
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: create-user [-username <name>] <email> <password>")
		os.Exit(1)
	}

	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		log.Fatal("DATABASE_PATH is required")
	}

	email := strings.ToLower(strings.TrimSpace(args[0]))
	password := args[1]
	if len(password) < minPasswordLen {
		log.Fatalf("password must be at least %d characters", minPasswordLen)
	}

	username := strings.TrimSpace(*usernameFlag)
	if username == "" {
		username = usernameFromEmail(email)
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

	uid := uuid.New().String()

	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("begin transaction: %v", err)
	}

	if err := db.CreateUserWithPassword(ctx, tx, uid, email, hash); err != nil {
		_ = tx.Rollback()
		log.Fatalf("create user: %v", err)
	}

	profile, err := db.CreateProfile(ctx, tx, uid, username, email, false)
	if err != nil {
		_ = tx.Rollback()
		log.Fatalf("create profile: %v", err)
	}

	billing := os.Getenv("BILLING_ENABLED") == "true"
	if _, err := db.CreateSubscription(ctx, tx, profile.ID, db.DefaultPlanID(billing)); err != nil {
		_ = tx.Rollback()
		log.Fatalf("create subscription: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("commit: %v", err)
	}

	fmt.Printf("created user id=%s email=%s username=%s\n", uid, email, profile.Username)
}

func usernameFromEmail(email string) string {
	at := strings.LastIndex(email, "@")
	local := email
	if at > 0 {
		local = email[:at]
	}
	var b strings.Builder
	for _, r := range strings.ToLower(local) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
		default:
			b.WriteRune('_')
		}
	}
	s := strings.Trim(b.String(), "_-")
	if len(s) < 3 {
		return "user"
	}
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}
