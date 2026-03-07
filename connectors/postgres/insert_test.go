package postgres

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestInsert_Success(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.insert"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "postgres.insert",
		Parameters: json.RawMessage(fmt.Sprintf(`{
			"table": "%s",
			"rows": [
				{"name": "delta", "value": 40, "active": true},
				{"name": "epsilon", "value": 50, "active": false}
			]
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
	if data["rows_affected"].(float64) != 2 {
		t.Errorf("rows_affected = %v, want 2", data["rows_affected"])
	}
}

func TestInsert_WithReturning(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.insert"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "postgres.insert",
		Parameters: json.RawMessage(fmt.Sprintf(`{
			"table": "%s",
			"rows": [{"name": "zeta", "value": 60}],
			"returning": ["id", "name"]
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
	if row["name"] != "zeta" {
		t.Errorf("returned name = %v, want zeta", row["name"])
	}
	if row["id"] == nil {
		t.Error("returned id is nil, want a value")
	}
}

func TestInsert_WithExplicitColumns(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.insert"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "postgres.insert",
		Parameters: json.RawMessage(fmt.Sprintf(`{
			"table": "%s",
			"columns": ["name", "value"],
			"rows": [{"name": "eta", "value": 70}]
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

func TestInsert_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["postgres.insert"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing table", `{"rows":[{"name":"x"}]}`},
		{"missing rows", `{"table":"test"}`},
		{"empty rows", `{"table":"test","rows":[]}`},
		{"invalid JSON", `{invalid}`},
		{"invalid table name", `{"table":"'; DROP TABLE--","rows":[{"x":1}]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "postgres.insert",
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

func TestInsert_ConstraintViolation(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.insert"]

	// NOT NULL violation — name is required.
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "postgres.insert",
		Parameters: json.RawMessage(fmt.Sprintf(`{
			"table": "%s",
			"columns": ["name"],
			"rows": [{"name": null}]
		}`, table)),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for NULL violation, got nil")
	}
}
