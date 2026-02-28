package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// DBTX is the interface satisfied by both *pgxpool.Pool and pgx.Tx.
// Using this interface lets production code work with a connection pool
// and test code work with a per-test transaction that rolls back automatically.
type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// TxStarter can begin a new database transaction. *pgxpool.Pool satisfies
// this interface. In test code where DBTX is already a pgx.Tx, callers
// should use the provided DBTX directly instead of starting a nested tx.
type TxStarter interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

// BeginOrContinue starts a new transaction if db implements TxStarter (i.e.,
// it's a *pgxpool.Pool), otherwise it returns db as-is (it's already a
// pgx.Tx in tests). The returned bool indicates whether a new transaction
// was started — if true, the caller is responsible for calling Commit/Rollback.
func BeginOrContinue(ctx context.Context, d DBTX) (DBTX, bool, error) {
	if starter, ok := d.(TxStarter); ok {
		tx, err := starter.Begin(ctx)
		if err != nil {
			return nil, false, fmt.Errorf("begin transaction: %w", err)
		}
		return tx, true, nil
	}
	// Already a transaction (e.g., in tests) — use it directly.
	return d, false, nil
}

// CommitTx commits the DBTX if it is a pgx.Tx. This should only be called
// when BeginOrContinue returned owned=true.
func CommitTx(ctx context.Context, d DBTX) error {
	if tx, ok := d.(pgx.Tx); ok {
		return tx.Commit(ctx)
	}
	return nil
}

// RollbackTx rolls back the DBTX if it is a pgx.Tx. Safe to call even if
// already committed. This should only be called when BeginOrContinue returned
// owned=true.
func RollbackTx(ctx context.Context, d DBTX) error {
	if tx, ok := d.(pgx.Tx); ok {
		return tx.Rollback(ctx)
	}
	return nil
}

// Migrations holds the embedded SQL migration files from db/migrations/.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// Connect creates a pgx connection pool from the given database URL.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	return pool, nil
}

// OpenMigrationDB opens a *sql.DB configured for goose migrations using the
// embedded migration files. The caller is responsible for closing the returned
// connection. This is shared between Migrate() and cmd/migrate.
func OpenMigrationDB(databaseURL string) (*sql.DB, error) {
	goose.SetBaseFS(Migrations)

	conn, err := goose.OpenDBWithDriver("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("unable to open database for migrations: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("unable to set dialect: %w", err)
	}

	return conn, nil
}

// Migrate runs all pending migrations against the database.
func Migrate(ctx context.Context, databaseURL string) error {
	conn, err := OpenMigrationDB(databaseURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := goose.UpContext(ctx, conn, "migrations"); err != nil {
		return fmt.Errorf("unable to run migrations: %w", err)
	}

	return nil
}
