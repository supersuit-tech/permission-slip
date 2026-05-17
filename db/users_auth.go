package db

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// AuthUserRow is a minimal users row for password authentication.
type AuthUserRow struct {
	ID           string
	Email        string
	PasswordHash string
}

// CreateUserWithPassword inserts into users. Email is normalized to lower-case for lookups.
func CreateUserWithPassword(ctx context.Context, d DBTX, id, email, passwordHash string) error {
	_, err := d.Exec(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES ($1, lower($2), $3)`,
		id, email, passwordHash,
	)
	return err
}

// GetUserAuthByEmail loads a user by case-insensitive email for login.
func GetUserAuthByEmail(ctx context.Context, d DBTX, email string) (*AuthUserRow, error) {
	row := d.QueryRow(ctx,
		`SELECT id, email, password_hash FROM users WHERE lower(email) = lower($1)`,
		email,
	)
	var u AuthUserRow
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// AuthSession is a row from auth_sessions used for refresh-token rotation.
type AuthSession struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func scanAuthSession(row rowScanner) (*AuthSession, error) {
	var s AuthSession
	var expires sql.NullString
	var revoked sql.NullString
	if err := row.Scan(&s.ID, &s.UserID, &expires, &revoked); err != nil {
		return nil, err
	}
	var err2 error
	s.ExpiresAt, err2 = sqliteTimeRequired(expires)
	if err2 != nil {
		return nil, err2
	}
	if revoked.Valid && revoked.String != "" {
		t, err3 := sqliteTimeRequired(revoked)
		if err3 != nil {
			return nil, err3
		}
		s.RevokedAt = &t
	}
	return &s, nil
}

// CreateAuthSession inserts a refresh-token session row.
func CreateAuthSession(ctx context.Context, d DBTX, id, userID, refreshTokenHash string, expiresAt time.Time) error {
	_, err := d.Exec(ctx,
		`INSERT INTO auth_sessions (id, user_id, refresh_token_hash, expires_at) VALUES ($1, $2, $3, $4)`,
		id, userID, refreshTokenHash, TimestampForSQLite(expiresAt),
	)
	return err
}

// GetAuthSessionByRefreshHash returns a session matching the refresh token hash.
// The caller must treat revoked or expired sessions as invalid.
func GetAuthSessionByRefreshHash(ctx context.Context, d DBTX, refreshTokenHash string) (*AuthSession, error) {
	row := d.QueryRow(ctx,
		`SELECT id, user_id, expires_at, revoked_at FROM auth_sessions WHERE refresh_token_hash = $1`,
		refreshTokenHash,
	)
	s, err := scanAuthSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// RevokeAuthSession sets revoked_at if the session is still active.
func RevokeAuthSession(ctx context.Context, d DBTX, sessionID string, revokedAt time.Time) error {
	_, err := d.Exec(ctx,
		`UPDATE auth_sessions SET revoked_at = $2 WHERE id = $1 AND revoked_at IS NULL`,
		sessionID, TimestampForSQLite(revokedAt),
	)
	return err
}

// RevokeAuthSessionByRefreshHash revokes the session matching the given refresh token hash.
func RevokeAuthSessionByRefreshHash(ctx context.Context, d DBTX, refreshTokenHash string, revokedAt time.Time) error {
	_, err := d.Exec(ctx,
		`UPDATE auth_sessions SET revoked_at = $2 WHERE refresh_token_hash = $1 AND revoked_at IS NULL`,
		refreshTokenHash, TimestampForSQLite(revokedAt),
	)
	return err
}
