package protonmail

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestRemoveLabel_ExpungesLabelMailboxUIDs(t *testing.T) {
	origConnect := removeLabelConnectAndSelect
	origExpand := removeLabelExpandUIDsFn
	origFetch := removeLabelFetchEnvelopesFn
	origSelect := removeLabelSelectLabelMailbox
	origFind := removeLabelFindLabelUIDsFn
	origExpunge := removeLabelMarkDeletedAndExpunge
	t.Cleanup(func() {
		removeLabelConnectAndSelect = origConnect
		removeLabelExpandUIDsFn = origExpand
		removeLabelFetchEnvelopesFn = origFetch
		removeLabelSelectLabelMailbox = origSelect
		removeLabelFindLabelUIDsFn = origFind
		removeLabelMarkDeletedAndExpunge = origExpunge
	})

	removeLabelConnectAndSelect = func(connectors.Credentials, time.Duration, string, connectors.MailboxUIDValidityStore) (*imapSession, error) {
		return &imapSession{}, nil
	}
	removeLabelExpandUIDsFn = func(threadExpandSession, []uint32) ([]uint32, error) {
		return []uint32{140}, nil
	}
	removeLabelFetchEnvelopesFn = func(_ *imapSession, _ imap.UIDSet) ([]emailSummary, error) {
		return []emailSummary{{
			UID:             140,
			MessageIDHeader: "<msg-140@example.com>",
			Subject:         "Hello",
		}}, nil
	}
	removeLabelSelectLabelMailbox = func(_ *imapSession, labelMailbox string) error {
		if labelMailbox != "Labels/Work" {
			t.Fatalf("select label mailbox = %q, want Labels/Work", labelMailbox)
		}
		return nil
	}
	removeLabelFindLabelUIDsFn = func(_ *imapSession, summaries []emailSummary) ([]uint32, error) {
		if len(summaries) != 1 || summaries[0].MessageIDHeader != "<msg-140@example.com>" {
			t.Fatalf("unexpected summaries: %+v", summaries)
		}
		return []uint32{9001}, nil
	}

	var expungedUIDs []uint32
	removeLabelMarkDeletedAndExpunge = func(_ *imapSession, labelUIDs []uint32) error {
		expungedUIDs = labelUIDs
		return nil
	}

	conn := New()
	action := &removeLabelAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"message_id": 140,
		"label":      "Work",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.remove_label",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(expungedUIDs) != 1 || expungedUIDs[0] != 9001 {
		t.Fatalf("expunged UIDs = %v, want [9001]", expungedUIDs)
	}

	var payload map[string]any
	if err := json.Unmarshal(result.Data, &payload); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if payload["status"] != "label_removed" {
		t.Errorf("status = %v, want label_removed", payload["status"])
	}
}

func TestFindLabelMailboxUIDs_RequiresMessageID(t *testing.T) {
	t.Parallel()

	_, err := findLabelMailboxUIDs(&imapSession{}, []emailSummary{{
		UID: 1,
	}})
	if err == nil {
		t.Fatal("expected error for missing Message-ID")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestRemoveLabel_MissingLabel(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &removeLabelAction{conn: conn}
	params, _ := json.Marshal(map[string]any{"message_id": 1})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.remove_label",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing label")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestRemoveLabel_IncludeThreadFalseSkipsExpansion(t *testing.T) {
	origConnect := removeLabelConnectAndSelect
	origExpand := removeLabelExpandUIDsFn
	origFetch := removeLabelFetchEnvelopesFn
	origSelect := removeLabelSelectLabelMailbox
	origFind := removeLabelFindLabelUIDsFn
	origExpunge := removeLabelMarkDeletedAndExpunge
	t.Cleanup(func() {
		removeLabelConnectAndSelect = origConnect
		removeLabelExpandUIDsFn = origExpand
		removeLabelFetchEnvelopesFn = origFetch
		removeLabelSelectLabelMailbox = origSelect
		removeLabelFindLabelUIDsFn = origFind
		removeLabelMarkDeletedAndExpunge = origExpunge
	})

	removeLabelConnectAndSelect = func(connectors.Credentials, time.Duration, string, connectors.MailboxUIDValidityStore) (*imapSession, error) {
		return &imapSession{}, nil
	}
	removeLabelExpandUIDsFn = func(threadExpandSession, []uint32) ([]uint32, error) {
		t.Fatal("expandArchiveUIDs should not be called when include_thread is false")
		return nil, nil
	}

	var fetchedUIDs []uint32
	removeLabelFetchEnvelopesFn = func(_ *imapSession, uidSet imap.UIDSet) ([]emailSummary, error) {
		if nums, ok := uidSet.Nums(); ok {
			for _, uid := range nums {
				fetchedUIDs = append(fetchedUIDs, uint32(uid))
			}
		}
		return []emailSummary{{
			UID:             140,
			MessageIDHeader: "<msg-140@example.com>",
		}}, nil
	}
	removeLabelSelectLabelMailbox = func(_ *imapSession, _ string) error { return nil }
	removeLabelFindLabelUIDsFn = func(_ *imapSession, _ []emailSummary) ([]uint32, error) {
		return []uint32{9001}, nil
	}
	removeLabelMarkDeletedAndExpunge = func(_ *imapSession, _ []uint32) error { return nil }

	conn := New()
	action := &removeLabelAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"message_id":     140,
		"label":          "Work",
		"include_thread": false,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.remove_label",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fetchedUIDs) != 1 || fetchedUIDs[0] != 140 {
		t.Fatalf("fetched UIDs = %v, want [140]", fetchedUIDs)
	}
}
