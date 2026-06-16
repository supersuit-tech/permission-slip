package protonmail

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestParseLabelMessageParams(t *testing.T) {
	t.Parallel()

	params, err := parseLabelMessageParams([]byte(`{"message_id": 5, "label": "Work"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(params.MessageIDs) != 1 || params.MessageIDs[0] != 5 {
		t.Errorf("MessageIDs = %v, want [5]", params.MessageIDs)
	}
	if params.Label != "Work" {
		t.Errorf("Label = %q, want Work", params.Label)
	}
	if !includeThreadEnabled(params.IncludeThread) {
		t.Error("expected omitted include_thread to default to enabled")
	}

	params, err = parseLabelMessageParams([]byte(`{"message_id": 1, "label": "Work", "include_thread": false}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if includeThreadEnabled(params.IncludeThread) {
		t.Error("expected explicit include_thread=false to disable expansion")
	}
}

func TestLabelMessageParams_Validate(t *testing.T) {
	t.Parallel()

	p := &labelMessageParams{
		MessageIDs: []uint32{1},
		Label:      "Work",
	}
	if err := p.validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Folder != "INBOX" {
		t.Errorf("Folder = %q, want INBOX", p.Folder)
	}
	if p.LabelMailbox != "Labels/Work" {
		t.Errorf("LabelMailbox = %q, want Labels/Work", p.LabelMailbox)
	}
}

func TestLabelMessageParams_MissingLabel(t *testing.T) {
	t.Parallel()

	p := &labelMessageParams{MessageIDs: []uint32{1}}
	err := p.validate()
	if err == nil {
		t.Fatal("expected error for missing label")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestLabelMessageParams_SameFolderAndLabelRejected(t *testing.T) {
	t.Parallel()

	p := &labelMessageParams{
		MessageIDs: []uint32{1},
		Folder:     "Labels/Work",
		Label:      "Work",
	}
	err := p.validate()
	if err == nil {
		t.Fatal("expected validation error when source folder equals label mailbox")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}
