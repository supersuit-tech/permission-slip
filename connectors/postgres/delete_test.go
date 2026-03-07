package postgres

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDelete_Success(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.delete"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "postgres.delete",
		Parameters: json.RawMessage(fmt.Sprintf(`{
			"table": "%s",
			"where": {"name": "beta"}
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

func TestDelete_WithReturning(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.delete"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "postgres.delete",
		Parameters: json.RawMessage(fmt.Sprintf(`{
			"table": "%s",
			"where": {"active": false},
			"returning": ["name", "value"]
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
	if row["name"] != "beta" {
		t.Errorf("returned name = %v, want beta", row["name"])
	}
}

func TestDelete_NoMatchingRows(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.delete"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "postgres.delete",
		Parameters: json.RawMessage(fmt.Sprintf(`{
			"table": "%s",
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

func TestDelete_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["postgres.delete"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing table", `{"where":{"id":1}}`},
		{"missing where", `{"table":"test"}`},
		{"empty where", `{"table":"test","where":{}}`},
		{"invalid table", `{"table":"'; DROP--","where":{"id":1}}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "postgres.delete",
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
