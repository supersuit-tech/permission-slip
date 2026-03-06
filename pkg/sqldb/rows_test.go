package sqldb

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestScanRows_Basic(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mockRows := sqlmock.NewRows([]string{"id", "name"}).
		AddRow(1, "alice").
		AddRow(2, "bob")
	mock.ExpectQuery("SELECT").WillReturnRows(mockRows)

	rows, err := db.Query("SELECT id, name FROM users")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	columns, results, err := ScanRows(rows, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(columns) != 2 || columns[0] != "id" || columns[1] != "name" {
		t.Errorf("columns = %v, want [id name]", columns)
	}
	if len(results) != 2 {
		t.Fatalf("results len = %d, want 2", len(results))
	}
	if results[0]["name"] != "alice" {
		t.Errorf("results[0][name] = %v, want alice", results[0]["name"])
	}
}

func TestScanRows_ByteToString(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mockRows := sqlmock.NewRows([]string{"data"}).
		AddRow([]byte("hello"))
	mock.ExpectQuery("SELECT").WillReturnRows(mockRows)

	rows, err := db.Query("SELECT data FROM t")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	_, results, err := ScanRows(rows, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Should be converted to string, not remain as []byte.
	if s, ok := results[0]["data"].(string); !ok || s != "hello" {
		t.Errorf("results[0][data] = %v (%T), want string hello", results[0]["data"], results[0]["data"])
	}
}

func TestScanRows_EmptyResultSet(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mockRows := sqlmock.NewRows([]string{"id"})
	mock.ExpectQuery("SELECT").WillReturnRows(mockRows)

	rows, err := db.Query("SELECT id FROM t")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	columns, results, err := ScanRows(rows, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(columns) != 1 {
		t.Errorf("columns = %v, want [id]", columns)
	}
	// Should return empty slice, not nil.
	if results == nil {
		t.Fatal("results should not be nil")
	}
	if len(results) != 0 {
		t.Errorf("results len = %d, want 0", len(results))
	}
}

func TestScanRows_MaxRows(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mockRows := sqlmock.NewRows([]string{"id"}).
		AddRow(1).AddRow(2).AddRow(3).AddRow(4).AddRow(5)
	mock.ExpectQuery("SELECT").WillReturnRows(mockRows)

	rows, err := db.Query("SELECT id FROM t")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// maxRows=2 should read at most 3 rows (2+1 for truncation detection).
	_, results, err := ScanRows(rows, 2, nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 3 {
		t.Errorf("results len = %d, want 3 (maxRows+1)", len(results))
	}
}

func TestScanRows_ErrMapper(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mockRows := sqlmock.NewRows([]string{"id"}).
		AddRow(1).
		RowError(0, errors.New("pg connection lost"))
	mock.ExpectQuery("SELECT").WillReturnRows(mockRows)

	rows, err := db.Query("SELECT id FROM t")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	customMapped := false
	_, _, err = ScanRows(rows, 0, func(e error) error {
		customMapped = true
		return &connectors.ExternalError{Message: "custom: " + e.Error()}
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if !customMapped {
		t.Error("expected errMapper to be called")
	}
	var extErr *connectors.ExternalError
	if !errors.As(err, &extErr) {
		t.Errorf("expected ExternalError, got %T", err)
	}
}

func TestDetectTruncation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rows      int
		limit     int
		wantLen   int
		wantTrunc bool
	}{
		{"under_limit", 3, 5, 3, false},
		{"at_limit", 5, 5, 5, false},
		{"over_limit", 6, 5, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make([]map[string]interface{}, tt.rows)
			for i := range input {
				input[i] = map[string]interface{}{"i": i}
			}

			results, truncated := DetectTruncation(input, tt.limit)
			if len(results) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(results), tt.wantLen)
			}
			if truncated != tt.wantTrunc {
				t.Errorf("truncated = %v, want %v", truncated, tt.wantTrunc)
			}
		})
	}
}
