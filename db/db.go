package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// DBTX is the interface implemented by *Pool and *Tx. Both wrap database/sql
// types but expose pgx-style context-first method signatures so existing call
// sites that were written against the old pgx-based DBTX don't need renames.
type DBTX interface {
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...any) *sql.Row
}

// stdlibSQL is the subset of *sql.DB / *sql.Tx that Pool / Tx delegate to.
type stdlibSQL interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Pool wraps a *sql.DB and exposes DBTX. Returned by Connect; held by Deps.
type Pool struct {
	raw *sql.DB
}

func (p *Pool) Exec(ctx context.Context, q string, args ...any) (sql.Result, error) {
	return p.raw.ExecContext(ctx, q, args...)
}
func (p *Pool) Query(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return p.raw.QueryContext(ctx, q, args...)
}
func (p *Pool) QueryRow(ctx context.Context, q string, args ...any) *sql.Row {
	return p.raw.QueryRowContext(ctx, q, args...)
}

// Close releases the underlying connection pool.
func (p *Pool) Close() error { return p.raw.Close() }

// Ping checks connectivity.
func (p *Pool) Ping(ctx context.Context) error { return p.raw.PingContext(ctx) }

// Raw exposes the underlying *sql.DB for callers that need it (e.g., goose).
func (p *Pool) Raw() *sql.DB { return p.raw }

// Tx wraps a *sql.Tx and exposes DBTX.
type Tx struct {
	raw *sql.Tx
}

func (t *Tx) Exec(ctx context.Context, q string, args ...any) (sql.Result, error) {
	return t.raw.ExecContext(ctx, q, args...)
}
func (t *Tx) Query(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return t.raw.QueryContext(ctx, q, args...)
}
func (t *Tx) QueryRow(ctx context.Context, q string, args ...any) *sql.Row {
	return t.raw.QueryRowContext(ctx, q, args...)
}

// Commit commits the transaction.
func (t *Tx) Commit() error { return t.raw.Commit() }

// Rollback rolls back the transaction. Safe to call after Commit.
func (t *Tx) Rollback() error { return t.raw.Rollback() }

// TxStarter can begin a new transaction. *Pool satisfies this interface; *Tx
// does not (no nested transactions in SQLite without savepoints).
type TxStarter interface {
	Begin(ctx context.Context) (*Tx, error)
}

// Begin starts a new transaction on the pool.
func (p *Pool) Begin(ctx context.Context) (*Tx, error) {
	tx, err := p.raw.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Tx{raw: tx}, nil
}

// BeginOrContinue starts a new transaction if d implements TxStarter (i.e.,
// it's a *Pool), otherwise it returns d as-is (it's already a *Tx in tests).
// The returned bool indicates whether a new transaction was started — if true,
// the caller is responsible for calling CommitTx/RollbackTx.
func BeginOrContinue(ctx context.Context, d DBTX) (DBTX, bool, error) {
	if starter, ok := d.(TxStarter); ok {
		tx, err := starter.Begin(ctx)
		if err != nil {
			return nil, false, fmt.Errorf("begin transaction: %w", err)
		}
		return tx, true, nil
	}
	return d, false, nil
}

// CommitTx commits the DBTX if it is a *Tx. Only call when BeginOrContinue
// returned owned=true.
func CommitTx(_ context.Context, d DBTX) error {
	if tx, ok := d.(*Tx); ok {
		return tx.Commit()
	}
	return nil
}

// RollbackTx rolls back the DBTX if it is a *Tx. Safe to call even if already
// committed. Only call when BeginOrContinue returned owned=true.
func RollbackTx(_ context.Context, d DBTX) error {
	if tx, ok := d.(*Tx); ok {
		return tx.Rollback()
	}
	return nil
}

// Migrations holds the embedded SQL migration files.
//
//go:embed migrations_sqlite/*.sql
var Migrations embed.FS

// sqliteDSN appends pragmas required for correctness on every connection:
// foreign_keys=on enforces FK constraints, busy_timeout avoids SQLITE_BUSY on
// concurrent writers, journal_mode=WAL gives us a single-writer / many-reader
// setup. The pragmas are applied by modernc.org/sqlite on every new connection
// the pool opens.
func sqliteDSN(path string) string {
	if strings.Contains(path, "?") {
		return path
	}
	if path == ":memory:" {
		return "file::memory:?cache=shared&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	}
	return path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
}

// Connect opens a SQLite database at the given path and verifies connectivity.
// path may be a file path (e.g. /data/permission-slip.db) or ":memory:".
func Connect(ctx context.Context, path string) (*Pool, error) {
	conn, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	if path == ":memory:" {
		// shared-cache :memory: requires a single connection so all queries
		// observe the same database.
		conn.SetMaxOpenConns(1)
	} else {
		// SQLite is single-writer; keep the pool modest to limit contention.
		conn.SetMaxOpenConns(8)
	}
	if err := conn.PingContext(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}
	return &Pool{raw: conn}, nil
}

// OpenMigrationDB opens a *sql.DB configured for goose migrations using the
// embedded SQLite migration files. The caller closes the returned connection.
func OpenMigrationDB(path string) (*sql.DB, error) {
	goose.SetBaseFS(Migrations)

	conn, err := goose.OpenDBWithDriver("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, fmt.Errorf("open sqlite database for migrations: %w", err)
	}
	if err := goose.SetDialect("sqlite3"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set goose dialect: %w", err)
	}
	return conn, nil
}

// Migrate runs all pending migrations against the database.
func Migrate(ctx context.Context, path string) error {
	conn, err := OpenMigrationDB(path)
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := goose.UpContext(ctx, conn, "migrations_sqlite"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// IsUniqueViolation reports whether err is a SQLite UNIQUE constraint violation.
// Replaces the Postgres-specific pgconn.PgError code 23505 check.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed: UNIQUE")
}

// IsCheckViolation reports whether err is a SQLite CHECK constraint violation.
func IsCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "CHECK constraint failed") ||
		strings.Contains(msg, "constraint failed: CHECK")
}

// IsForeignKeyViolation reports whether err is a SQLite foreign key violation.
func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "FOREIGN KEY constraint failed") ||
		strings.Contains(msg, "constraint failed: FOREIGN KEY")
}

// ErrNoRows is an alias for sql.ErrNoRows.
var ErrNoRows = sql.ErrNoRows

// IsNoRows reports whether err is sql.ErrNoRows.
func IsNoRows(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// RowsAffected returns the rows-affected count from a sql.Result, panicking
// on the rare error case (which the SQLite driver does not produce). Provided
// so call sites can keep the pgx-era simple `tag.RowsAffected()` shape.
func RowsAffected(r sql.Result) int64 {
	n, err := r.RowsAffected()
	if err != nil {
		// modernc.org/sqlite never returns an error here. Panic so a future
		// driver change surfaces immediately rather than silently returning 0.
		panic(fmt.Sprintf("RowsAffected: %v", err))
	}
	return n
}

// InPlaceholders returns a comma-separated list of positional placeholders
// starting at startParam (1-indexed), e.g. InPlaceholders(3, 4) → "$3,$4,$5,$6".
// Used to build dynamic IN (...) clauses for queries that need variable-length
// parameter lists (replacing Postgres's = ANY($N) array operator).
func InPlaceholders(startParam, count int) string {
	ps := make([]string, count)
	for i := range ps {
		ps[i] = fmt.Sprintf("$%d", startParam+i)
	}
	return strings.Join(ps, ",")
}

// StringsToArgs converts a []string to []any for use with variadic SQL args.
func StringsToArgs(ss []string) []any {
	args := make([]any, len(ss))
	for i, s := range ss {
		args[i] = s
	}
	return args
}
