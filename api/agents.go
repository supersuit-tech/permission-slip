package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

type agentResponse struct {
	AgentID          int64      `json:"agent_id"`
	Status           string     `json:"status"`
	Metadata         any        `json:"metadata,omitempty"`
	ConfirmationCode *string    `json:"confirmation_code,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	RegisteredAt     *time.Time `json:"registered_at,omitempty"`
	DeactivatedAt    *time.Time `json:"deactivated_at,omitempty"`
	LastActiveAt     *time.Time `json:"last_active_at,omitempty"`
	RequestCount30d  *int       `json:"request_count_30d,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type agentListResponse struct {
	Data       []agentResponse `json:"data"`
	HasMore    bool            `json:"has_more"`
	NextCursor *string         `json:"next_cursor,omitempty"`
}

func init() {
	RegisterRouteGroup(RegisterAgentRoutes)
}

// RegisterAgentRoutes adds agent-related endpoints to the mux.
func RegisterAgentRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /agents", requireProfile(handleListAgents(deps)))
	mux.Handle("GET /agents/{agent_id}", requireProfile(handleGetAgent(deps)))
	mux.Handle("PATCH /agents/{agent_id}", requireProfile(handleUpdateAgent(deps)))
	mux.Handle("POST /agents/{agent_id}/deactivate", requireProfile(handleDeactivateAgent(deps)))
	mux.Handle("POST /agents/{agent_id}/register", requireProfile(handleRegisterAgentDashboard(deps)))

	// Agent-authenticated endpoint: lets a registered agent check its own status.
	requireAgentSig := RequireAgentSignature(deps)
	mux.Handle("GET /agents/me", requireAgentSig(handleGetAgentMe(deps)))
}

func handleListAgents(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		limit, ok := parsePaginationLimit(w, r)
		if !ok {
			return
		}

		// Parse cursor: "<RFC3339Nano>,<agent_id>".
		var cursor *db.AgentCursor
		if v := r.URL.Query().Get("after"); v != "" {
			c, err := parseAgentCursor(v)
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invalid pagination cursor"))
				return
			}
			cursor = c
		}

		page, err := db.GetAgentsByApprover(r.Context(), deps.DB, userID, limit, cursor)
		if err != nil {
			log.Printf("[%s] ListAgents: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list agents"))
			return
		}

		data := make([]agentResponse, len(page.Agents))
		for i, item := range page.Agents {
			data[i] = toAgentListItemResponse(item)
		}

		resp := agentListResponse{
			Data:    data,
			HasMore: page.HasMore,
		}
		if page.HasMore && len(page.Agents) > 0 {
			c := encodeAgentCursor(page.Agents[len(page.Agents)-1].Agent)
			resp.NextCursor = &c
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

func handleGetAgent(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}

		agent, err := db.GetAgentByID(r.Context(), deps.DB, agentID, userID)
		if err != nil {
			log.Printf("[%s] GetAgent: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to get agent"))
			return
		}
		if agent == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
			return
		}

		RespondJSON(w, http.StatusOK, toAgentResponse(*agent))
	}
}

type updateAgentRequest struct {
	Metadata json.RawMessage `json:"metadata" validate:"required"`
}

func handleUpdateAgent(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}

		var body updateAgentRequest
		if !DecodeJSONOrReject(w, r, &body) {
			return
		}
		if !ValidateRequest(w, r, &body) {
			return
		}

		// Validate that metadata is a JSON object (not an array, string, number, etc.).
		// The JSONB || merge operator requires both sides to be objects; non-objects
		// would either corrupt data or cause a Postgres error.
		if err := ValidateJSONObject(body.Metadata); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "metadata must be a JSON object"))
			return
		}

		agent, err := db.UpdateAgentMetadata(r.Context(), deps.DB, agentID, userID, body.Metadata)
		if err != nil {
			log.Printf("[%s] UpdateAgent: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update agent"))
			return
		}
		if agent == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
			return
		}

		RespondJSON(w, http.StatusOK, toAgentResponse(*agent))
	}
}

func handleDeactivateAgent(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}

		agent, err := db.DeactivateAgent(r.Context(), deps.DB, agentID, userID)
		if err != nil {
			log.Printf("[%s] DeactivateAgent: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to deactivate agent"))
			return
		}
		if agent == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
			return
		}

		if err := db.InsertAuditEvent(r.Context(), deps.DB, db.InsertAuditEventParams{
			UserID:     userID,
			AgentID:    agent.AgentID,
			EventType:  db.AuditEventAgentDeactivated,
			Outcome:    "deactivated",
			SourceID:   fmt.Sprintf("ad:%d", agent.AgentID),
			SourceType: "agent",
			AgentMeta:  agent.Metadata,
		}); err != nil {
			log.Printf("[%s] DeactivateAgent: audit event: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
		}

		RespondJSON(w, http.StatusOK, toAgentResponse(*agent))
	}
}

func handleRegisterAgentDashboard(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := Profile(r.Context()).ID

		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}

		agent, err := db.RegisterAgent(r.Context(), deps.DB, agentID, userID)
		if err != nil {
			log.Printf("[%s] RegisterAgent: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to register agent"))
			return
		}
		if agent == nil {
			// No matching pending agent — check if it exists at all to give a specific error.
			existing, lookupErr := db.GetAgentByID(r.Context(), deps.DB, agentID, userID)
			if lookupErr != nil {
				log.Printf("[%s] RegisterAgent: lookup: %v", TraceID(r.Context()), lookupErr)
				CaptureError(r.Context(), lookupErr)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to register agent"))
				return
			}
			if existing == nil {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
				return
			}
			// Agent exists but is not pending — it's already registered or deactivated.
			RespondError(w, r, http.StatusConflict, Conflict(ErrAgentAlreadyRegistered, fmt.Sprintf("Agent is already %s", existing.Status)))
			return
		}

		// Emit audit event (best-effort).
		if err := db.InsertAuditEvent(r.Context(), deps.DB, db.InsertAuditEventParams{
			UserID:     userID,
			AgentID:    agent.AgentID,
			EventType:  db.AuditEventAgentRegistered,
			Outcome:    "registered",
			SourceID:   fmt.Sprintf("ar:%d", agent.AgentID),
			SourceType: "agent",
			AgentMeta:  agent.Metadata,
		}); err != nil {
			log.Printf("[%s] RegisterAgent: audit event: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
		}

		RespondJSON(w, http.StatusOK, toAgentResponse(*agent))
	}
}

// requireAgentOwnership checks that agentID belongs to userID. If the check
// fails or the agent is not owned by the user, it writes the appropriate error
// response and returns false. The caller should return immediately when false
// is returned.
func requireAgentOwnership(w http.ResponseWriter, r *http.Request, deps *Deps, agentID int64, userID string) bool {
	owns, err := db.AgentBelongsToUser(r.Context(), deps.DB, agentID, userID)
	if err != nil {
		log.Printf("[%s] ownership check: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to verify agent ownership"))
		return false
	}
	if !owns {
		RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
		return false
	}
	return true
}

// requireAgentSignature looks up the agent by ID (unscoped), verifies the
// Ed25519 request signature, and checks that the signature header's agent_id
// matches the path parameter. On success it returns the agent and parsed
// signature. On failure it writes the appropriate error response and returns
// nil, nil, false — the caller should return immediately.
//
// bodyBytes should be nil for GET requests and the raw request body for POST/PUT.
func requireAgentSignature(w http.ResponseWriter, r *http.Request, deps *Deps, agentID int64, bodyBytes []byte) (*db.Agent, *ParsedSignature, bool) {
	agent, err := db.GetAgentByIDUnscoped(r.Context(), deps.DB, agentID)
	if err != nil {
		log.Printf("[%s] requireAgentSignature: lookup agent: %v", TraceID(r.Context()), err)
		CaptureError(r.Context(), err)
		RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to process request"))
		return nil, nil, false
	}
	if agent == nil {
		RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
		return nil, nil, false
	}

	sig, err := VerifyAgentSignature(agent.PublicKey, r, bodyBytes)
	if err != nil {
		if errors.Is(err, ErrSigTimestampExpired) {
			RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrTimestampExpired, "Signature timestamp expired"))
			return nil, nil, false
		}
		RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidSignature, "Signature verification failed"))
		return nil, nil, false
	}

	if sig.AgentID != agentID {
		resp := BadRequest(ErrAgentIDMismatch, "Agent ID in path must match signature header")
		resp.Error.Details = map[string]any{
			"path_agent_id":   agentID,
			"header_agent_id": sig.AgentID,
		}
		RespondError(w, r, http.StatusBadRequest, resp)
		return nil, nil, false
	}

	// Per-agent rate limit (post-auth: identity is now verified).
	if !checkAgentRateLimit(w, r, deps, agent.AgentID) {
		return nil, nil, false
	}

	// Replay protection: record the verified signature hash so it cannot be
	// replayed within the ±5-minute timestamp window. This runs AFTER rate
	// limiting so replayed requests still count against the attacker's quota.
	if !consumeSignatureOrReject(w, r, deps, sig, agent.AgentID) {
		return nil, nil, false
	}

	return agent, sig, true
}

// parsePathAgentID extracts and validates the agent_id path parameter.
// Returns the parsed ID and true on success, or writes an error response and
// returns false.
func parsePathAgentID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := r.PathValue("agent_id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "agent_id must be a positive integer"))
		return 0, false
	}
	return id, true
}

func encodeAgentCursor(a db.Agent) string {
	return a.CreatedAt.UTC().Format(time.RFC3339Nano) + "," + strconv.FormatInt(a.AgentID, 10)
}

func parseAgentCursor(raw string) (*db.AgentCursor, error) {
	comma := strings.IndexByte(raw, ',')
	if comma < 0 {
		return nil, fmt.Errorf("missing comma separator")
	}
	t, err := time.Parse(time.RFC3339Nano, raw[:comma])
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp: %w", err)
	}
	idStr := raw[comma+1:]
	if idStr == "" {
		return nil, fmt.Errorf("empty agent_id")
	}
	agentID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid agent_id: %w", err)
	}
	return &db.AgentCursor{CreatedAt: t, AgentID: agentID}, nil
}

func toAgentResponse(a db.Agent) agentResponse {
	resp := agentResponse{
		AgentID:       a.AgentID,
		Status:        a.Status,
		ExpiresAt:     a.ExpiresAt,
		RegisteredAt:  a.RegisteredAt,
		DeactivatedAt: a.DeactivatedAt,
		LastActiveAt:  a.LastActiveAt,
		CreatedAt:     a.CreatedAt,
	}
	// Only expose the confirmation code for pending agents — it's cleared on
	// verification, but guard here as defense-in-depth.
	if a.Status == db.AgentStatusPending {
		resp.ConfirmationCode = a.ConfirmationCode
	}
	if len(a.Metadata) > 0 {
		var meta any
		if err := json.Unmarshal(a.Metadata, &meta); err == nil {
			resp.Metadata = meta
		}
	}
	return resp
}

func toAgentListItemResponse(item db.AgentListItem) agentResponse {
	resp := toAgentResponse(item.Agent)
	resp.RequestCount30d = &item.RequestCount30d
	return resp
}

// ── Agent-authenticated endpoints ────────────────────────────────────────────

// agentSelfResponse is the response for GET /agents/me. It exposes only the
// fields an agent needs to know about itself — no confirmation_code, no
// approver_id, no request_count.
type agentSelfResponse struct {
	AgentID       int64         `json:"agent_id"`
	Status        string        `json:"status"`
	Approver      *approverInfo `json:"approver,omitempty"`
	Metadata      any           `json:"metadata,omitempty"`
	RegisteredAt  *time.Time    `json:"registered_at,omitempty"`
	DeactivatedAt *time.Time    `json:"deactivated_at,omitempty"`
	LastActiveAt  *time.Time    `json:"last_active_at,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
}

func handleGetAgentMe(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())
		if agent == nil {
			// Shouldn't happen — RequireAgentSignature guarantees this.
			RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidSignature, "Agent authentication required"))
			return
		}

		resp := agentSelfResponse{
			AgentID:       agent.AgentID,
			Status:        agent.Status,
			RegisteredAt:  agent.RegisteredAt,
			DeactivatedAt: agent.DeactivatedAt,
			LastActiveAt:  agent.LastActiveAt,
			CreatedAt:     agent.CreatedAt,
		}
		if len(agent.Metadata) > 0 {
			var meta any
			if err := json.Unmarshal(agent.Metadata, &meta); err == nil {
				resp.Metadata = meta
			}
		}

		// Look up the approver's profile to include their username.
		profile, err := db.GetProfileByUserID(r.Context(), deps.DB, agent.ApproverID)
		if err != nil {
			log.Printf("[%s] GetAgentMe: lookup approver profile: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
		} else if profile != nil {
			resp.Approver = &approverInfo{Username: profile.Username}
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}
