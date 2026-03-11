package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
)

func decodeAuditEventList(t *testing.T, body []byte) auditEventListResponse {
	t.Helper()
	var resp auditEventListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal audit event list response: %v", err)
	}
	return resp
}

// ── GET /audit-events ───────────────────────────────────────────────────────

func TestListAuditEvents_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-events", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAuditEventList(t, w.Body.Bytes())
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 events, got %d", len(resp.Data))
	}
	if resp.HasMore {
		t.Error("expected has_more=false")
	}
}

func TestListAuditEvents_ReturnsEvents(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))
	testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-events", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAuditEventList(t, w.Body.Bytes())
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 events, got %d", len(resp.Data))
	}

	for _, e := range resp.Data {
		if e.EventType != "approval.approved" && e.EventType != "approval.denied" {
			t.Errorf("unexpected event type %q", e.EventType)
		}
		if e.AgentID != agentID {
			t.Errorf("expected agent_id %d, got %d", agentID, e.AgentID)
		}
	}
}

func TestListAuditEvents_WithFilters(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))
	testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Filter by outcome
	r := authenticatedRequest(t, http.MethodGet, "/audit-events?outcome=approved", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAuditEventList(t, w.Body.Bytes())
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 event with outcome=approved, got %d", len(resp.Data))
	}
	if resp.Data[0].Outcome != "approved" {
		t.Errorf("expected outcome 'approved', got %q", resp.Data[0].Outcome)
	}
}

func TestListAuditEvents_FilterByEventType(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))
	testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-events?event_type=approval.denied", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAuditEventList(t, w.Body.Bytes())
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 denied event, got %d", len(resp.Data))
	}
	if resp.Data[0].EventType != "approval.denied" {
		t.Errorf("expected event_type 'approval.denied', got %q", resp.Data[0].EventType)
	}
}

func TestListAuditEvents_FilterByAgentID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	agent1 := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	agent2 := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

	testhelper.InsertAuditEvent(t, tx, uid, agent1, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))
	testhelper.InsertAuditEvent(t, tx, uid, agent2, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-events?agent_id="+itoa64(agent1), uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAuditEventList(t, w.Body.Bytes())
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 event for agent1, got %d", len(resp.Data))
	}
	for _, e := range resp.Data {
		if e.AgentID != agent1 {
			t.Errorf("expected only agent %d, got %d", agent1, e.AgentID)
		}
	}
}

func TestListAuditEvents_Pagination(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		ts := base.Add(time.Duration(i) * time.Minute)
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), ts)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Page 1
	r := authenticatedRequest(t, http.MethodGet, "/audit-events?limit=2", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("page 1: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	page1 := decodeAuditEventList(t, w.Body.Bytes())
	if len(page1.Data) != 2 {
		t.Fatalf("page 1: expected 2 events, got %d", len(page1.Data))
	}
	if !page1.HasMore {
		t.Error("page 1: expected has_more=true")
	}
	if page1.NextCursor == nil {
		t.Fatal("page 1: expected non-nil next_cursor")
	}

	// Page 2
	r = authenticatedRequest(t, http.MethodGet, "/audit-events?limit=2&after="+*page1.NextCursor, uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("page 2: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	page2 := decodeAuditEventList(t, w.Body.Bytes())
	if len(page2.Data) != 2 {
		t.Fatalf("page 2: expected 2 events, got %d", len(page2.Data))
	}
	if !page2.HasMore {
		t.Error("page 2: expected has_more=true")
	}

	// Page 3
	r = authenticatedRequest(t, http.MethodGet, "/audit-events?limit=2&after="+*page2.NextCursor, uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("page 3: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	page3 := decodeAuditEventList(t, w.Body.Bytes())
	if len(page3.Data) != 1 {
		t.Fatalf("page 3: expected 1 event, got %d", len(page3.Data))
	}
	if page3.HasMore {
		t.Error("page 3: expected has_more=false")
	}
}

func TestListAuditEvents_InvalidLimit(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	for _, limit := range []string{"abc", "0", "-1", "101"} {
		r := authenticatedRequest(t, http.MethodGet, "/audit-events?limit="+limit, uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("limit=%q: expected 400, got %d", limit, w.Code)
		}
	}
}

func TestListAuditEvents_InvalidFilters(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	tests := []struct {
		name  string
		query string
	}{
		{"invalid agent_id", "agent_id=abc"},
		{"negative agent_id", "agent_id=-1"},
		{"invalid event_type", "event_type=invalid.type"},
		{"invalid outcome", "outcome=invalid"},
		{"invalid cursor", "after=not-a-timestamp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := authenticatedRequest(t, http.MethodGet, "/audit-events?"+tt.query, uid)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestListAuditEvents_Unauthenticated(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/audit-events", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListAuditEvents_ResponseStructure(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-events", uid)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify JSON structure
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	for _, key := range []string{"data", "has_more"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing key %q in response", key)
		}
	}

	resp := decodeAuditEventList(t, w.Body.Bytes())
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 event, got %d", len(resp.Data))
	}

	event := resp.Data[0]
	if event.EventType == "" {
		t.Error("expected non-empty event_type")
	}
	if event.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
	if event.AgentID == 0 {
		t.Error("expected non-zero agent_id")
	}
	if event.Outcome == "" {
		t.Error("expected non-empty outcome")
	}
	if event.Action == nil {
		t.Error("expected non-nil action for approval event")
	}
}

// ── GET /audit-logs ───────────────────────────────────────────────────────

func decodeAuditLogExport(t *testing.T, body []byte) auditLogExportResponse {
	t.Helper()
	var resp auditLogExportResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to unmarshal audit log export response: %v", err)
	}
	return resp
}

func TestExportAuditLogs_RequiresSince(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-logs", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without since param, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExportAuditLogs_InvalidSince(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-logs?since=not-a-timestamp", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid since, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExportAuditLogs_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-logs?since=2026-01-01T00:00:00Z", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAuditLogExport(t, w.Body.Bytes())
	if len(resp.Data) != 0 {
		t.Errorf("expected 0 events, got %d", len(resp.Data))
	}
	if resp.HasMore {
		t.Error("expected has_more=false")
	}
}

func TestExportAuditLogs_FiltersBySince(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	// Insert events at different times.
	old := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	recent := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), old)
	testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"), recent)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// since=2026-02-01 should only return the recent event.
	r := authenticatedRequest(t, http.MethodGet, "/audit-logs?since=2026-02-01T00:00:00Z", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAuditLogExport(t, w.Body.Bytes())
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 event after since filter, got %d", len(resp.Data))
	}
	if resp.Data[0].EventType != "approval.denied" {
		t.Errorf("expected event_type 'approval.denied', got %q", resp.Data[0].EventType)
	}
}

func TestExportAuditLogs_ChronologicalOrder(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		ts := base.Add(time.Duration(i) * time.Minute)
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), ts)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-logs?since=2026-01-01T00:00:00Z", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAuditLogExport(t, w.Body.Bytes())
	if len(resp.Data) != 3 {
		t.Fatalf("expected 3 events, got %d", len(resp.Data))
	}

	// Verify chronological order (oldest first).
	for i := 1; i < len(resp.Data); i++ {
		prev := resp.Data[i-1]
		curr := resp.Data[i]
		if curr.Timestamp.Before(prev.Timestamp) {
			t.Errorf("events not in chronological order: event[%d]=%v before event[%d]=%v",
				i, curr.Timestamp, i-1, prev.Timestamp)
		}
	}
}

func TestExportAuditLogs_Pagination(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	base := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		ts := base.Add(time.Duration(i) * time.Minute)
		testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), ts)
	}

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// Page 1
	r := authenticatedRequest(t, http.MethodGet, "/audit-logs?since=2026-01-01T00:00:00Z&limit=2", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("page 1: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	page1 := decodeAuditLogExport(t, w.Body.Bytes())
	if len(page1.Data) != 2 {
		t.Fatalf("page 1: expected 2 events, got %d", len(page1.Data))
	}
	if !page1.HasMore {
		t.Error("page 1: expected has_more=true")
	}
	if page1.NextCursor == nil {
		t.Fatal("page 1: expected non-nil next_cursor")
	}

	// Page 2
	r = authenticatedRequest(t, http.MethodGet, "/audit-logs?since=2026-01-01T00:00:00Z&limit=2&after="+*page1.NextCursor, uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("page 2: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	page2 := decodeAuditLogExport(t, w.Body.Bytes())
	if len(page2.Data) != 2 {
		t.Fatalf("page 2: expected 2 events, got %d", len(page2.Data))
	}
	if !page2.HasMore {
		t.Error("page 2: expected has_more=true")
	}

	// Page 3 (last page)
	r = authenticatedRequest(t, http.MethodGet, "/audit-logs?since=2026-01-01T00:00:00Z&limit=2&after="+*page2.NextCursor, uid)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("page 3: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	page3 := decodeAuditLogExport(t, w.Body.Bytes())
	if len(page3.Data) != 1 {
		t.Fatalf("page 3: expected 1 event, got %d", len(page3.Data))
	}
	if page3.HasMore {
		t.Error("page 3: expected has_more=false")
	}
}

func TestExportAuditLogs_InvalidLimit(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	for _, limit := range []string{"abc", "0", "-1", "1001"} {
		r := authenticatedRequest(t, http.MethodGet, "/audit-logs?since=2026-01-01T00:00:00Z&limit="+limit, uid)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		if w.Code != http.StatusBadRequest {
			t.Errorf("limit=%q: expected 400, got %d", limit, w.Code)
		}
	}
}

func TestExportAuditLogs_Unauthenticated(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/audit-logs?since=2026-01-01T00:00:00Z", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestExportAuditLogs_DoesNotLeakOtherUsers(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	agentID1 := testhelper.InsertUserWithAgent(t, tx, uid1, "u_"+uid1[:8])
	agentID2 := testhelper.InsertUserWithAgent(t, tx, uid2, "u_"+uid2[:8])

	testhelper.InsertAuditEvent(t, tx, uid1, agentID1, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))
	testhelper.InsertAuditEvent(t, tx, uid2, agentID2, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-logs?since=2020-01-01T00:00:00Z", uid1)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAuditLogExport(t, w.Body.Bytes())
	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 event (only user1's), got %d", len(resp.Data))
	}
	if resp.Data[0].Outcome != "approved" {
		t.Errorf("expected user1's approved event, got outcome=%q", resp.Data[0].Outcome)
	}
}

func TestExportAuditLogs_IncludesIDAndSourceID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-logs?since=2020-01-01T00:00:00Z", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify raw JSON includes id and source_id fields
	var raw struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(raw.Data) != 1 {
		t.Fatalf("expected 1 event, got %d", len(raw.Data))
	}
	if _, ok := raw.Data[0]["id"]; !ok {
		t.Error("export response missing 'id' field")
	}
	if _, ok := raw.Data[0]["source_id"]; !ok {
		t.Error("export response missing 'source_id' field")
	}
}

func TestExportAuditLogs_UntilFilter(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	jan := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	feb := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	mar := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"), jan)
	testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"), feb)
	testhelper.InsertAuditEventAt(t, tx, uid, agentID, "approval.cancelled", "cancelled", testhelper.GenerateID(t, "appr_"), mar)

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// since=jan, until=mar should return jan and feb events
	r := authenticatedRequest(t, http.MethodGet, "/audit-logs?since=2026-01-01T00:00:00Z&until=2026-03-01T00:00:00Z", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAuditLogExport(t, w.Body.Bytes())
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 events within [jan, mar), got %d", len(resp.Data))
	}
}

func TestExportAuditLogs_UntilBeforeSince(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-logs?since=2026-03-01T00:00:00Z&until=2026-01-01T00:00:00Z", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 when until < since, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExportAuditLogs_EventTypeFilter(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))
	testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.denied", "denied", testhelper.GenerateID(t, "appr_"))
	testhelper.InsertAuditEvent(t, tx, uid, agentID, "agent.registered", "registered", "ar:"+itoa64(agentID))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-logs?since=2020-01-01T00:00:00Z&event_type=approval.approved,approval.denied", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeAuditLogExport(t, w.Body.Bytes())
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 approval events, got %d", len(resp.Data))
	}
	for _, e := range resp.Data {
		if e.EventType != "approval.approved" && e.EventType != "approval.denied" {
			t.Errorf("unexpected event_type %q", e.EventType)
		}
	}
}

func TestExportAuditLogs_InvalidEventType(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/audit-logs?since=2020-01-01T00:00:00Z&event_type=bogus.type", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid event_type, got %d: %s", w.Code, w.Body.String())
	}
}

func TestExportAuditLogs_ActivityFeedOmitsID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	testhelper.InsertAuditEvent(t, tx, uid, agentID, "approval.approved", "approved", testhelper.GenerateID(t, "appr_"))

	deps := &Deps{DB: tx, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	// The activity feed (GET /audit-events) should NOT include id but SHOULD
	// include source_id (added for click-through detail views).
	r := authenticatedRequest(t, http.MethodGet, "/audit-events", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var raw struct {
		Data []map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if len(raw.Data) != 1 {
		t.Fatalf("expected 1 event, got %d", len(raw.Data))
	}
	if _, ok := raw.Data[0]["id"]; ok {
		t.Error("activity feed response should NOT include 'id' field")
	}
	if _, ok := raw.Data[0]["source_id"]; !ok {
		t.Error("activity feed response should include 'source_id' field")
	}
}

// itoa64 is a helper to convert int64 to string for URL query params.
func itoa64(n int64) string {
	return strconv.FormatInt(n, 10)
}
