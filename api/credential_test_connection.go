package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/connectors/protonmail"
	"github.com/supersuit-tech/permission-slip/db"
)

const protonmailService = "protonmail"

// testProtonBridgeConnection is the Bridge handshake used by the API. Tests may replace it.
var testProtonBridgeConnection = func(ctx context.Context, creds connectors.Credentials, timeout time.Duration) error {
	return protonmail.TestBridgeConnectionContext(ctx, creds, timeout)
}

type testCredentialConnectionRequest struct {
	Service      string         `json:"service" validate:"required"`
	Credentials  map[string]any `json:"credentials"`
	CredentialID *string        `json:"credential_id,omitempty"`
}

type testCredentialConnectionResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

func init() {
	RegisterRouteGroup(RegisterCredentialTestConnectionRoutes)
}

// RegisterCredentialTestConnectionRoutes adds the Bridge connectivity test endpoint.
func RegisterCredentialTestConnectionRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("POST /credentials/test-connection", requireProfile(handleTestCredentialConnection(deps)))
}

func handleTestCredentialConnection(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		var req testCredentialConnectionRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		if req.Service != protonmailService {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "test-connection is only supported for protonmail credentials"))
			return
		}

		credStrings, credID, err := resolveTestConnectionCredentials(r.Context(), deps, profile.ID, req)
		if err != nil {
			var credErr *db.CredentialError
			if errors.As(err, &credErr) && credErr.Code == db.CredentialErrNotFound {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrCredentialNotFound, "Credential not found"))
				return
			}
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, err.Error()))
			return
		}

		connErrCtx := ConnectorContext{
			ConnectorID: protonmailService,
			ActionType:  "protonmail.read_inbox",
		}

		creds := connectors.NewCredentials(credStrings)
		if err := testProtonBridgeConnection(r.Context(), creds, 30*time.Second); err != nil {
			if credID != "" {
				_ = db.SetProtonmailHealth(r.Context(), deps.DB, credID, db.ProtonmailHealthState{
					Status:    db.ProtonmailHealthError,
					CheckedAt: time.Now().UTC(),
					Message:   connectorErrorMessage(err),
				})
			}
			if handleConnectorError(w, r, err, connErrCtx) {
				return
			}
			log.Printf("[%s] TestCredentialConnection: %v", TraceID(r.Context()), err)
			CaptureConnectorError(r.Context(), err, connErrCtx)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Bridge connection test failed"))
			return
		}

		if credID != "" {
			if err := db.SetProtonmailHealth(r.Context(), deps.DB, credID, db.ProtonmailHealthState{
				Status:    db.ProtonmailHealthOK,
				CheckedAt: time.Now().UTC(),
			}); err != nil {
				log.Printf("[%s] TestCredentialConnection: persist health: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
			}
		}

		RespondJSON(w, http.StatusOK, testCredentialConnectionResponse{
			OK:      true,
			Message: "Bridge connection successful (IMAP and SMTP)",
		})
	}
}

func resolveTestConnectionCredentials(
	ctx context.Context,
	deps *Deps,
	userID string,
	req testCredentialConnectionRequest,
) (map[string]string, string, error) {
	var base map[string]string
	var credID string

	if req.CredentialID != nil && *req.CredentialID != "" {
		credID = *req.CredentialID
		if deps.Vault == nil {
			return nil, "", errors.New("credential vault not available")
		}
		owned, err := db.GetOwnedCredential(ctx, deps.DB, credID, userID)
		if err != nil {
			return nil, "", err
		}
		if owned.Service != protonmailService {
			return nil, "", errors.New("credential service does not match protonmail")
		}
		raw, err := deps.Vault.ReadSecret(ctx, deps.DB, owned.VaultSecretID)
		if err != nil {
			return nil, "", err
		}
		if err := json.Unmarshal(raw, &base); err != nil {
			return nil, "", err
		}
	}

	merged := make(map[string]any)
	for k, v := range base {
		merged[k] = v
	}
	for k, v := range req.Credentials {
		if s, ok := v.(string); ok && s != "" {
			merged[k] = s
		}
	}

	if len(merged) == 0 {
		return nil, "", errors.New("credentials are required when credential_id is not provided")
	}

	candidates, err := db.GetRequiredCredentialsByService(ctx, deps.DB, protonmailService)
	if err != nil {
		return nil, "", err
	}
	credStrings, pickErr := resolveAndValidateCredentialPayload(protonmailService, candidates, merged)
	if pickErr != nil {
		return nil, "", pickErr
	}
	return credStrings, credID, nil
}

func connectorErrorMessage(err error) string {
	switch {
	case connectors.IsAuthError(err):
		var ae *connectors.AuthError
		if errors.As(err, &ae) && ae.Message != "" {
			return ae.Message
		}
	case connectors.IsTimeoutError(err):
		var te *connectors.TimeoutError
		if errors.As(err, &te) && te.Message != "" {
			return te.Message
		}
	case connectors.IsExternalError(err):
		var ee *connectors.ExternalError
		if errors.As(err, &ee) && ee.Message != "" {
			return ee.Message
		}
	case connectors.IsValidationError(err):
		var ve *connectors.ValidationError
		if errors.As(err, &ve) && ve.Message != "" {
			return ve.Message
		}
	}
	return err.Error()
}

func maybeValidateLiveCredentials(ctx context.Context, deps *Deps, service string, credStrings map[string]string) error {
	if deps.Connectors == nil || service != protonmailService {
		return nil
	}
	conn, ok := deps.Connectors.Get(protonmailService)
	if !ok {
		return nil
	}
	return conn.ValidateCredentials(ctx, connectors.NewCredentials(credStrings))
}
