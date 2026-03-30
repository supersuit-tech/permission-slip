package mongodb

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestFind_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mongodb.find"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "missing database",
			params: `{"collection":"users"}`,
		},
		{
			name:   "missing collection",
			params: `{"database":"mydb"}`,
		},
		{
			name:   "invalid JSON",
			params: `{invalid}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "mongodb.find",
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

func TestFind_InvalidLimit(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mongodb.find"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "limit too high",
			params: `{"database":"mydb","collection":"users","limit":1001}`,
		},
		{
			name:   "limit zero",
			params: `{"database":"mydb","collection":"users","limit":0}`,
		},
		{
			name:   "negative limit",
			params: `{"database":"mydb","collection":"users","limit":-1}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "mongodb.find",
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

func TestFind_DisallowedFilterOperator(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mongodb.find"]

	tests := []struct {
		name   string
		params string
	}{
		{
			name:   "$where operator",
			params: `{"database":"mydb","collection":"users","filter":{"$where":"this.age > 5"}}`,
		},
		{
			name:   "$regex operator",
			params: `{"database":"mydb","collection":"users","filter":{"name":{"$regex":".*"}}}`,
		},
		{
			name:   "nested disallowed operator",
			params: `{"database":"mydb","collection":"users","filter":{"$and":[{"name":{"$regex":"test"}}]}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "mongodb.find",
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

func TestFind_MissingCredentials(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["mongodb.find"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "mongodb.find",
		Parameters:  json.RawMessage(`{"database":"mydb","collection":"users"}`),
		Credentials: connectors.NewCredentials(map[string]string{}),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
