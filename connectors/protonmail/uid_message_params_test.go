package protonmail

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestParseUIDMessageParams_SingleID(t *testing.T) {
	t.Parallel()

	params, err := parseUIDMessageParams([]byte(`{"message_id": 5, "folder": "INBOX"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params.MessageIDs) != 1 || params.MessageIDs[0] != 5 {
		t.Fatalf("expected [5], got %v", params.MessageIDs)
	}
}

func TestParseUIDMessageParams_MissingIDs(t *testing.T) {
	t.Parallel()

	_, err := parseUIDMessageParams([]byte(`{"folder": "INBOX"}`))
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestMoveToFolderParams_MissingTarget(t *testing.T) {
	t.Parallel()

	params, err := parseMoveToFolderParams([]byte(`{"message_id": 5}`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if err := params.validate(); err == nil {
		t.Fatal("expected validation error for missing target_folder")
	}
}

func TestMoveToFolderParams_SameFolder(t *testing.T) {
	t.Parallel()

	raw, _ := json.Marshal(map[string]any{
		"message_id":    5,
		"folder":        "INBOX",
		"target_folder": "INBOX",
	})
	params, err := parseMoveToFolderParams(raw)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if err := params.validate(); err == nil {
		t.Fatal("expected validation error for identical folders")
	}
}

func TestDeleteEmailParams_RejectsTrashFolder(t *testing.T) {
	t.Parallel()

	params, err := parseUIDMessageParams([]byte(`{"message_id": 5, "folder": "Trash"}`))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	action := &deleteEmailAction{conn: New()}
	_, err = action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.delete",
		Parameters:  mustJSON(t, map[string]any{"message_id": params.MessageIDs[0], "folder": params.Folder}),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error for Trash folder")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}
