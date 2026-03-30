package firestore

import (
	"encoding/json"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGet_FoundAndFiltered(t *testing.T) {
	t.Parallel()
	mock := newMockRunner()
	mock.getData["users/alice"] = map[string]interface{}{"name": "Alice", "secret": "x"}
	conn := newForTest(mock)
	action := conn.Actions()["firestore.get"]

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "firestore.get",
		Parameters: json.RawMessage(`{
			"path":"users/alice",
			"allowed_paths":["users/alice"],
			"allowed_read_fields":["name"]
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(res.Data, &payload); err != nil {
		t.Fatal(err)
	}
	if !payload["found"].(bool) {
		t.Fatal("expected found")
	}
	data := payload["data"].(map[string]interface{})
	if _, has := data["secret"]; has {
		t.Fatalf("secret should be filtered: %#v", data)
	}
	if data["name"] != "Alice" {
		t.Fatalf("name missing: %#v", data)
	}
}

func TestGet_NotFoundGRPCReturnsFoundFalse(t *testing.T) {
	t.Parallel()
	mock := newMockRunner()
	mock.getErr = status.Error(codes.NotFound, "no document")
	conn := newForTest(mock)
	action := conn.Actions()["firestore.get"]

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "firestore.get",
		Parameters: json.RawMessage(`{
			"path":"users/missing",
			"allowed_paths":["users/missing"]
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("NotFound should be success with found:false, got err %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(res.Data, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["found"].(bool) {
		t.Fatal("expected found false")
	}
	if payload["data"] != nil {
		t.Fatalf("expected data null, got %#v", payload["data"])
	}
}

func TestGet_NotInAllowlist(t *testing.T) {
	t.Parallel()
	conn := newForTest(newMockRunner())
	action := conn.Actions()["firestore.get"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "firestore.get",
		Parameters: json.RawMessage(`{
			"path":"users/bob",
			"allowed_paths":["users/alice"]
		}`),
		Credentials: validCreds(),
	})
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("want validation error, got %v", err)
	}
}
