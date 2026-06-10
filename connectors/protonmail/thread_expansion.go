package protonmail

import (
	"sort"

	"github.com/emersion/go-imap/v2"
)

// includeThreadEnabled interprets the optional include_thread parameter.
// A nil value means the parameter was omitted, which defaults to enabled.
func includeThreadEnabled(v *bool) bool {
	return v == nil || *v
}

// threadExpandSession is the IMAP surface needed to expand archive targets to
// their full conversations. Tests provide a fake implementation.
type threadExpandSession interface {
	fetchEnvelopes(uidSet imap.UIDSet) ([]emailSummary, error)
	searchUIDsBySubject(subject string) ([]imap.UID, error)
}

func (s *imapSession) fetchEnvelopes(uidSet imap.UIDSet) ([]emailSummary, error) {
	return fetchEnvelopesByUID(s, uidSet)
}

func (s *imapSession) searchUIDsBySubject(subject string) ([]imap.UID, error) {
	if subject == "" {
		return nil, nil
	}
	criteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{{
			Key:   "SUBJECT",
			Value: subject,
		}},
	}
	searchData, err := s.client.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, mapIMAPError(err)
	}
	return searchData.AllUIDs(), nil
}

// expandArchiveUIDs widens the requested UIDs to every message in their
// conversations within the selected folder. Targets with an empty normalized
// subject are never subject-matched and stay as single-message archives.
func expandArchiveUIDs(session threadExpandSession, targetUIDs []uint32) ([]uint32, error) {
	if len(targetUIDs) == 0 {
		return nil, nil
	}

	targetSet := uidSetFromMessageIDs(targetUIDs)
	targets, err := session.fetchEnvelopes(targetSet)
	if err != nil {
		return nil, err
	}

	byUID := make(map[uint32]emailSummary, len(targets))
	for _, e := range targets {
		byUID[e.UID] = e
	}

	candidateUIDs := make(map[uint32]struct{})
	for _, uid := range targetUIDs {
		summary, ok := byUID[uid]
		if !ok {
			continue
		}
		norm := normalizeSubject(summary.Subject)
		if norm == "" {
			continue
		}
		matches, err := session.searchUIDsBySubject(norm)
		if err != nil {
			return nil, err
		}
		for _, matchUID := range matches {
			candidateUIDs[uint32(matchUID)] = struct{}{}
		}
	}

	var toFetch imap.UIDSet
	for uid := range candidateUIDs {
		if _, already := byUID[uid]; already {
			continue
		}
		toFetch.AddNum(imap.UID(uid))
	}

	if len(toFetch) > 0 {
		candidates, err := session.fetchEnvelopes(toFetch)
		if err != nil {
			return nil, err
		}
		for _, e := range candidates {
			byUID[e.UID] = e
		}
	}

	allEmails := make([]emailSummary, 0, len(byUID))
	for _, e := range byUID {
		allEmails = append(allEmails, e)
	}

	return uidsInSameThreadsAs(allEmails, targetUIDs), nil
}

// threadUnionFind groups email summaries by conversation using the same
// In-Reply-To and normalized-subject rules as groupIntoThreads.
type threadUnionFind struct {
	parent []int
}

func buildThreadUnionFind(emails []emailSummary) threadUnionFind {
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

	return threadUnionFind{parent: parent}
}

func (uf threadUnionFind) find(i int) int {
	if uf.parent[i] != i {
		uf.parent[i] = uf.find(uf.parent[i])
	}
	return uf.parent[i]
}

// uidsInSameThreadsAs returns every UID that shares a conversation group with
// at least one of the target UIDs. Results are sorted ascending.
func uidsInSameThreadsAs(emails []emailSummary, targetUIDs []uint32) []uint32 {
	if len(emails) == 0 || len(targetUIDs) == 0 {
		return deduplicateUint32(targetUIDs)
	}

	uf := buildThreadUnionFind(emails)

	uidToIndex := make(map[uint32]int, len(emails))
	for i, e := range emails {
		uidToIndex[e.UID] = i
	}

	targetRoots := make(map[int]struct{})
	for _, uid := range targetUIDs {
		if idx, ok := uidToIndex[uid]; ok {
			targetRoots[uf.find(idx)] = struct{}{}
		}
	}

	// Targets missing from the fetched set are still archived as requested.
	outSet := make(map[uint32]struct{}, len(targetUIDs))
	for _, uid := range targetUIDs {
		outSet[uid] = struct{}{}
	}

	for i, e := range emails {
		if _, ok := targetRoots[uf.find(i)]; ok {
			outSet[e.UID] = struct{}{}
		}
	}

	out := make([]uint32, 0, len(outSet))
	for uid := range outSet {
		out = append(out, uid)
	}
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out
}
