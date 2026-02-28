package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/db"
)

type agentContextKey struct{}
type parsedSigContextKey struct{}

// RequireAgentSignature returns middleware that validates agent signature
// authentication via the X-Permission-Slip-Signature header.
//
// It parses the signature header, looks up the agent by ID, verifies the
// Ed25519 signature against the agent's stored public key, and checks that
// the agent is in "registered" status.
//
// On success, the authenticated agent and parsed signature are stored in
// the request context for retrieval via AuthenticatedAgent(ctx) and
// AuthenticatedSignature(ctx). The request body is re-set so downstream
// handlers can read it.
func RequireAgentSignature(deps *Deps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if deps.DB == nil {
				RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Database not available"))
				return
			}

			// Read body for signature verification.
			bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBodySize))
			if err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Failed to read request body"))
				return
			}
			// Re-set body so downstream handlers can read it.
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			// Parse signature header.
			headerVal := r.Header.Get(signatureHeader)
			if headerVal == "" {
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidSignature, "Missing X-Permission-Slip-Signature header"))
				return
			}

			sig, err := ParseSignatureHeader(headerVal)
			if err != nil {
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidSignature, "Invalid signature header"))
				return
			}

			// Look up agent by ID.
			agent, err := db.GetAgentByIDUnscoped(r.Context(), deps.DB, sig.AgentID)
			if err != nil {
				log.Printf("[%s] RequireAgentSignature: agent lookup: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to verify agent"))
				return
			}
			if agent == nil {
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrAgentNotFound, "Agent not found"))
				return
			}

			// Verify the Ed25519 signature against the agent's stored public key.
			pubKey, err := ParseEd25519PublicKey(agent.PublicKey)
			if err != nil {
				log.Printf("[%s] RequireAgentSignature: parse stored public key: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to verify agent"))
				return
			}

			if err := VerifyEd25519Signature(pubKey, sig, r, bodyBytes); err != nil {
				if errors.Is(err, ErrSigTimestampExpired) {
					RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrTimestampExpired, "Signature timestamp expired"))
					return
				}
				RespondError(w, r, http.StatusUnauthorized, Unauthorized(ErrInvalidSignature, "Signature verification failed"))
				return
			}

			// Require the agent to be registered.
			if agent.Status != db.AgentStatusRegistered {
				RespondError(w, r, http.StatusForbidden, Forbidden(ErrAgentNotAuthorized, fmt.Sprintf("Agent is %s, not registered", agent.Status)))
				return
			}

			// Per-agent rate limit (post-auth: identity is now verified).
			if !checkAgentRateLimit(w, r, deps, agent.AgentID) {
				return
			}

			ctx := context.WithValue(r.Context(), agentContextKey{}, agent)
			ctx = context.WithValue(ctx, parsedSigContextKey{}, sig)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AuthenticatedAgent returns the agent from the request context, set by
// RequireAgentSignature middleware. Returns nil if not present.
func AuthenticatedAgent(ctx context.Context) *db.Agent {
	a, _ := ctx.Value(agentContextKey{}).(*db.Agent)
	return a
}

// AuthenticatedSignature returns the parsed signature from the request
// context, set by RequireAgentSignature middleware. Returns nil if not present.
func AuthenticatedSignature(ctx context.Context) *ParsedSignature {
	s, _ := ctx.Value(parsedSigContextKey{}).(*ParsedSignature)
	return s
}
