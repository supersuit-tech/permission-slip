package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/shared"
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

// --- Validation ---

var servicePattern = regexp.MustCompile(`^[a-z][a-z0-9_.-]*$`)

// --- Routes ---

// RegisterCredentialRoutes adds credential-related endpoints to the mux.
func RegisterCredentialRoutes(mux *http.ServeMux, deps *Deps) {
	requireProfile := RequireProfile(deps)
	mux.Handle("GET /credentials", requireProfile(handleListCredentials(deps)))
	mux.Handle("POST /credentials", requireProfile(handleStoreCredential(deps)))
	mux.Handle("DELETE /credentials/{credential_id}", requireProfile(handleDeleteCredential(deps)))
}

// --- Handlers ---

func handleListCredentials(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile := Profile(r.Context())

		creds, err := db.ListCredentialsByUser(r.Context(), deps.DB, profile.ID)
		if err != nil {
			log.Printf("[%s] ListCredentials: %v", TraceID(r.Context()), err)
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
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to store credential"))
			return
		}

		// Serialize the credentials object — never log the raw value.
		credJSON, err := json.Marshal(req.Credentials)
		if err != nil {
			log.Printf("[%s] StoreCredential: marshal credentials: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to store credential"))
			return
		}

		// Begin a transaction so limit check + vault insert + credential row
		// insert are atomic. The advisory lock prevents TOCTOU races where
		// concurrent requests could both pass the limit check.
		tx, owned, err := db.BeginOrContinue(r.Context(), deps.DB)
		if err != nil {
			log.Printf("[%s] StoreCredential: begin tx: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to store credential"))
			return
		}
		if owned {
			defer db.RollbackTx(r.Context(), tx) //nolint:errcheck // best-effort cleanup
		}

		if err := db.AcquireCredentialLimitLock(r.Context(), tx, profile.ID); err != nil {
			log.Printf("[%s] StoreCredential: advisory lock: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to store credential"))
			return
		}
		if checkCredentialLimit(r.Context(), w, r, tx, profile.ID) {
			return
		}

		// Encrypt credentials and store in vault.
		vaultSecretID, err := deps.Vault.CreateSecret(r.Context(), tx, credID, credJSON)
		if err != nil {
			log.Printf("[%s] StoreCredential: vault create: %v", TraceID(r.Context()), err)
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
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to store credential"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] StoreCredential: commit: %v", TraceID(r.Context()), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to store credential"))
				return
			}
		}

		RespondJSON(w, http.StatusCreated, toCredentialSummary(*cred))
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
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete credential"))
			return
		}

		// Delete the vault secret (idempotent — no error if already gone).
		if err := deps.Vault.DeleteSecret(r.Context(), tx, result.VaultSecretID); err != nil {
			log.Printf("[%s] DeleteCredential: vault delete: %v", TraceID(r.Context()), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to delete credential"))
			return
		}

		if owned {
			if err := db.CommitTx(r.Context(), tx); err != nil {
				log.Printf("[%s] DeleteCredential: commit: %v", TraceID(r.Context()), err)
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

