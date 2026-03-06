package mysql

import (
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDelete_Success(t *testing.T) {
	t.Parallel()

	conn, mock, cleanup := newTestConnector()
	defer cleanup()

	mock.ExpectExec("DELETE FROM `users` WHERE `id` = \\?").
		WithArgs(float64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	action := conn.Actions()["mysql.delete"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.delete",
		Parameters:  json.RawMessage(`{"table":"users","where":{"id":1}}`),
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

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDelete_MultipleWhereConditions(t *testing.T) {
	t.Parallel()

	conn, mock, cleanup := newTestConnector()
	defer cleanup()

	mock.ExpectExec("DELETE FROM `users` WHERE `active` = \\? AND `role` = \\?").
		WithArgs(false, "guest").
		WillReturnResult(sqlmock.NewResult(0, 5))

	action := conn.Actions()["mysql.delete"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.delete",
		Parameters:  json.RawMessage(`{"table":"users","where":{"active":false,"role":"guest"}}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["rows_affected"] != float64(5) {
		t.Errorf("rows_affected = %v, want 5", data["rows_affected"])
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestDelete_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.delete"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing table", `{"where":{"id":1}}`},
		{"missing where", `{"table":"users"}`},
		{"empty where", `{"table":"users","where":{}}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "mysql.delete",
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

func TestDelete_TableNotAllowed(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.delete"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.delete",
		Parameters:  json.RawMessage(`{"table":"secrets","where":{"id":1},"allowed_tables":["temp_records"]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDelete_InvalidTableName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.delete"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.delete",
		Parameters:  json.RawMessage(`{"table":"users; DROP TABLE users","where":{"id":1}}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDelete_InvalidColumnInWhere(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.delete"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.delete",
		Parameters:  json.RawMessage(`{"table":"users","where":{"bad col":1}}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
