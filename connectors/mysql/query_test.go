package mysql

import (
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestQuery_Success(t *testing.T) {
	t.Parallel()

	conn, mock, cleanup := newTestConnector()
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id", "name", "email"}).
		AddRow(1, "Alice", "alice@example.com").
		AddRow(2, "Bob", "bob@example.com")
	mock.ExpectQuery("SELECT id, name, email FROM users WHERE active = \\? LIMIT 1000").
		WithArgs(true).
		WillReturnRows(rows)

	action := conn.Actions()["mysql.query"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.query",
		Parameters:  json.RawMessage(`{"sql":"SELECT id, name, email FROM users WHERE active = ?","args":[true]}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["row_count"] != float64(2) {
		t.Errorf("row_count = %v, want 2", data["row_count"])
	}

	resultRows, ok := data["rows"].([]any)
	if !ok {
		t.Fatalf("rows is not an array: %T", data["rows"])
	}
	if len(resultRows) != 2 {
		t.Errorf("got %d rows, want 2", len(resultRows))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestQuery_WithRowLimit(t *testing.T) {
	t.Parallel()

	conn, mock, cleanup := newTestConnector()
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery("SELECT id FROM users LIMIT 10").
		WillReturnRows(rows)

	action := conn.Actions()["mysql.query"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.query",
		Parameters:  json.RawMessage(`{"sql":"SELECT id FROM users","row_limit":10}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestQuery_ExistingLimitNotOverridden(t *testing.T) {
	t.Parallel()

	conn, mock, cleanup := newTestConnector()
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
	mock.ExpectQuery("SELECT id FROM users LIMIT 5").
		WillReturnRows(rows)

	action := conn.Actions()["mysql.query"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.query",
		Parameters:  json.RawMessage(`{"sql":"SELECT id FROM users LIMIT 5"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestQuery_NonSelectRejected(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.query"]

	tests := []struct {
		name string
		sql  string
	}{
		{"insert", `INSERT INTO users (name) VALUES ('test')`},
		{"update", `UPDATE users SET name = 'test'`},
		{"delete", `DELETE FROM users`},
		{"drop", `DROP TABLE users`},
		{"empty", ``},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, _ := json.Marshal(map[string]any{"sql": tt.sql})
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "mysql.query",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestQuery_DangerousKeywordsInSelect(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.query"]

	tests := []struct {
		name string
		sql  string
	}{
		{"select with delete", "SELECT * FROM users; DELETE FROM users"},
		{"select with drop", "SELECT * FROM users; DROP TABLE users"},
		{"select with insert", "SELECT * FROM users; INSERT INTO users (name) VALUES ('hack')"},
		{"select with update", "SELECT * FROM users; UPDATE users SET name = 'hack'"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, _ := json.Marshal(map[string]any{"sql": tt.sql})
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "mysql.query",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestQuery_SafeColumnNames(t *testing.T) {
	t.Parallel()

	conn, mock, cleanup := newTestConnector()
	defer cleanup()

	// Column name "deleted" contains "DELETE" as a substring but should be allowed.
	rows := sqlmock.NewRows([]string{"deleted"}).AddRow(false)
	mock.ExpectQuery("SELECT deleted FROM users LIMIT 1000").
		WillReturnRows(rows)

	action := conn.Actions()["mysql.query"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.query",
		Parameters:  json.RawMessage(`{"sql":"SELECT deleted FROM users"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestQuery_AllowedTables(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.query"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.query",
		Parameters:  json.RawMessage(`{"sql":"SELECT * FROM secrets","allowed_tables":["users","orders"]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestQuery_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.query"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.query",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestQuery_MissingCredentials(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.query"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.query",
		Parameters:  json.RawMessage(`{"sql":"SELECT 1"}`),
		Credentials: connectors.NewCredentials(map[string]string{}),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
}

func TestContainsKeyword(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		sql     string
		keyword string
		want    bool
	}{
		{"standalone DELETE", "SELECT * FROM users; DELETE FROM users", "DELETE", true},
		{"DELETED column", "SELECT DELETED FROM users", "DELETE", false},
		{"UPDATE standalone", "SELECT * FROM t; UPDATE t SET x=1", "UPDATE", true},
		{"UPDATED column", "SELECT UPDATED_AT FROM t", "UPDATE", false},
		{"INSERT standalone", "INSERT INTO users VALUES (1)", "INSERT", true},
		{"INSERTS table", "SELECT * FROM INSERTS", "INSERT", false},
		{"DROP standalone", "DROP TABLE users", "DROP", true},
		{"DROPDOWN column", "SELECT DROPDOWN FROM t", "DROP", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsKeyword(tt.sql, tt.keyword); got != tt.want {
				t.Errorf("containsKeyword(%q, %q) = %v, want %v", tt.sql, tt.keyword, got, tt.want)
			}
		})
	}
}
