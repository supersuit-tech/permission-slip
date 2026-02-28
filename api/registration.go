package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

const (
	// defaultRegistrationTTL is the default time an agent has to verify after
	// registering via an invite (5 minutes).
	defaultRegistrationTTL = 5 * 60 // seconds
)

// Error codes specific to invite registration.
const (
	ErrInviteNotFound ErrorCode = "invite_not_found"
	ErrInviteExpired  ErrorCode = "invite_expired"
	ErrInviteLocked   ErrorCode = "invite_locked"
)

// ── Request / Response types ────────────────────────────────────────────────

type registerAgentRequest struct {
	RequestID string          `json:"request_id" validate:"required"`
	PublicKey string          `json:"public_key" validate:"required"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type registerAgentResponse struct {
	AgentID              int64      `json:"agent_id"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"`
	VerificationRequired bool       `json:"verification_required"`
}

type verifyRegistrationRequest struct {
	RequestID        string `json:"request_id" validate:"required"`
	ConfirmationCode string `json:"confirmation_code" validate:"required"`
}

type verifyRegistrationResponse struct {
	Status       string     `json:"status"`
	RegisteredAt *time.Time `json:"registered_at,omitempty"`
}

// ── Route registration ──────────────────────────────────────────────────────

// RegisterRegistrationRoutes adds the versioned agent registration endpoints
// to the mux (mounted under /api/v1/). The invite endpoint is intentionally
// excluded — it lives at the top level (/invite/{code}) via InviteHandler.
func RegisterRegistrationRoutes(mux *http.ServeMux, deps *Deps) {
	mux.HandleFunc("POST /agents/{agent_id}/verify", handleVerifyRegistration(deps))
}

// InviteHandler returns an http.Handler for POST /invite/{invite_code}.
// This is mounted at the top level (outside /api/v1/) because the invite URL
// is a user-facing onboarding entry point, not a versioned API resource.
// Includes TraceIDMiddleware for request tracing.
func InviteHandler(deps *Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /invite/{invite_code}", handleRegisterAgent(deps))
	return TraceIDMiddleware(mux)
}

// ── POST /invite/{invite_code} ──────────────────────────────────────────────

func handleRegisterAgent(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Database not available"))
			return
		}

		// Read the full body for signature verification.
		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBodySize))
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Failed to read request body"))
			return
		}

		// Decode the JSON body.
		var req registerAgentRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Malformed or invalid JSON"))
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		// Validate the public key is a valid Ed25519 key.
		if _, err := ParseEd25519PublicKey(req.PublicKey); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidPublicKey, "Invalid Ed25519 public key: "+err.Error()))
			return
		}

		// Verify the request signature against the submitted public key.
		if _, err := VerifyRegistrationSignature(req.PublicKey, r, bodyBytes); err != nil {
			if errors.Is(err, ErrSigTimestampExpired) {
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrTimestampExpired, "Signature timestamp expired"))
				return
			}
			RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidSignature, "Signature verification failed"))
			return
		}

		// Extract and validate the invite code from the URL path.
		inviteCode := r.PathValue("invite_code")
		if inviteCode == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "invite_code is required"))
			return
		}

		// Hash the invite code for lookup.
		codeHash := hashCodeHex(inviteCode, deps.InviteHMACKey)

		// Generate the confirmation code up front (before the transaction) so
		// that a crypto/rand failure doesn't cause a needless rollback.
		confirmCode, err := generateConfirmationCodePlaintext()
		if err != nil {
			log.Printf("[%s] RegisterAgent: generate confirmation code: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to process registration"))
			return
		}

		// Validate and prepare metadata (must be a JSON object if provided).
		var metadata []byte
		if len(req.Metadata) > 0 {
			var obj map[string]json.RawMessage
			if err := json.Unmarshal(req.Metadata, &obj); err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "metadata must be a JSON object"))
				return
			}
			metadata = []byte(req.Metadata)
		}

		// Wrap invite consumption, agent creation, and audit logging in a
		// single transaction so that a failure in any step rolls back the
		// entire operation. Without this, ConsumeInvite could mark the invite
		// as consumed while InsertPendingAgent fails, permanently losing the
		// invite (TOCTOU).
		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] RegisterAgent: begin tx: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to process registration"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx) //nolint:errcheck // best-effort on failure path
		}

		// Try to consume the invite atomically.
		invite, err := db.ConsumeInvite(r.Context(), tx, codeHash)
		if err != nil {
			log.Printf("[%s] RegisterAgent: consume invite: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to process registration"))
			return
		}

		if invite == nil {
			// Invite not consumed — figure out why for a specific error message.
			existing, lookupErr := db.LookupInviteByCodeHash(r.Context(), tx, codeHash)
			if lookupErr != nil {
				log.Printf("[%s] RegisterAgent: lookup invite: %v", TraceID(r.Context()), lookupErr)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to process registration"))
				return
			}
			if existing == nil {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrInviteNotFound, "No invite found for this code"))
				return
			}
			if existing.Status == "consumed" {
				resp := Conflict(ErrAgentAlreadyRegistered, "This invite has already been used")
				RespondError(w, r, http.StatusConflict, resp)
				return
			}
			if existing.VerificationAttempts >= 5 {
				resp := newErrorResponse(ErrInviteLocked, "Invite locked after too many failed attempts — ask the user to generate a new one", false)
				resp.Error.Details = map[string]any{
					"failed_attempts": existing.VerificationAttempts,
					"max_attempts":    5,
				}
				RespondError(w, r, http.StatusLocked, resp)
				return
			}
			// Must be expired.
			RespondError(w, r, http.StatusGone, Gone(ErrInviteExpired, "Invite has expired — ask the user to generate a new one"))
			return
		}

		// Insert the pending agent.
		agent, err := db.InsertPendingAgent(
			r.Context(), tx,
			invite.UserID, req.PublicKey, confirmCode,
			defaultRegistrationTTL, metadata,
		)
		if err != nil {
			log.Printf("[%s] RegisterAgent: insert pending agent: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to create agent registration"))
			return
		}

		// Log audit event.
		if err := db.InsertAuditEvent(r.Context(), tx, db.InsertAuditEventParams{
			UserID:     invite.UserID,
			AgentID:    agent.AgentID,
			EventType:  db.AuditEventAgentRegistered,
			Outcome:    "pending",
			SourceID:   fmt.Sprintf("ri:%s", invite.ID),
			SourceType: "registration_invite",
			AgentMeta:  metadata,
		}); err != nil {
			log.Printf("[%s] RegisterAgent: audit event: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to process registration"))
			return
		}

		// All writes succeeded — commit the transaction.
		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] RegisterAgent: commit tx: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to process registration"))
				return
			}
		}

		RespondJSON(w, http.StatusOK, registerAgentResponse{
			AgentID:              agent.AgentID,
			ExpiresAt:            agent.ExpiresAt,
			VerificationRequired: true,
		})
	}
}

// ── POST /agents/{agent_id}/verify ───────────────────────────────────────────

func handleVerifyRegistration(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Database not available"))
			return
		}

		// Parse agent_id from path.
		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}

		// Read the full body for signature verification.
		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBodySize))
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Failed to read request body"))
			return
		}

		// Decode the JSON body.
		var req verifyRegistrationRequest
		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Malformed or invalid JSON"))
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		// Validate confirmation code format (6 chars from safe set, optional hyphen).
		normalized := normalizeConfirmationCode(req.ConfirmationCode)
		if len(normalized) != 6 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Invalid confirmation code format"))
			return
		}

		// Authenticate: look up agent, verify signature, check agent_id match.
		_, _, ok = requireAgentSignature(w, r, deps, agentID, bodyBytes)
		if !ok {
			return
		}

		// Verify the confirmation code (plaintext comparison).
		registered, verifyErr := db.VerifyAgentConfirmationCode(r.Context(), deps.DB, agentID, normalized)
		if verifyErr != nil {
			switch {
			case errors.Is(verifyErr, db.ErrAgentNotPending):
				if registered != nil && registered.Status == db.AgentStatusRegistered {
					RespondError(w, r, http.StatusConflict, Conflict(ErrAgentAlreadyRegistered, "Agent is already registered"))
				} else {
					RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent registration not found or expired"))
				}
			case errors.Is(verifyErr, db.ErrRegistrationExpired):
				resp := Gone(ErrRegistrationExpired, "Registration has expired")
				if registered != nil && registered.ExpiresAt != nil {
					resp.Error.Details = map[string]any{
						"expired_at": registered.ExpiresAt.Format(time.RFC3339),
					}
				}
				RespondError(w, r, http.StatusGone, resp)
			case errors.Is(verifyErr, db.ErrVerificationLocked):
				resp := Gone(ErrVerificationLocked, "Too many failed verification attempts")
				if registered != nil {
					resp.Error.Details = map[string]any{
						"failed_attempts": registered.VerificationAttempts,
						"max_attempts":    5,
					}
				}
				RespondError(w, r, http.StatusGone, resp)
			case errors.Is(verifyErr, db.ErrInvalidConfirmation):
				resp := Unauthorized(ErrInvalidCode, "Incorrect confirmation code")
				if registered != nil {
					attemptsRemaining := 5 - registered.VerificationAttempts
					if attemptsRemaining < 0 {
						attemptsRemaining = 0
					}
					resp.Error.Details = map[string]any{
						"attempts_remaining": attemptsRemaining,
					}
				}
				RespondError(w, r, http.StatusUnauthorized, resp)
			default:
				log.Printf("[%s] VerifyRegistration: %v", TraceID(r.Context()), verifyErr)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to verify registration"))
			}
			return
		}
		if registered == nil {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent registration not found or expired"))
			return
		}

		// Log audit event for successful registration.
		if err := db.InsertAuditEvent(r.Context(), deps.DB, db.InsertAuditEventParams{
			UserID:     registered.ApproverID,
			AgentID:    registered.AgentID,
			EventType:  db.AuditEventAgentRegistered,
			Outcome:    "registered",
			SourceID:   fmt.Sprintf("ar:%d", registered.AgentID),
			SourceType: "agent",
			AgentMeta:  registered.Metadata,
		}); err != nil {
			log.Printf("[%s] VerifyRegistration: audit event: %v", TraceID(r.Context()), err)
		}

		RespondJSON(w, http.StatusOK, verifyRegistrationResponse{
			Status:       "registered",
			RegisteredAt: registered.RegisteredAt,
		})
	}
}
