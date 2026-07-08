package api

import (
	"context"
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

type agentWebhookOpError struct {
	Status int
	Body   ErrorResponse
}

func (e *agentWebhookOpError) Error() string {
	if e == nil {
		return ""
	}
	return e.Body.Error.Message
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
		agent := AuthenticatedAgent(r.Context())

		var req setAgentWebhookRequest
		if !DecodeJSONOrReject(w, r, &req) {
			return
		}
		if !ValidateRequest(w, r, &req) {
			return
		}

		resp, opErr := setAgentWebhookCore(r.Context(), deps, agent.AgentID, req.URL, req.Token)
		if opErr != nil {
			RespondError(w, r, opErr.Status, opErr.Body)
			return
		}
		RespondJSON(w, http.StatusOK, resp)
	}
}

func handleGetAgentWebhook(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())
		runTest := r.URL.Query().Get("test") == "true"

		resp, opErr := getAgentWebhookCore(r.Context(), deps, agent.AgentID, runTest)
		if opErr != nil {
			RespondError(w, r, opErr.Status, opErr.Body)
			return
		}
		RespondJSON(w, http.StatusOK, resp)
	}
}

func handleDeleteAgentWebhook(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := AuthenticatedAgent(r.Context())
		if opErr := deleteAgentWebhookCore(r.Context(), deps, agent.AgentID); opErr != nil {
			RespondError(w, r, opErr.Status, opErr.Body)
			return
		}
		RespondJSON(w, http.StatusOK, map[string]bool{"cleared": true})
	}
}

func setAgentWebhookCore(ctx context.Context, deps *Deps, agentID int64, rawURL, rawToken string) (*agentWebhookStatusResponse, *agentWebhookOpError) {
	if deps.Vault == nil {
		return nil, &agentWebhookOpError{
			Status: http.StatusServiceUnavailable,
			Body:   ServiceUnavailable("Vault not available"),
		}
	}

	url := strings.TrimSpace(rawURL)
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return nil, &agentWebhookOpError{
			Status: http.StatusBadRequest,
			Body:   BadRequest(ErrInvalidRequest, "token is required"),
		}
	}
	if err := connectors.ValidatePrivateNetworkURL(url, "url"); err != nil {
		var valErr *connectors.ValidationError
		if errors.As(err, &valErr) {
			return nil, &agentWebhookOpError{
				Status: http.StatusBadRequest,
				Body:   BadRequest(ErrInvalidWebhookURL, valErr.Message),
			}
		}
		return nil, &agentWebhookOpError{
			Status: http.StatusBadRequest,
			Body:   BadRequest(ErrInvalidWebhookURL, err.Error()),
		}
	}

	prevCfg, err := db.GetAgentWebhookConfig(ctx, deps.DB, agentID)
	if err != nil {
		log.Printf("[%s] setAgentWebhookCore: load config: %v", TraceID(ctx), err)
		CaptureError(ctx, err)
		return nil, &agentWebhookOpError{
			Status: http.StatusInternalServerError,
			Body:   InternalError("Failed to save webhook"),
		}
	}

	secretName := "agent_webhook_" + strconv.FormatInt(agentID, 10)
	vaultID, err := deps.Vault.CreateSecret(ctx, deps.DB, secretName, []byte(token))
	if err != nil {
		log.Printf("[%s] setAgentWebhookCore: vault create: %v", TraceID(ctx), err)
		CaptureError(ctx, err)
		return nil, &agentWebhookOpError{
			Status: http.StatusInternalServerError,
			Body:   InternalError("Failed to save webhook token"),
		}
	}

	if err := db.SetAgentWebhook(ctx, deps.DB, agentID, url, vaultID); err != nil {
		_ = deps.Vault.DeleteSecret(ctx, deps.DB, vaultID)
		log.Printf("[%s] setAgentWebhookCore: update agent: %v", TraceID(ctx), err)
		CaptureError(ctx, err)
		return nil, &agentWebhookOpError{
			Status: http.StatusInternalServerError,
			Body:   InternalError("Failed to save webhook"),
		}
	}

	if prevCfg != nil && prevCfg.WebhookTokenVaultID != nil && *prevCfg.WebhookTokenVaultID != "" && *prevCfg.WebhookTokenVaultID != vaultID {
		if delErr := deps.Vault.DeleteSecret(ctx, deps.DB, *prevCfg.WebhookTokenVaultID); delErr != nil {
			log.Printf("[%s] setAgentWebhookCore: delete old vault secret: %v", TraceID(ctx), delErr)
		}
	}

	testCtx, cancel := contextWithTimeout(ctx, 15*time.Second)
	defer cancel()
	testResult, err := deliverAgentWakeTest(testCtx, deps, agentID)
	if err != nil {
		log.Printf("[%s] setAgentWebhookCore: test delivery: %v", TraceID(ctx), err)
	}

	resp := agentWebhookStatusResponse{
		Configured: true,
		WebhookURL: url,
		Test:       testResult,
	}
	populateWebhookSharedURLWarning(ctx, deps, &resp, agentID, url)
	return &resp, nil
}

func getAgentWebhookCore(ctx context.Context, deps *Deps, agentID int64, runTest bool) (*agentWebhookStatusResponse, *agentWebhookOpError) {
	cfg, err := db.GetAgentWebhookConfig(ctx, deps.DB, agentID)
	if err != nil {
		log.Printf("[%s] getAgentWebhookCore: %v", TraceID(ctx), err)
		CaptureError(ctx, err)
		return nil, &agentWebhookOpError{
			Status: http.StatusInternalServerError,
			Body:   InternalError("Failed to load webhook"),
		}
	}

	resp := agentWebhookStatusResponse{}
	if cfg != nil && cfg.WebhookURL != nil && *cfg.WebhookURL != "" {
		resp.Configured = true
		resp.WebhookURL = *cfg.WebhookURL
		populateWebhookSharedURLWarning(ctx, deps, &resp, agentID, *cfg.WebhookURL)
	}

	if runTest {
		testCtx, cancel := contextWithTimeout(ctx, 15*time.Second)
		defer cancel()
		testResult, err := deliverAgentWakeTest(testCtx, deps, agentID)
		if err != nil {
			log.Printf("[%s] getAgentWebhookCore: test: %v", TraceID(ctx), err)
			CaptureError(ctx, err)
			return nil, &agentWebhookOpError{
				Status: http.StatusInternalServerError,
				Body:   InternalError("Webhook test failed"),
			}
		}
		resp.Test = testResult
	}

	return &resp, nil
}

func deleteAgentWebhookCore(ctx context.Context, deps *Deps, agentID int64) *agentWebhookOpError {
	prevVaultID, err := db.ClearAgentWebhook(ctx, deps.DB, agentID)
	if err != nil {
		log.Printf("[%s] deleteAgentWebhookCore: %v", TraceID(ctx), err)
		CaptureError(ctx, err)
		return &agentWebhookOpError{
			Status: http.StatusInternalServerError,
			Body:   InternalError("Failed to clear webhook"),
		}
	}
	if deps.Vault != nil && prevVaultID != nil && *prevVaultID != "" {
		if delErr := deps.Vault.DeleteSecret(ctx, deps.DB, *prevVaultID); delErr != nil {
			log.Printf("[%s] deleteAgentWebhookCore: vault delete: %v", TraceID(ctx), delErr)
		}
	}
	return nil
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
