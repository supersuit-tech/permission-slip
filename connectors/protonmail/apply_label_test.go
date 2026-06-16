package protonmail

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestApplyLabel_ThreadExpansionCopiesAllUIDs(t *testing.T) {
	origConnect := applyLabelConnectAndSelect
	origExpand := applyLabelExpandUIDsFn
	origCopy := applyLabelPerformCopy
	t.Cleanup(func() {
		applyLabelConnectAndSelect = origConnect
		applyLabelExpandUIDsFn = origExpand
		applyLabelPerformCopy = origCopy
	})

	applyLabelConnectAndSelect = func(connectors.Credentials, time.Duration, string, connectors.MailboxUIDValidityStore) (*imapSession, error) {
		return &imapSession{}, nil
	}
	applyLabelExpandUIDsFn = func(threadExpandSession, []uint32) ([]uint32, error) {
		return []uint32{125, 132, 140}, nil
	}

	var copiedUIDs []uint32
	var copiedLabel string
	applyLabelPerformCopy = func(_ *imapSession, uids []uint32, labelMailbox string) error {
		copiedUIDs = uids
		copiedLabel = labelMailbox
		return nil
	}

	conn := New()
	action := &applyLabelAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"message_id": 140,
		"label":      "Work",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.apply_label",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if copiedLabel != "Labels/Work" {
		t.Errorf("copy dest = %q, want Labels/Work", copiedLabel)
	}
	if len(copiedUIDs) != 3 {
		t.Fatalf("copied %d UIDs, want 3: %v", len(copiedUIDs), copiedUIDs)
	}

	var payload map[string]any
	if err := json.Unmarshal(result.Data, &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload["thread_expanded"] != true {
		t.Errorf("thread_expanded = %v, want true", payload["thread_expanded"])
	}
}

func TestApplyLabel_IncludeThreadFalseSkipsExpansion(t *testing.T) {
	origConnect := applyLabelConnectAndSelect
	origExpand := applyLabelExpandUIDsFn
	origCopy := applyLabelPerformCopy
	t.Cleanup(func() {
		applyLabelConnectAndSelect = origConnect
		applyLabelExpandUIDsFn = origExpand
		applyLabelPerformCopy = origCopy
	})

	applyLabelConnectAndSelect = func(connectors.Credentials, time.Duration, string, connectors.MailboxUIDValidityStore) (*imapSession, error) {
		return &imapSession{}, nil
	}
	applyLabelExpandUIDsFn = func(threadExpandSession, []uint32) ([]uint32, error) {
		t.Fatal("expandArchiveUIDs should not be called when include_thread is false")
		return nil, nil
	}

	var copiedUIDs []uint32
	applyLabelPerformCopy = func(_ *imapSession, uids []uint32, _ string) error {
		copiedUIDs = uids
		return nil
	}

	conn := New()
	action := &applyLabelAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"message_id":     140,
		"label":          "Work",
		"include_thread": false,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.apply_label",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(copiedUIDs) != 1 || copiedUIDs[0] != 140 {
		t.Fatalf("copied UIDs = %v, want [140]", copiedUIDs)
	}
}

func TestApplyLabel_MissingMessageIDs(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &applyLabelAction{conn: conn}
	params, _ := json.Marshal(map[string]any{"label": "Work"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.apply_label",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing message_ids")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}
