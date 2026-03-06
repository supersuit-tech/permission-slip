package mysql

import (
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdate_Success(t *testing.T) {
	t.Parallel()

	conn, mock, cleanup := newTestConnector()
	defer cleanup()

	mock.ExpectExec("UPDATE `users` SET `name` = \\? WHERE `id` = \\?").
		WithArgs("Bob", float64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	action := conn.Actions()["mysql.update"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.update",
		Parameters:  json.RawMessage(`{"table":"users","set":{"name":"Bob"},"where":{"id":1}}`),
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

func TestUpdate_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.update"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing table", `{"set":{"name":"Bob"},"where":{"id":1}}`},
		{"missing set", `{"table":"users","where":{"id":1}}`},
		{"empty set", `{"table":"users","set":{},"where":{"id":1}}`},
		{"missing where", `{"table":"users","set":{"name":"Bob"}}`},
		{"empty where", `{"table":"users","set":{"name":"Bob"},"where":{}}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "mysql.update",
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

func TestUpdate_TableNotAllowed(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.update"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.update",
		Parameters:  json.RawMessage(`{"table":"secrets","set":{"data":"new"},"where":{"id":1},"allowed_tables":["users"]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdate_ColumnNotAllowed(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.update"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.update",
		Parameters:  json.RawMessage(`{"table":"users","set":{"password":"hack"},"where":{"id":1},"allowed_columns":["name","email","id"]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdate_InvalidTableName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.update"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.update",
		Parameters:  json.RawMessage(`{"table":"users; DROP TABLE","set":{"name":"Bob"},"where":{"id":1}}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdate_InvalidColumnInSet(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.update"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.update",
		Parameters:  json.RawMessage(`{"table":"users","set":{"bad col":"value"},"where":{"id":1}}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdate_InvalidColumnInWhere(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mysql.update"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mysql.update",
		Parameters:  json.RawMessage(`{"table":"users","set":{"name":"Bob"},"where":{"bad col":1}}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
