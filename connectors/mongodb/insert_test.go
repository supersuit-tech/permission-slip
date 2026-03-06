package mongodb

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestInsert_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mongodb.insert"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "missing database",
			params: `{"collection":"users","documents":[{"name":"Alice"}]}`,
		},
		{
			name:   "missing collection",
			params: `{"database":"mydb","documents":[{"name":"Alice"}]}`,
		},
		{
			name:   "missing documents",
			params: `{"database":"mydb","collection":"users"}`,
		},
		{
			name:   "empty documents array",
			params: `{"database":"mydb","collection":"users","documents":[]}`,
		},
		{
			name:   "invalid JSON",
			params: `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "mongodb.insert",
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

func TestInsert_TooManyDocuments(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mongodb.insert"]

	// Build an array with 101 documents.
	docs := "["
	for i := range 101 {
		if i > 0 {
			docs += ","
		}
		docs += fmt.Sprintf(`{"i":%d}`, i)
	}
	docs += "]"

	params := fmt.Sprintf(`{"database":"mydb","collection":"users","documents":%s}`, docs)
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mongodb.insert",
		Parameters:  json.RawMessage(params),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
