package protonmail

import (
	"reflect"
	"testing"

	"github.com/emersion/go-imap/v2"
)

type fakeThreadExpandSession struct {
	byUID     map[uint32]emailSummary
	searchHit map[string][]imap.UID
}

func (f *fakeThreadExpandSession) fetchEnvelopes(uidSet imap.UIDSet) ([]emailSummary, error) {
	var out []emailSummary
	for uid, summary := range f.byUID {
		if uidSet.Contains(imap.UID(uid)) {
			out = append(out, summary)
		}
	}
	return out, nil
}

func (f *fakeThreadExpandSession) searchUIDsBySubject(subject string) ([]imap.UID, error) {
	return f.searchHit[subject], nil
}

func TestIncludeThreadEnabled(t *testing.T) {
	t.Parallel()

	if !includeThreadEnabled(nil) {
		t.Error("expected nil (omitted) to default to enabled")
	}
	enabled := true
	if !includeThreadEnabled(&enabled) {
		t.Error("expected explicit true to be enabled")
	}
	disabled := false
	if includeThreadEnabled(&disabled) {
		t.Error("expected explicit false to be disabled")
	}
}

func TestUidsInSameThreadsAs_SubstringFalsePositiveExcluded(t *testing.T) {
	t.Parallel()

	// SUBJECT search can return "legacy media litigation update" as a substring
	// hit for "legacy media litigation", but normalized subjects differ and
	// there is no header link — they must stay separate.
	emails := []emailSummary{
		summaryFixture(10, "RE: RE: Legacy Media litigation", "2026-06-01T10:00:00Z", "a@example.com"),
		summaryFixture(20, "Legacy Media litigation update", "2026-06-02T10:00:00Z", "b@example.com"),
	}

	got := uidsInSameThreadsAs(emails, []uint32{10})
	if want := []uint32{10}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUidsInSameThreadsAs_ReplyChainGrouped(t *testing.T) {
	t.Parallel()

	emails := []emailSummary{
		summaryFixture(125, "Legacy Media litigation", "2026-06-01T10:00:00Z", "a@example.com"),
		summaryFixture(132, "RE: Legacy Media litigation", "2026-06-02T10:00:00Z", "b@example.com", "a@example.com"),
		summaryFixture(140, "RE: RE: Legacy Media litigation", "2026-06-03T10:00:00Z", "c@example.com", "b@example.com"),
	}

	got := uidsInSameThreadsAs(emails, []uint32{140})
	if want := []uint32{125, 132, 140}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestUidsInSameThreadsAs_EmptySubjectTargetAlone(t *testing.T) {
	t.Parallel()

	emails := []emailSummary{
		summaryFixture(1, "", "2026-06-01T10:00:00Z", "a@example.com"),
		summaryFixture(2, "", "2026-06-02T10:00:00Z", "b@example.com"),
	}

	got := uidsInSameThreadsAs(emails, []uint32{1})
	if want := []uint32{1}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExpandArchiveUIDs_ReChainViaSubjectSearch(t *testing.T) {
	t.Parallel()

	fake := &fakeThreadExpandSession{
		byUID: map[uint32]emailSummary{
			125: summaryFixture(125, "Legacy Media litigation", "2026-06-01T10:00:00Z", "a@example.com"),
			132: summaryFixture(132, "RE: Legacy Media litigation", "2026-06-02T10:00:00Z", "b@example.com", "a@example.com"),
			140: summaryFixture(140, "RE: RE: Legacy Media litigation", "2026-06-03T10:00:00Z", "c@example.com", "b@example.com"),
		},
		searchHit: map[string][]imap.UID{
			"legacy media litigation": {125, 132, 140},
		},
	}

	got, err := expandArchiveUIDs(fake, []uint32{140})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []uint32{125, 132, 140}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExpandArchiveUIDs_EmptySubjectSkipsSearch(t *testing.T) {
	t.Parallel()

	fake := &fakeThreadExpandSession{
		byUID: map[uint32]emailSummary{
			5: summaryFixture(5, "", "2026-06-01T10:00:00Z", "a@example.com"),
		},
		searchHit: map[string][]imap.UID{
			"": {5, 6, 7},
		},
	}

	got, err := expandArchiveUIDs(fake, []uint32{5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []uint32{5}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExpandArchiveUIDs_SubstringFalsePositiveExcluded(t *testing.T) {
	t.Parallel()

	fake := &fakeThreadExpandSession{
		byUID: map[uint32]emailSummary{
			10: summaryFixture(10, "RE: RE: Legacy Media litigation", "2026-06-01T10:00:00Z", "a@example.com"),
			20: summaryFixture(20, "Legacy Media litigation update", "2026-06-02T10:00:00Z", "b@example.com"),
		},
		searchHit: map[string][]imap.UID{
			"legacy media litigation": {10, 20},
		},
	}

	got, err := expandArchiveUIDs(fake, []uint32{10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if want := []uint32{10}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
