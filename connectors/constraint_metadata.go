package connectors

import (
	"context"
	"encoding/json"
	"errors"
)

// ErrConstraintMetadataUnavailable indicates verified metadata could not be
// resolved for the requested action parameters (e.g. IMAP fetch failed).
var ErrConstraintMetadataUnavailable = errors.New("constraint metadata unavailable")

// ConstraintMetadataResolver is optionally implemented by connectors that can
// resolve verified metadata for constraint matching. Values are fetched from
// the external service (e.g. real email From headers) — never from agent-supplied
// parameters.
type ConstraintMetadataResolver interface {
	// ResolveConstraintMetadata returns verified metadata for the target
	// resource(s) referenced by params. For batch or thread-expanded actions,
	// every message in the effective set must be represented (e.g. all senders
	// in a "senders" array) so callers can enforce fail-closed matching.
	ResolveConstraintMetadata(ctx context.Context, actionType string, params json.RawMessage, creds Credentials) (map[string]any, error)
}
