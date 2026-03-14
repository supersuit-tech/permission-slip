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

	// Test case-insensitive rejection per RFC 3501.
	for _, folder := range []string{"Archive", "archive", "ARCHIVE"} {
		t.Run(folder, func(t *testing.T) {
			t.Parallel()

			params, _ := json.Marshal(map[string]any{
				"message_ids": []int{1},
				"folder":      folder,
			})

			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "protonmail.archive_email",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatalf("expected error when source folder is %q", folder)
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("folder %q: expected ValidationError, got: %T", folder, err)
			}
		})
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

func TestArchiveEmail_SingleMessageID(t *testing.T) {
	t.Parallel()

	// Verify that message_id (singular) is accepted as a shorthand.
	params, err := parseArchiveParams([]byte(`{"message_id": 5}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params.MessageIDs) != 1 || params.MessageIDs[0] != 5 {
		t.Errorf("expected [5], got %v", params.MessageIDs)
	}
}

func TestArchiveEmail_SingleAndBatchCombined(t *testing.T) {
	t.Parallel()

	// Both message_id and message_ids provided — both should be included.
	params, err := parseArchiveParams([]byte(`{"message_id": 5, "message_ids": [1, 2]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params.MessageIDs) != 3 {
		t.Errorf("expected 3 IDs, got %d: %v", len(params.MessageIDs), params.MessageIDs)
	}
}

func TestArchiveEmail_Deduplication(t *testing.T) {
	t.Parallel()

	p := &archiveEmailParams{MessageIDs: []uint32{3, 1, 3, 2, 1}}
	if err := p.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.MessageIDs) != 3 {
		t.Errorf("expected 3 unique IDs after dedup, got %d: %v", len(p.MessageIDs), p.MessageIDs)
	}
	// Order preserved: 3, 1, 2
	expected := []uint32{3, 1, 2}
	for i, id := range p.MessageIDs {
		if id != expected[i] {
			t.Errorf("MessageIDs[%d] = %d, want %d", i, id, expected[i])
		}
	}
}

func TestArchiveEmail_ArchiveFolderRejectedViaSingleID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &archiveEmailAction{conn: conn}

	// Use the message_id shorthand with Archive folder.
	params, _ := json.Marshal(map[string]any{
		"message_id": 1,
		"folder":     "Archive",
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
