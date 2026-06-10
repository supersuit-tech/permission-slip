package protonmail

import (
	"sort"
	"strings"
	"time"
)

// groupByThreadEnabled interprets the optional group_by_thread parameter.
// A nil value means the parameter was omitted, which defaults to enabled.
func groupByThreadEnabled(v *bool) bool {
	return v == nil || *v
}

// groupIntoThreads collapses a flat list of email summaries into one entry per
// conversation: the latest message of each thread, annotated with ThreadSize
// and ThreadUIDs so agents can still reach earlier messages.
//
// Grouping happens in two passes over the fetched window:
//  1. Header pass: an email is joined to the email whose Message-ID appears in
//     its In-Reply-To header. The IMAP ENVELOPE carries both fields, so this
//     needs no extra fetches. ENVELOPE does not include References, so a chain
//     whose intermediate parents fall outside the window can break here.
//  2. Subject fallback pass: groups sharing a normalized subject (Re:/Fwd:
//     prefixes stripped) are joined to bridge those gaps. This can merge two
//     genuinely distinct conversations that share a subject line — an accepted
//     tradeoff for a listing view. Emails with empty normalized subjects are
//     never merged by this pass.
//
// Thread counts only cover the fetched window: a long thread whose older
// messages were not fetched is partially represented.
func groupIntoThreads(emails []emailSummary) []emailSummary {
	if len(emails) <= 1 {
		if len(emails) == 1 {
			annotated := emails[0]
			annotated.ThreadSize = 1
			annotated.ThreadUIDs = []uint32{annotated.UID}
			return []emailSummary{annotated}
		}
		return emails
	}

	parent := make([]int, len(emails))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}

	// Header pass: link replies to the message they answer.
	byMessageID := make(map[string]int, len(emails))
	for i, e := range emails {
		if e.MessageIDHeader != "" {
			byMessageID[e.MessageIDHeader] = i
		}
	}
	for i, e := range emails {
		for _, ref := range e.InReplyTo {
			if j, ok := byMessageID[ref]; ok {
				union(i, j)
			}
		}
	}

	// Subject fallback pass: bridge chains broken by missing parents.
	bySubject := make(map[string]int, len(emails))
	for i, e := range emails {
		subject := normalizeSubject(e.Subject)
		if subject == "" {
			continue
		}
		if j, ok := bySubject[subject]; ok {
			union(i, j)
		} else {
			bySubject[subject] = i
		}
	}

	groups := make(map[int][]int)
	for i := range emails {
		root := find(i)
		groups[root] = append(groups[root], i)
	}

	threads := make([]emailSummary, 0, len(groups))
	for _, members := range groups {
		canonical := members[0]
		for _, m := range members[1:] {
			if summaryNewer(emails[m], emails[canonical]) {
				canonical = m
			}
		}
		thread := emails[canonical]
		thread.ThreadSize = len(members)
		thread.ThreadUIDs = make([]uint32, 0, len(members))
		for _, m := range members {
			thread.ThreadUIDs = append(thread.ThreadUIDs, emails[m].UID)
		}
		sort.Slice(thread.ThreadUIDs, func(a, b int) bool {
			return thread.ThreadUIDs[a] < thread.ThreadUIDs[b]
		})
		threads = append(threads, thread)
	}

	// Most recent last, matching the flat listing order.
	sort.Slice(threads, func(a, b int) bool {
		return summaryNewer(threads[b], threads[a])
	})
	return threads
}

// summaryNewer reports whether a is more recent than b, comparing envelope
// dates with UID as tie-break (higher UID = later delivery).
func summaryNewer(a, b emailSummary) bool {
	ta := parseSummaryDate(a.Date)
	tb := parseSummaryDate(b.Date)
	if !ta.Equal(tb) {
		return ta.After(tb)
	}
	return a.UID > b.UID
}

// parseSummaryDate parses the RFC 3339 date stored on summaries. Unparseable
// dates compare as the zero time, i.e. older than everything.
func parseSummaryDate(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// subjectPrefixes are reply/forward markers stripped during normalization.
var subjectPrefixes = []string{"re:", "fwd:", "fw:"}

// normalizeSubject strips reply/forward prefixes and lowercases the subject so
// "RE: RE: Project update" and "Project update" group together.
func normalizeSubject(subject string) string {
	s := strings.ToLower(strings.TrimSpace(subject))
	for {
		stripped := false
		for _, prefix := range subjectPrefixes {
			if strings.HasPrefix(s, prefix) {
				s = strings.TrimSpace(s[len(prefix):])
				stripped = true
			}
		}
		if !stripped {
			return s
		}
	}
}
