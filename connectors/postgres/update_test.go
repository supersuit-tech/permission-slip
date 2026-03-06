package postgres

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdate_Success(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.update"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "postgres.update",
		Parameters: json.RawMessage(fmt.Sprintf(`{
			"table": "%s",
			"set": {"value": 99},
			"where": {"name": "alpha"}
		}`, table)),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["rows_affected"].(float64) != 1 {
		t.Errorf("rows_affected = %v, want 1", data["rows_affected"])
	}
}

func TestUpdate_WithReturning(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.update"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "postgres.update",
		Parameters: json.RawMessage(fmt.Sprintf(`{
			"table": "%s",
			"set": {"active": false},
			"where": {"name": "alpha"},
			"returning": ["name", "active"]
		}`, table)),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	returned := data["returned"].([]interface{})
	if len(returned) != 1 {
		t.Fatalf("returned has %d rows, want 1", len(returned))
	}
	row := returned[0].(map[string]interface{})
	if row["active"] != false {
		t.Errorf("returned active = %v, want false", row["active"])
	}
}

func TestUpdate_NoMatchingRows(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.update"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "postgres.update",
		Parameters: json.RawMessage(fmt.Sprintf(`{
			"table": "%s",
			"set": {"value": 0},
			"where": {"name": "nonexistent"}
		}`, table)),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["rows_affected"].(float64) != 0 {
		t.Errorf("rows_affected = %v, want 0", data["rows_affected"])
	}
}

func TestUpdate_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["postgres.update"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing table", `{"set":{"x":1},"where":{"id":1}}`},
		{"missing set", `{"table":"test","where":{"id":1}}`},
		{"missing where", `{"table":"test","set":{"x":1}}`},
		{"empty set", `{"table":"test","set":{},"where":{"id":1}}`},
		{"empty where", `{"table":"test","set":{"x":1},"where":{}}`},
		{"invalid table", `{"table":"'; DROP--","set":{"x":1},"where":{"id":1}}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "postgres.update",
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
