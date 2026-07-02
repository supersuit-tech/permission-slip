package imessage

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	defaultSendService      = "auto"
	deliveryPollInterval    = 500 * time.Millisecond
	deliveryPollMaxAttempts = 10
	sendStatusDelivered     = "delivered"
	sendStatusSent          = "sent"
	sendStatusFailed        = "failed"
	sendStatusPending       = "pending"
)

type sendStatusResult struct {
	OK           bool           `json:"ok"`
	GUID         string         `json:"guid"`
	SendState    string         `json:"send_state"`
	Service      string         `json:"service,omitempty"`
	CheckedAt    string         `json:"checked_at,omitempty"`
	DeliveredAt  string         `json:"delivered_at,omitempty"`
	StatusFields map[string]any `json:"status_fields,omitempty"`
}

// resolveDeliveryDisclosure predicts how a send will be routed for approval UI.
func resolveDeliveryDisclosure(p sendMessageParams, chatObj *chat) (service string, disclosure string) {
	effective := p.Service
	if effective == "" {
		effective = defaultSendService
	}
	noFallback := p.NoSMSFallback != nil && *p.NoSMSFallback

	if chatObj != nil {
		chatService := strings.ToUpper(strings.TrimSpace(chatObj.Service))
		switch effective {
		case "sms":
			return "sms", "Will send as SMS via relay"
		case "imessage":
			return "imessage", "Will send as iMessage"
		case "auto":
			if chatService == "SMS" {
				return "sms", "Will send as SMS via relay"
			}
			return "imessage", "Will send as iMessage"
		}
	}

	switch effective {
	case "sms":
		return "sms", "Will send as SMS via relay"
	case "imessage":
		if noFallback {
			return "imessage", "Will send as iMessage (no SMS fallback)"
		}
		return "imessage", "Will send as iMessage"
	case "auto":
		if noFallback {
			return "imessage", "Will send as iMessage (no SMS fallback)"
		}
		return "auto", "Will send as iMessage; falls back to SMS via relay if unavailable"
	default:
		return effective, ""
	}
}

func fetchSendStatus(ctx context.Context, client *imsgClient, creds connectors.Credentials, guid string) (*sendStatusResult, error) {
	var result sendStatusResult
	if err := client.rpcCall(ctx, creds, "message.send_status", map[string]any{"guid": guid}, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func isTerminalSendState(state string) bool {
	switch state {
	case sendStatusDelivered, sendStatusSent, sendStatusFailed:
		return true
	default:
		return false
	}
}

// verifySendDelivery polls message.send_status until delivery completes or times out.
// Returns an error only for explicit delivery failure or request cancellation.
// Transport errors during polling are treated as non-fatal (caller should report unknown state).
func verifySendDelivery(ctx context.Context, client *imsgClient, creds connectors.Credentials, guid string) (*sendStatusResult, error) {
	var last *sendStatusResult
	for attempt := 0; attempt < deliveryPollMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.Canceled) {
				return last, &connectors.CanceledError{Message: err.Error()}
			}
			return last, nil
		}
		status, err := fetchSendStatus(ctx, client, creds, guid)
		if err != nil {
			return last, nil
		}
		last = status
		if status.SendState == sendStatusFailed {
			return status, &connectors.ExternalError{
				Message: "message delivery failed — SMS relay may be offline or the recipient is unreachable",
			}
		}
		if isTerminalSendState(status.SendState) {
			return status, nil
		}
		if attempt+1 < deliveryPollMaxAttempts {
			select {
			case <-ctx.Done():
				if errors.Is(ctx.Err(), context.Canceled) {
					return last, &connectors.CanceledError{Message: ctx.Err().Error()}
				}
				return last, nil
			case <-time.After(deliveryPollInterval):
			}
		}
	}
	return last, nil
}

// tryIdempotentSend returns an existing successful send when retry_guid is already delivered.
func tryIdempotentSend(ctx context.Context, client *imsgClient, creds connectors.Credentials, guid string) (map[string]any, bool, error) {
	if guid == "" {
		return nil, false, nil
	}
	status, err := fetchSendStatus(ctx, client, creds, guid)
	if err != nil {
		return nil, false, err
	}
	if status.SendState != sendStatusDelivered && status.SendState != sendStatusSent {
		return nil, false, nil
	}
	return map[string]any{
		"ok":         true,
		"guid":       guid,
		"idempotent": true,
		"send_state": status.SendState,
		"service":    status.Service,
		"delivery":   status,
	}, true, nil
}
