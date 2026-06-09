package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/shared"
)

// --- Request / response types ---

type storeCredentialRequest struct {
	Service     string         `json:"service" validate:"required"`
	Credentials map[string]any `json:"credentials" validate:"required,min=1"`
	Label       *string        `json:"label,omitempty"`
}

type credentialSummary struct {
	ID        string    `json:"id"`
	Service   string    `json:"service"`
	Label     *string   `json:"label,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type credentialListResponse struct {
	Credentials []credentialSummary `json:"credentials"`
}

type deleteCredentialResponse struct {
	ID        string    `json:"id"`
	DeletedAt time.Time `json:"deleted_at"`
}

// updateCredentialRequestRaw uses json.RawMessage for PATCH semantics.
type updateCredentialRequestRaw struct {
	Label       json.RawMessage `json:"label"`
	Credentials json.RawMessage `json:"credentials"`
}

// --- Validation ---

var servicePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]*$`)

// --- Routes ---

func init() {
	RegisterRouteGroup(RegisterCredentialRoutes)
}

// RegisterCredentialRoutes adds credential-related endpoints to the mux.
func RegisterCredentialRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /credentials", requireProfile(handleListCredentials(deps)))
	mux.Handle("POST /credentials", requireProfile(handleStoreCredential(deps)))
	mux.Handle("PATCH /credentials/{credential_id}", requireProfile(handleUpdateCredential(deps)))
	mux.Handle("DELETE /credentials/{credential_id}", requireProfile(handleDeleteCredential(deps)))
}

// --- Handlers ---

func handleListCredentials(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		creds, err := db.ListCredentialsByUser(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] ListCredentials: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to list credentials"))
			return
		}

		data := make([]credentialSummary, len(creds))
		for i, c := range creds {
			data[i] = toCredentialSummary(c)
		}

		RespondJSON(w, http.StatusOK, credentialListResponse{Credentials: data})
	}
}

func handleStoreCredential(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Vault == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Credential vault not available"))
			return
		}

		profile := Profile(r.Context())

		var req storeCredentialRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		if len(req.Service) > shared.CredentialServiceMaxLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "service exceeds maximum length"))
			return
		}
		if !servicePattern.MatchString(req.Service) {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "service must start with a lowercase letter and contain only lowercase letters, digits, underscores, dots, or hyphens"))
			return
		}
		if req.Label != nil && len(*req.Label) > shared.CredentialLabelMaxLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "label exceeds maximum length"))
			return
		}

		credID, err := generatePrefixedID("cred_", 16)
		if err != nil {
			log.Printf("[%s] StoreCredential: generate ID: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to store credential"))
			return
		}

		// Begin a transaction so vault insert + credential row insert are atomic.
		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] StoreCredential: begin tx: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to store credential"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx) //nolint:errcheck // best-effort cleanup
		}

		candidates, err := db.GetRequiredCredentialsByService(r.Context(), tx, req.Service)
		if err != nil {
			log.Printf("[%s] StoreCredential: list required credentials: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to store credential"))
			return
		}
		credStrings, pickErr := resolveAndValidateCredentialPayload(req.Service, candidates, req.Credentials)
		if pickErr != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, pickErr.Error()))
			return
		}
		credJSON, err := json.Marshal(credStrings)
		if err != nil {
			log.Printf("[%s] StoreCredential: marshal credentials: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to store credential"))
			return
		}

		// Encrypt credentials and store in vault.
		vaultSecretID, err := deps.Vault.CreateSecret(r.Context(), tx, credID, credJSON)
		if err != nil {
			log.Printf("[%s] StoreCredential: vault create: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to store credential"))
			return
		}

		cred, err := db.CreateCredential(r.Context(), tx, db.CreateCredentialParams{
			ID:            credID,
			UserID:        profile.ID,
			Service:       req.Service,
			Label:         req.Label,
			VaultSecretID: vaultSecretID,
		})
		if err != nil {
			var credErr *db.CredentialError
			if errors.As(err, &credErr) && credErr.Code == db.CredentialErrDuplicate {
				resp := Conflict(ErrDuplicateCredential, "Credentials already stored for this service with this label")
				resp.Error.Details = map[string]any{
					"service": req.Service,
				}
				if req.Label != nil {
					resp.Error.Details["label"] = *req.Label
				}
				RespondError(w, r, http.StatusConflict, resp)
				return
			}
			log.Printf("[%s] StoreCredential: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to store credential"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] StoreCredential: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to store credential"))
				return
			}
		}

		RespondJSON(w, http.StatusCreated, toCredentialSummary(*cred))
	}
}

func handleUpdateCredential(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())
		credID := r.PathValue("credential_id")

		if credID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "credential_id is required"))
			return
		}

		var req updateCredentialRequestRaw
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}

		newLabel, labelProvided, err := parseOptionalString(req.Label)
		if err != nil {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "label must be a string or null"))
			return
		}
		if labelProvided && newLabel != nil && len(*newLabel) > shared.CredentialLabelMaxLength {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "label exceeds maximum length"))
			return
		}

		var credentialUpdates map[string]any
		credentialsProvided := len(req.Credentials) > 0
		if credentialsProvided {
			if isRawJSONNull(req.Credentials) {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "credentials must be an object"))
				return
			}
			if err := json.Unmarshal(req.Credentials, &credentialUpdates); err != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "credentials must be an object"))
				return
			}
		}

		hasCredentialFieldUpdates := false
		for _, v := range credentialUpdates {
			s, ok := v.(string)
			if !ok {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "credential values must be strings"))
				return
			}
			if s != "" {
				hasCredentialFieldUpdates = true
				break
			}
		}

		if !labelProvided && !hasCredentialFieldUpdates {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "at least one field must be provided"))
			return
		}

		if hasCredentialFieldUpdates && deps.Vault == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Credential vault not available"))
			return
		}

		tx, txOwned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] UpdateCredential: begin tx: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update credential"))
			return
		}
		if txOwned {
			defer db.RollbackTx(r.Context(), tx) //nolint:errcheck // best-effort cleanup
		}

		ownedCred, err := db.GetOwnedCredential(r.Context(), tx, credID, profile.ID)
		if err != nil {
			var credErr *db.CredentialError
			if errors.As(err, &credErr) && credErr.Code == db.CredentialErrNotFound {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrCredentialNotFound, "Credential not found"))
				return
			}
			log.Printf("[%s] UpdateCredential: load: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update credential"))
			return
		}

		result := ownedCred.Credential

		if hasCredentialFieldUpdates {
			raw, err := deps.Vault.ReadSecret(r.Context(), tx, ownedCred.VaultSecretID)
			if err != nil {
				log.Printf("[%s] UpdateCredential: vault read: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update credential"))
				return
			}

			var existing map[string]string
			if err := json.Unmarshal(raw, &existing); err != nil {
				log.Printf("[%s] UpdateCredential: unmarshal existing: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update credential"))
				return
			}

			merged := make(map[string]any, len(existing))
			for k, v := range existing {
				merged[k] = v
			}
			for k, v := range credentialUpdates {
				s, ok := v.(string)
				if !ok || s == "" {
					continue
				}
				merged[k] = s
			}

			candidates, err := db.GetRequiredCredentialsByService(r.Context(), tx, ownedCred.Service)
			if err != nil {
				log.Printf("[%s] UpdateCredential: list required credentials: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update credential"))
				return
			}
			credStrings, pickErr := resolveAndValidateCredentialPayload(ownedCred.Service, candidates, merged)
			if pickErr != nil {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, pickErr.Error()))
				return
			}
			credJSON, err := json.Marshal(credStrings)
			if err != nil {
				log.Printf("[%s] UpdateCredential: marshal credentials: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update credential"))
				return
			}

			if err := deps.Vault.UpdateSecret(r.Context(), tx, ownedCred.VaultSecretID, credJSON); err != nil {
				log.Printf("[%s] UpdateCredential: vault update: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update credential"))
				return
			}
		}

		if labelProvided {
			updated, err := db.UpdateCredentialLabel(r.Context(), tx, credID, profile.ID, newLabel)
			if err != nil {
				var credErr *db.CredentialError
				if errors.As(err, &credErr) {
					switch credErr.Code {
					case db.CredentialErrNotFound:
						RespondError(w, r, http.StatusNotFound, NotFound(ErrCredentialNotFound, "Credential not found"))
						return
					case db.CredentialErrDuplicate:
						resp := Conflict(ErrDuplicateCredential, "Credentials already stored for this service with this label")
						resp.Error.Details = map[string]any{
							"service": ownedCred.Service,
						}
						if newLabel != nil {
							resp.Error.Details["label"] = *newLabel
						}
						RespondError(w, r, http.StatusConflict, resp)
						return
					}
				}
				log.Printf("[%s] UpdateCredential: label: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update credential"))
				return
			}
			result = *updated
		}

		if txOwned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] UpdateCredential: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to update credential"))
				return
			}
		}

		RespondJSON(w, http.StatusOK, toCredentialSummary(result))
	}
}

func handleDeleteCredential(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Vault == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Credential vault not available"))
			return
		}

		profile := Profile(r.Context())
		credID := r.PathValue("credential_id")

		if credID == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "credential_id is required"))
			return
		}

		// Begin a transaction so credential row delete + vault secret delete are atomic.
		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] DeleteCredential: begin tx: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete credential"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx) //nolint:errcheck // best-effort cleanup
		}

		// Delete is user-scoped: returns 404 for both "doesn't exist" and
		// "belongs to another user" to avoid leaking credential existence.
		result, err := db.DeleteCredential(r.Context(), tx, credID, profile.ID)
		if err != nil {
			var credErr *db.CredentialError
			if errors.As(err, &credErr) && credErr.Code == db.CredentialErrNotFound {
				RespondError(w, r, http.StatusNotFound, NotFound(ErrCredentialNotFound, "Credential not found"))
				return
			}
			log.Printf("[%s] DeleteCredential: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete credential"))
			return
		}

		// Delete the vault secret (idempotent — no error if already gone).
		if err := deps.Vault.DeleteSecret(r.Context(), tx, result.VaultSecretID); err != nil {
			log.Printf("[%s] DeleteCredential: vault delete: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete credential"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] DeleteCredential: commit: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete credential"))
				return
			}
		}

		RespondJSON(w, http.StatusOK, deleteCredentialResponse{
			ID:        credID,
			DeletedAt: result.DeletedAt,
		})
	}
}

// --- Helpers ---

func toCredentialSummary(c db.Credential) credentialSummary {
	return credentialSummary{
		ID:        c.ID,
		Service:   c.Service,
		Label:     c.Label,
		CreatedAt: c.CreatedAt,
	}
}

// resolveAndValidateCredentialPayload picks the matching connector credential row
// for a service and validates keys/values. Returns string map for vault storage.
//
// When no connector row exists for the service, or only OAuth rows exist, there is
// no static field schema to enforce — accept any string-valued map (legacy behavior
// for settings UI, ad-hoc services, and bot-token style credentials).
func resolveAndValidateCredentialPayload(service string, candidates []db.RequiredCredential, submitted map[string]any) (map[string]string, error) {
	var matches []db.RequiredCredential
	for _, c := range candidates {
		if c.AuthType == "oauth2" {
			continue
		}
		matches = append(matches, c)
	}
	if len(matches) == 0 {
		return credentialMapStrings(submitted)
	}
	if len(matches) == 1 {
		if err := db.ValidateStaticCredentialKeys(matches[0], submitted); err != nil {
			return nil, err
		}
		return credentialMapStrings(submitted)
	}
	// Multiple connectors share this service name — disambiguate by field schema.
	var picked *db.RequiredCredential
	for i := range matches {
		if err := db.ValidateStaticCredentialKeys(matches[i], submitted); err != nil {
			continue
		}
		if picked != nil && !sameCredentialSchema(*picked, matches[i]) {
			return nil, fmt.Errorf("ambiguous credential schema for service %q — contact support", service)
		}
		cp := matches[i]
		picked = &cp
	}
	if picked == nil {
		return nil, fmt.Errorf("credentials do not match any registered connector for service %q", service)
	}
	return credentialMapStrings(submitted)
}

func sameCredentialSchema(a, b db.RequiredCredential) bool {
	if a.AuthType != b.AuthType {
		return false
	}
	return db.CredentialFieldSpecsMatch(a.CredentialFields, b.CredentialFields)
}

func credentialMapStrings(m map[string]any) (map[string]string, error) {
	out := make(map[string]string, len(m))
	for k, v := range m {
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("credential key %q must be a string", k)
		}
		out[k] = s
	}
	return out, nil
}
