package api

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
)

type setAgentWebhookRequest struct {
	URL   string `json:"url" validate:"required"`
	Token string `json:"token" validate:"required"`
}

const agentWebhookSharedURLWarning = "Another of your agents is registered with this same webhook URL. Wakes without a session_key are delivered to the gateway's main session and may reach the wrong agent. Include session_key in approval context, or give each agent its own gateway."

type agentWebhookStatusResponse struct {
	Configured bool                    `json:"configured"`
	WebhookURL string                  `json:"webhook_url,omitempty"`
	Warning    string                  `json:"warning,omitempty"`
	Test       *agentWebhookTestResult `json:"test,omitempty"`
}

type agentWebhookTestResult struct {
	Configured bool   `json:"configured"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	WebhookURL string `json:"webhook_url,omitempty"`
	LatencyMS  int64  `json:"latency_ms,omitempty"`
}

func init() {
	RegisterRouteGroup(RegisterAgentWebhookRoutes)
}

func RegisterAgentWebhookRoutes(mux *http.ServeMux, deps *Deps) {
	requireAgent := RequireAgentSignature(deps)
	mux.Handle("PUT /agent/webhook", requireAgent(handlePutAgentWebhook(deps)))
	mux.Handle("GET /agent/webhook", requireAgent(handleGetAgentWebhook(deps)))
	mux.Handle("DELETE /agent/webhook", requireAgent(handleDeleteAgentWebhook(deps)))
}

func handlePutAgentWebhook(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Vault == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Vault not available"))
			return
		}
		agent := AuthenticatedAgent(r.Context())

		var req setAgentWebhookRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		req.URL = strings.TrimSpace(req.URL)
		req.Token = strings.TrimSpace(req.Token)
		if req.Token == "" {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "token is required"))
			return
		}
		if err := connectors.ValidatePrivateNetworkURL(req.URL, "url"); err != nil {
			var valErr *connectors.ValidationError
			if errors.As(err, &valErr) {
				RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidWebhookURL, valErr.Message))
				return
			}
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidWebhookURL, err.Error()))
			return
		}

		prevCfg, err := db.GetAgentWebhookConfig(r.Context(), deps.DB, agent.AgentID)
		if err != nil {
			log.Printf("[%s] PutAgentWebhook: load config: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save webhook"))
			return
		}

		secretName := "agent_webhook_" + strconv.FormatInt(agent.AgentID, 10)
		vaultID, err := deps.Vault.CreateSecret(r.Context(), deps.DB, secretName, []byte(req.Token))
		if err != nil {
			log.Printf("[%s] PutAgentWebhook: vault create: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save webhook token"))
			return
		}

		if err := db.SetAgentWebhook(r.Context(), deps.DB, agent.AgentID, req.URL, vaultID); err != nil {
			_ = deps.Vault.DeleteSecret(r.Context(), deps.DB, vaultID)
			log.Printf("[%s] PutAgentWebhook: update agent: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to save webhook"))
			return
		}

		if prevCfg != nil && prevCfg.WebhookTokenVaultID != nil && *prevCfg.WebhookTokenVaultID != "" && *prevCfg.WebhookTokenVaultID != vaultID {
			if delErr := deps.Vault.DeleteSecret(r.Context(), deps.DB, *prevCfg.WebhookTokenVaultID); delErr != nil {
				log.Printf("[%s] PutAgentWebhook: delete old vault secret: %v", TraceID(r.Context()), delErr)
			}
		}

		testCtx, cancel := contextWithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		testResult, err := deliverAgentWakeTest(testCtx, deps, agent.AgentID)
		if err != nil {
			log.Printf("[%s] PutAgentWebhook: test delivery: %v", TraceID(r.Context()), err)
		}

		resp := agentWebhookStatusResponse{
			Configured: true,
			WebhookURL: req.URL,
			Test:       testResult,
		}
		populateWebhookSharedURLWarning(r.Context(), deps, &resp, agent.AgentID, req.URL)
		RespondJSON(w, http.StatusOK, resp)
	}
}

func handleGetAgentWebhook(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())
		cfg, err := db.GetAgentWebhookConfig(r.Context(), deps.DB, agent.AgentID)
		if err != nil {
			log.Printf("[%s] GetAgentWebhook: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to load webhook"))
			return
		}
		resp := agentWebhookStatusResponse{}
		if cfg != nil && cfg.WebhookURL != nil && *cfg.WebhookURL != "" {
			resp.Configured = true
			resp.WebhookURL = *cfg.WebhookURL
			populateWebhookSharedURLWarning(r.Context(), deps, &resp, agent.AgentID, *cfg.WebhookURL)
		}

		if r.URL.Query().Get("test") == "true" {
			testCtx, cancel := contextWithTimeout(r.Context(), 15*time.Second)
			defer cancel()
			testResult, err := deliverAgentWakeTest(testCtx, deps, agent.AgentID)
			if err != nil {
				log.Printf("[%s] GetAgentWebhook: test: %v", TraceID(r.Context()), err)
				CaptureError(r.Context(), err)
				RespondError(w, r, http.StatusInternalServerError, InternalError("Webhook test failed"))
				return
			}
			resp.Test = testResult
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

func handleDeleteAgentWebhook(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())
		prevVaultID, err := db.ClearAgentWebhook(r.Context(), deps.DB, agent.AgentID)
		if err != nil {
			log.Printf("[%s] DeleteAgentWebhook: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to clear webhook"))
			return
		}
		if deps.Vault != nil && prevVaultID != nil && *prevVaultID != "" {
			if delErr := deps.Vault.DeleteSecret(r.Context(), deps.DB, *prevVaultID); delErr != nil {
				log.Printf("[%s] DeleteAgentWebhook: vault delete: %v", TraceID(r.Context()), delErr)
			}
		}
		RespondJSON(w, http.StatusOK, map[string]bool{"cleared": true})
	}
}

func populateWebhookSharedURLWarning(ctx context.Context, deps *Deps, resp *agentWebhookStatusResponse, agentID int64, webhookURL string) {
	shared, err := db.WebhookURLSharedByOtherAgent(ctx, deps.DB, agentID, webhookURL)
	if err != nil {
		log.Printf("[%s] webhook shared URL check: %v", TraceID(ctx), err)
		return
	}
	if shared {
		resp.Warning = agentWebhookSharedURLWarning
	}
}
