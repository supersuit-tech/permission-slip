package protonmail

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestArchiveEmail_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &archiveEmailAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.archive_email",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestArchiveEmail_MissingMessageIDs(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &archiveEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]any{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.archive_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing message_ids")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestArchiveEmail_EmptyMessageIDs(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &archiveEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"message_ids": []int{},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.archive_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for empty message_ids")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestArchiveEmail_ZeroMessageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &archiveEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"message_ids": []int{0},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.archive_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for zero message_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestArchiveEmail_TooManyMessageIDs(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &archiveEmailAction{conn: conn}

	ids := make([]int, 51)
	for i := range ids {
		ids[i] = i + 1
	}
	params, _ := json.Marshal(map[string]any{
		"message_ids": ids,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.archive_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for too many message_ids")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestArchiveEmail_ArchiveFolderRejected(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &archiveEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"message_ids": []int{1},
		"folder":      "Archive",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.archive_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when source folder is Archive")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestArchiveEmailParams_Defaults(t *testing.T) {
	t.Parallel()

	p := &archiveEmailParams{MessageIDs: []uint32{1}}
	if err := p.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Folder != "INBOX" {
		t.Errorf("expected default folder 'INBOX', got %q", p.Folder)
	}
}
