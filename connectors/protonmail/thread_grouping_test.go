package protonmail

import (
	"encoding/json"
	"reflect"
	"testing"
)

// summaryFixture builds an emailSummary for grouping tests. inReplyTo may be
// empty for thread roots or messages with stripped headers.
func summaryFixture(uid uint32, subject, date, messageID string, inReplyTo ...string) emailSummary {
	return emailSummary{
		UID:             uid,
		Subject:         subject,
		Date:            date,
		MessageIDHeader: messageID,
		InReplyTo:       inReplyTo,
	}
}

func threadByUID(t *testing.T, threads []emailSummary, uid uint32) emailSummary {
	t.Helper()
	for _, thread := range threads {
		if thread.UID == uid {
			return thread
		}
	}
	t.Fatalf("no thread with canonical UID %d in %+v", uid, threads)
	return emailSummary{}
}

func TestGroupIntoThreads_LinearReplyChain(t *testing.T) {
	t.Parallel()

	emails := []emailSummary{
		summaryFixture(33, "Project update", "2026-06-01T10:00:00Z", "a@example.com"),
		summaryFixture(40, "RE: Project update", "2026-06-02T10:00:00Z", "b@example.com", "a@example.com"),
		summaryFixture(146, "RE: RE: Project update", "2026-06-03T10:00:00Z", "c@example.com", "b@example.com"),
	}

	threads := groupIntoThreads(emails)
	if len(threads) != 1 {
		t.Fatalf("expected 1 thread, got %d: %+v", len(threads), threads)
	}
	thread := threads[0]
	if thread.UID != 146 {
		t.Errorf("expected latest message UID 146 as canonical, got %d", thread.UID)
	}
	if thread.ThreadSize != 3 {
		t.Errorf("expected thread_size 3, got %d", thread.ThreadSize)
	}
	if want := []uint32{33, 40, 146}; !reflect.DeepEqual(thread.ThreadUIDs, want) {
		t.Errorf("expected thread_uids %v, got %v", want, thread.ThreadUIDs)
	}
}

func TestGroupIntoThreads_SubjectFallbackBridgesGaps(t *testing.T) {
	t.Parallel()

	// The middle of the chain was not fetched: neither email references the
	// other directly, so only the normalized subject links them.
	emails := []emailSummary{
		summaryFixture(10, "Project update", "2026-06-01T10:00:00Z", "a@example.com"),
		summaryFixture(50, "RE: RE: Project update", "2026-06-03T10:00:00Z", "c@example.com", "missing@example.com"),
	}

	threads := groupIntoThreads(emails)
	if len(threads) != 1 {
		t.Fatalf("expected 1 thread, got %d: %+v", len(threads), threads)
	}
	if threads[0].UID != 50 {
		t.Errorf("expected canonical UID 50, got %d", threads[0].UID)
	}
	if want := []uint32{10, 50}; !reflect.DeepEqual(threads[0].ThreadUIDs, want) {
		t.Errorf("expected thread_uids %v, got %v", want, threads[0].ThreadUIDs)
	}
}

func TestGroupIntoThreads_DistinctSubjectsStaySeparate(t *testing.T) {
	t.Parallel()

	emails := []emailSummary{
		summaryFixture(1, "Invoice", "2026-06-01T10:00:00Z", "a@example.com"),
		summaryFixture(2, "Lunch plans", "2026-06-02T10:00:00Z", "b@example.com"),
	}

	threads := groupIntoThreads(emails)
	if len(threads) != 2 {
		t.Fatalf("expected 2 threads, got %d: %+v", len(threads), threads)
	}
	for _, thread := range threads {
		if thread.ThreadSize != 1 {
			t.Errorf("UID %d: expected thread_size 1, got %d", thread.UID, thread.ThreadSize)
		}
		if want := []uint32{thread.UID}; !reflect.DeepEqual(thread.ThreadUIDs, want) {
			t.Errorf("UID %d: expected thread_uids %v, got %v", thread.UID, want, thread.ThreadUIDs)
		}
	}
}

func TestGroupIntoThreads_EmptySubjectsNeverMergedByFallback(t *testing.T) {
	t.Parallel()

	emails := []emailSummary{
		summaryFixture(1, "", "2026-06-01T10:00:00Z", "a@example.com"),
		summaryFixture(2, "", "2026-06-02T10:00:00Z", "b@example.com"),
		summaryFixture(3, "Re:", "2026-06-03T10:00:00Z", "c@example.com"),
	}

	threads := groupIntoThreads(emails)
	if len(threads) != 3 {
		t.Fatalf("expected 3 threads (empty subjects never merge), got %d: %+v", len(threads), threads)
	}
}

func TestGroupIntoThreads_LatestByDateWithUIDTieBreak(t *testing.T) {
	t.Parallel()

	emails := []emailSummary{
		summaryFixture(5, "Topic", "2026-06-02T10:00:00Z", "a@example.com"),
		summaryFixture(9, "Re: Topic", "2026-06-02T10:00:00Z", "b@example.com", "a@example.com"),
		summaryFixture(7, "Re: Topic", "2026-06-01T10:00:00Z", "c@example.com", "a@example.com"),
	}

	threads := groupIntoThreads(emails)
	if len(threads) != 1 {
		t.Fatalf("expected 1 thread, got %d: %+v", len(threads), threads)
	}
	// UIDs 5 and 9 share a date; the higher UID wins the tie.
	if threads[0].UID != 9 {
		t.Errorf("expected canonical UID 9 (date tie broken by UID), got %d", threads[0].UID)
	}
}

func TestGroupIntoThreads_MostRecentThreadLast(t *testing.T) {
	t.Parallel()

	emails := []emailSummary{
		summaryFixture(4, "Newest topic", "2026-06-05T10:00:00Z", "d@example.com"),
		summaryFixture(1, "Old topic", "2026-06-01T10:00:00Z", "a@example.com"),
		summaryFixture(2, "Re: Old topic", "2026-06-02T10:00:00Z", "b@example.com", "a@example.com"),
		summaryFixture(3, "Middle topic", "2026-06-03T10:00:00Z", "c@example.com"),
	}

	threads := groupIntoThreads(emails)
	if len(threads) != 3 {
		t.Fatalf("expected 3 threads, got %d: %+v", len(threads), threads)
	}
	var order []uint32
	for _, thread := range threads {
		order = append(order, thread.UID)
	}
	if want := []uint32{2, 3, 4}; !reflect.DeepEqual(order, want) {
		t.Errorf("expected most-recent-last order %v, got %v", want, order)
	}
}

func TestGroupIntoThreads_SingleAndEmptyInput(t *testing.T) {
	t.Parallel()

	if got := groupIntoThreads(nil); len(got) != 0 {
		t.Errorf("expected empty result for nil input, got %+v", got)
	}

	threads := groupIntoThreads([]emailSummary{
		summaryFixture(42, "Solo", "2026-06-01T10:00:00Z", "a@example.com"),
	})
	if len(threads) != 1 {
		t.Fatalf("expected 1 thread, got %d", len(threads))
	}
	if threads[0].ThreadSize != 1 {
		t.Errorf("expected thread_size 1, got %d", threads[0].ThreadSize)
	}
	if want := []uint32{42}; !reflect.DeepEqual(threads[0].ThreadUIDs, want) {
		t.Errorf("expected thread_uids %v, got %v", want, threads[0].ThreadUIDs)
	}
}

func TestNormalizeSubject(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in   string
		want string
	}{
		{"Project update", "project update"},
		{"RE: Project update", "project update"},
		{"Re: re: RE: Project update", "project update"},
		{"Fwd: Project update", "project update"},
		{"FW: Re: Project update", "project update"},
		{"  Re:   Project update  ", "project update"},
		{"", ""},
		{"Re:", ""},
		{"Regarding the project", "regarding the project"},
	}
	for _, tc := range cases {
		if got := normalizeSubject(tc.in); got != tc.want {
			t.Errorf("normalizeSubject(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestGroupByThreadEnabled(t *testing.T) {
	t.Parallel()

	if !groupByThreadEnabled(nil) {
		t.Error("expected nil (omitted) to default to enabled")
	}
	enabled := true
	if !groupByThreadEnabled(&enabled) {
		t.Error("expected explicit true to be enabled")
	}
	disabled := false
	if groupByThreadEnabled(&disabled) {
		t.Error("expected explicit false to be disabled")
	}
}

func TestGroupByThreadParam_DefaultsAndExplicitFalse(t *testing.T) {
	t.Parallel()

	var inbox readInboxParams
	if err := json.Unmarshal([]byte(`{}`), &inbox); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !groupByThreadEnabled(inbox.GroupByThread) {
		t.Error("read_inbox: expected omitted group_by_thread to default to enabled")
	}

	var search searchEmailsParams
	if err := json.Unmarshal([]byte(`{"subject":"x","group_by_thread":false}`), &search); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if groupByThreadEnabled(search.GroupByThread) {
		t.Error("search_emails: expected explicit group_by_thread=false to disable grouping")
	}
}
