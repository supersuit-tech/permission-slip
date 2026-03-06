package mongodb

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdate_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mongodb.update"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "missing database",
			params: `{"collection":"users","filter":{"_id":"123"},"update":{"$set":{"name":"Bob"}}}`,
		},
		{
			name:   "missing collection",
			params: `{"database":"mydb","filter":{"_id":"123"},"update":{"$set":{"name":"Bob"}}}`,
		},
		{
			name:   "missing filter",
			params: `{"database":"mydb","collection":"users","update":{"$set":{"name":"Bob"}}}`,
		},
		{
			name:   "empty filter",
			params: `{"database":"mydb","collection":"users","filter":{},"update":{"$set":{"name":"Bob"}}}`,
		},
		{
			name:   "missing update",
			params: `{"database":"mydb","collection":"users","filter":{"_id":"123"}}`,
		},
		{
			name:   "empty update",
			params: `{"database":"mydb","collection":"users","filter":{"_id":"123"},"update":{}}`,
		},
		{
			name:   "invalid JSON",
			params: `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "mongodb.update",
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

func TestUpdate_RawReplacementNotAllowed(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mongodb.update"]

	// Attempt a raw document replacement (no $ operators).
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mongodb.update",
		Parameters:  json.RawMessage(`{"database":"mydb","collection":"users","filter":{"_id":"123"},"update":{"name":"Bob","age":30}}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdate_DisallowedFilterOperator(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mongodb.update"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mongodb.update",
		Parameters:  json.RawMessage(`{"database":"mydb","collection":"users","filter":{"$where":"true"},"update":{"$set":{"name":"Bob"}}}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
