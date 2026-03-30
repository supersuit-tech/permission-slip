package mongodb

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDelete_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mongodb.delete"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "missing database",
			params: `{"collection":"users","filter":{"_id":"123"}}`,
		},
		{
			name:   "missing collection",
			params: `{"database":"mydb","filter":{"_id":"123"}}`,
		},
		{
			name:   "missing filter",
			params: `{"database":"mydb","collection":"users"}`,
		},
		{
			name:   "empty filter",
			params: `{"database":"mydb","collection":"users","filter":{}}`,
		},
		{
			name:   "invalid JSON",
			params: `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "mongodb.delete",
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

func TestDelete_DisallowedFilterOperator(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mongodb.delete"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mongodb.delete",
		Parameters:  json.RawMessage(`{"database":"mydb","collection":"users","filter":{"$where":"true"}}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
