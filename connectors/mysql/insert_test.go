package mysql

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestInsert_Success(t *testing.T) {
	t.Parallel()

	conn, mock, cleanup := newTestConnector()
	defer cleanup()

	mock.ExpectExec("INSERT INTO `users`").
		WithArgs("alice@example.com", "Alice").
		WillReturnResult(sqlmock.NewResult(1, 1))

	action := conn.Actions()["mysql.insert"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.insert",
		Parameters:  json.RawMessage(`{"table":"users","rows":[{"name":"Alice","email":"alice@example.com"}]}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["rows_affected"] != float64(1) {
		t.Errorf("rows_affected = %v, want 1", data["rows_affected"])
	}
	if data["last_insert_id"] != float64(1) {
		t.Errorf("last_insert_id = %v, want 1", data["last_insert_id"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestInsert_MultipleRows(t *testing.T) {
	t.Parallel()

	conn, mock, cleanup := newTestConnector()
	defer cleanup()

	mock.ExpectExec("INSERT INTO `users`").
		WithArgs("alice@example.com", "Alice", "bob@example.com", "Bob").
		WillReturnResult(sqlmock.NewResult(2, 2))

	action := conn.Actions()["mysql.insert"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "mysql.insert",
		Parameters: json.RawMessage(`{
			"table": "users",
			"rows": [
				{"name": "Alice", "email": "alice@example.com"},
				{"name": "Bob", "email": "bob@example.com"}
			]
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["rows_affected"] != float64(2) {
		t.Errorf("rows_affected = %v, want 2", data["rows_affected"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestInsert_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.insert"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing table", `{"rows":[{"name":"Alice"}]}`},
		{"missing rows", `{"table":"users"}`},
		{"empty rows", `{"table":"users","rows":[]}`},
		{"empty row object", `{"table":"users","rows":[{}]}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "mysql.insert",
				Parameters:  json.RawMessage(tt.params),
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

func TestInsert_InvalidTableName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.insert"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.insert",
		Parameters:  json.RawMessage(`{"table":"users; DROP TABLE users","rows":[{"name":"Alice"}]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestInsert_TableNotAllowed(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.insert"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.insert",
		Parameters:  json.RawMessage(`{"table":"secrets","rows":[{"data":"sensitive"}],"allowed_tables":["users"]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestInsert_ColumnNotAllowed(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.insert"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.insert",
		Parameters:  json.RawMessage(`{"table":"users","rows":[{"name":"Alice","password":"secret"}],"allowed_columns":["name","email"]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestInsert_TooManyRows(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.insert"]

	// Build a rows array with maxInsertRows + 1 entries.
	rows := make([]map[string]any, maxInsertRows+1)
	for i := range rows {
		rows[i] = map[string]any{"name": fmt.Sprintf("user_%d", i)}
	}
	params, _ := json.Marshal(map[string]any{"table": "users", "rows": rows})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.insert",
		Parameters:  params,
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestInsert_InvalidColumnName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.insert"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.insert",
		Parameters:  json.RawMessage(`{"table":"users","rows":[{"name; DROP TABLE users":"Alice"}]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
