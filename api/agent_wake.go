package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/agentwake"
	"github.com/supersuit-tech/permission-slip/db"
)

// agentWakeHTTPClient is used for outbound wake webhooks to private agent networks.
// Redirects are not followed — the destination URL must be the final POST target.
var agentWakeHTTPClient = &http.Client{
	Timeout: 10 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

// notifyAgentApprovalResolved POSTs a wake to the agent's configured provider
// when a webhook is set. Fire-and-forget with retries; no-op when webhook is not set.
func notifyAgentApprovalResolved(deps *Deps, appr *db.Approval) {
	NotifyAgentApprovalResolvedSync(deps, appr)
}

// NotifyAgentApprovalResolvedSync is exported for background jobs that dispatch expiry wakes.
func NotifyAgentApprovalResolvedSync(deps *Deps, appr *db.Approval) {
	if deps == nil || deps.DB == nil || appr == nil {
		return
	}
	status := resolvedApprovalStatus(*appr)
	notifyAgentWake(deps, appr, status)
}

// notifyAgentWake delivers a wake for the given approval outcome.
func notifyAgentWake(deps *Deps, appr *db.Approval, status string) {
	if appr == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		delivery, err := buildAgentWakeDelivery(ctx, deps, appr, status)
		if err != nil || delivery == nil {
			return
		}
		agentwake.DeliverAsync(deps.Logger, agentWakeHTTPClient, delivery,
			slog.Int64("agent_id", appr.AgentID),
			slog.String("approval_id", appr.ApprovalID),
			slog.String("status", status),
		)
	}()
}

// deliverAgentWakeTest sends a test wake synchronously and returns the delivery result.
func deliverAgentWakeTest(ctx context.Context, deps *Deps, agentID int64) (*agentWebhookTestResult, error) {
	cfg, token, err := loadAgentWakeCredentials(ctx, deps, agentID)
	if err != nil {
		return nil, err
	}
	if cfg == nil || cfg.WebhookURL == nil || *cfg.WebhookURL == "" {
		return &agentWebhookTestResult{
			Configured: false,
			Success:    false,
			Message:    "No webhook configured",
		}, nil
	}
	delivery, err := agentwake.BuildTestDelivery(cfg.WebhookProvider, *cfg.WebhookURL, token, agentID)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	err = agentwake.DeliverPOST(ctx, agentWakeHTTPClient, delivery)
	result := &agentWebhookTestResult{
		Configured: true,
		WebhookURL: *cfg.WebhookURL,
	}
	if err != nil {
		result.Success = false
		result.Message = err.Error()
		return result, nil
	}
	result.Success = true
	result.Message = "Test wake delivered successfully"
	result.LatencyMS = time.Since(start).Milliseconds()
	return result, nil
}

func buildAgentWakeDelivery(ctx context.Context, deps *Deps, appr *db.Approval, status string) (*agentwake.Delivery, error) {
	if appr == nil {
		return nil, nil
	}

	webhookURL, provider, err := resolveApprovalWakeTarget(appr)
	if err != nil {
		if deps.Logger != nil {
			deps.Logger.Warn("agent wake: resolve target failed",
				slog.Int64("agent_id", appr.AgentID),
				slog.String("approval_id", appr.ApprovalID),
				slog.String("error", err.Error()))
		}
		return nil, err
	}

	cfg, token, err := loadAgentWakeCredentials(ctx, deps, appr.AgentID)
	if err != nil {
		if deps.Logger != nil {
			deps.Logger.Warn("agent wake: load credentials failed",
				slog.Int64("agent_id", appr.AgentID), slog.String("error", err.Error()))
		}
		return nil, err
	}

	if webhookURL == "" {
		if cfg == nil || cfg.WebhookURL == nil || *cfg.WebhookURL == "" {
			return nil, nil
		}
		webhookURL = *cfg.WebhookURL
		provider = cfg.WebhookProvider
		if provider == "" {
			provider = agentwake.ProviderOpenClaw
		}
	}

	if strings.TrimSpace(webhookURL) == "" {
		return nil, nil
	}

	return agentwake.BuildDelivery(provider, webhookURL, token, agentwake.WakeRequest{
		ApprovalID: appr.ApprovalID,
		Status:     status,
		SessionKey: agentwake.SessionKeyFromApprovalContext(appr.Context),
		AgentID:    appr.AgentID,
	})
}

// resolveApprovalWakeTarget returns the per-request callback URL when set on the approval.
func resolveApprovalWakeTarget(appr *db.Approval) (webhookURL, provider string, err error) {
	if appr.WakeCallbackURL == nil {
		return "", "", nil
	}
	url := strings.TrimSpace(*appr.WakeCallbackURL)
	if url == "" {
		return "", "", nil
	}
	provider, err = agentwake.InferProviderFromCallbackURL(url)
	if err != nil {
		return "", "", err
	}
	return url, provider, nil
}

func loadAgentWakeCredentials(ctx context.Context, deps *Deps, agentID int64) (*db.AgentWebhookConfig, string, error) {
	cfg, err := db.GetAgentWebhookConfig(ctx, deps.DB, agentID)
	if err != nil {
		return nil, "", err
	}
	if deps.Vault == nil || cfg == nil || cfg.WebhookTokenVaultID == nil || *cfg.WebhookTokenVaultID == "" {
		return cfg, "", nil
	}
	secret, err := deps.Vault.ReadSecret(ctx, deps.DB, *cfg.WebhookTokenVaultID)
	if err != nil {
		return nil, "", err
	}
	return cfg, string(secret), nil
}
