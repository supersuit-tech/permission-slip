package firestore

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSet_DisallowedWriteField(t *testing.T) {
	t.Parallel()
	conn := newForTest(newMockRunner())
	action := conn.Actions()["firestore.set"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "firestore.set",
		Parameters: json.RawMessage(`{
			"path":"users/alice",
			"allowed_paths":["users/alice"],
			"allowed_write_fields":["displayName"],
			"data":{"displayName":"Hi","secret":1}
		}`),
		Credentials: validCreds(),
	})
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("want validation error, got %v", err)
	}
}

func TestQuery_InvalidDocumentPathAsCollection(t *testing.T) {
	t.Parallel()
	conn := newForTest(newMockRunner())
	action := conn.Actions()["firestore.query"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "firestore.query",
		Parameters: json.RawMessage(`{
			"collection_path":"users/alice",
			"allowed_paths":["users/alice"]
		}`),
		Credentials: validCreds(),
	})
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("want validation error for document path as collection, got %v", err)
	}
}

func TestDocumentPath_InvalidEvenSegments(t *testing.T) {
	t.Parallel()
	if err := validateDocumentPath("users"); err == nil {
		t.Fatal("expected error for single-segment path")
	}
}
