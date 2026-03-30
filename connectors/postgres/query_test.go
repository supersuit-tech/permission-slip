package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestQuery_Success(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.query"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "postgres.query",
		Parameters:  json.RawMessage(fmt.Sprintf(`{"sql":"SELECT name, value FROM %s WHERE active = $1 ORDER BY name","params":[true]}`, table)),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}

	rowCount := data["row_count"].(float64)
	if rowCount != 2 {
		t.Errorf("row_count = %v, want 2", rowCount)
	}
	if data["truncated"].(bool) {
		t.Error("truncated = true, want false")
	}

	rows := data["rows"].([]interface{})
	firstRow := rows[0].(map[string]interface{})
	if firstRow["name"] != "alpha" {
		t.Errorf("first row name = %v, want alpha", firstRow["name"])
	}
}

func TestQuery_EmptyResult(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.query"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "postgres.query",
		Parameters:  json.RawMessage(fmt.Sprintf(`{"sql":"SELECT * FROM %s WHERE value > $1","params":[9999]}`, table)),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["row_count"].(float64) != 0 {
		t.Errorf("row_count = %v, want 0", data["row_count"])
	}
}

func TestQuery_MaxRows(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.query"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "postgres.query",
		Parameters:  json.RawMessage(fmt.Sprintf(`{"sql":"SELECT * FROM %s ORDER BY id","max_rows":2}`, table)),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["row_count"].(float64) != 2 {
		t.Errorf("row_count = %v, want 2", data["row_count"])
	}
	if !data["truncated"].(bool) {
		t.Error("truncated = false, want true")
	}
}

func TestQuery_RejectsNonSelect(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["postgres.query"]

	tests := []struct {
		name string
		sql  string
	}{
		{"INSERT", `{"sql":"INSERT INTO test VALUES (1)"}`},
		{"UPDATE", `{"sql":"UPDATE test SET x = 1"}`},
		{"DELETE", `{"sql":"DELETE FROM test"}`},
		{"DROP", `{"sql":"DROP TABLE test"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "postgres.query",
				Parameters:  json.RawMessage(tt.sql),
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

func TestQuery_WithCTE(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.query"]

	// Read-only CTE should succeed.
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "postgres.query",
		Parameters:  json.RawMessage(fmt.Sprintf(`{"sql":"WITH active AS (SELECT name, value FROM %s WHERE active = true) SELECT * FROM active ORDER BY name"}`, table)),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["row_count"].(float64) != 2 {
		t.Errorf("row_count = %v, want 2", data["row_count"])
	}
}

func TestQuery_ReadOnlyEnforcement(t *testing.T) {
	t.Parallel()
	table := setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.query"]

	// CTE with write operation should be rejected by the read-only transaction.
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "postgres.query",
		Parameters:  json.RawMessage(fmt.Sprintf(`{"sql":"WITH d AS (DELETE FROM %s RETURNING *) SELECT * FROM d"}`, table)),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for write in read-only transaction, got nil")
	}
}

func TestQuery_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["postgres.query"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing sql", `{}`},
		{"empty sql", `{"sql":""}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "postgres.query",
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

func TestQuery_BadCredentials(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["postgres.query"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "postgres.query",
		Parameters:  json.RawMessage(`{"sql":"SELECT 1"}`),
		Credentials: connectors.NewCredentials(map[string]string{"connection_string": "postgres://baduser:badpass@localhost:5432/nonexistent?sslmode=disable"}),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	// Should be either AuthError or ExternalError depending on the failure mode.
	if connectors.IsValidationError(err) {
		t.Errorf("unexpected ValidationError for bad credentials: %v", err)
	}
}

func TestQuery_Timeout(t *testing.T) {
	t.Parallel()
	setupTestTable(t)

	conn := New()
	action := conn.Actions()["postgres.query"]

	ctx, cancel := context.WithTimeout(t.Context(), 1)
	defer cancel()

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "postgres.query",
		Parameters:  json.RawMessage(`{"sql":"SELECT pg_sleep(10)"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
}
