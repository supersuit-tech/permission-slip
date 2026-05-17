package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// TimestampForSQLite formats t as UTC RFC3339Nano for storing in SQLite TEXT
// timestamp columns. Passing time.Time directly can serialize as a layout
// SQLite date/time functions cannot parse (e.g. "... +0000 UTC"), which breaks
// datetime()/julianday() comparisons in SQL.
func TimestampForSQLite(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

// NullableTimestampForSQLite returns nil for a nil pointer; otherwise the same
// as TimestampForSQLite.
func NullableTimestampForSQLite(t *time.Time) any {
	if t == nil {
		return nil
	}
	return TimestampForSQLite(*t)
}

// parseSQLiteTimestamp parses RFC3339 / RFC3339Nano timestamps stored as TEXT
// in SQLite (see strftime patterns in migrations). Also accepts common string
// forms produced when binding or reading time.Time via database/sql.
func parseSQLiteTimestamp(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty sqlite timestamp")
	}
	if i := strings.Index(s, " m="); i > 0 {
		s = strings.TrimSpace(s[:i])
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05Z07:00",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized sqlite timestamp: %q", s)
}

func sqliteTimeRequired(ns sql.NullString) (time.Time, error) {
	if !ns.Valid {
		return time.Time{}, fmt.Errorf("sqlite timestamp is null")
	}
	return parseSQLiteTimestamp(ns.String)
}

func sqliteTimePtr(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid || ns.String == "" {
		return nil, nil
	}
	t, err := parseSQLiteTimestamp(ns.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
