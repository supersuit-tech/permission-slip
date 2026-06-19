package imessage

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type sendMessageAction struct {
	conn *IMessageConnector
}

type sendMessageParams struct {
	To             []Handle `json:"to"`
	From           []Handle `json:"from"`
	ChatID         int      `json:"chat_id"`
	ChatIdentifier string   `json:"chat_identifier"`
	ChatGUID       string   `json:"chat_guid"`
	Text           string   `json:"text"`
	File           string   `json:"file"`
	Service        string   `json:"service"`
	NoSMSFallback  *bool    `json:"no_sms_fallback"`
	RetryGUID      string   `json:"retry_guid"`
}

func (p *sendMessageParams) validate() error {
	hasChatRef := p.ChatID > 0 || p.ChatIdentifier != "" || p.ChatGUID != ""
	hasTo := len(p.To) > 0
	if !hasChatRef && !hasTo {
		return &connectors.ValidationError{Message: "provide chat_id, chat_identifier, chat_guid, or to handles"}
	}
	if hasChatRef && hasTo {
		return &connectors.ValidationError{Message: "provide either a chat reference or to handles, not both"}
	}
	if hasTo {
		if len(p.To) > 1 {
			return &connectors.ValidationError{
				Message: "sending to multiple handles without an existing chat is not supported; use chat_id or chat_guid for group chats",
			}
		}
		if _, err := NormalizeHandles(p.To); err != nil {
			return err
		}
	}
	if p.From != nil {
		if _, err := NormalizeHandles(p.From); err != nil {
			return err
		}
	}
	if strings.TrimSpace(p.Text) == "" && strings.TrimSpace(p.File) == "" {
		return &connectors.ValidationError{Message: "at least one of text or file is required"}
	}
	if p.Service == "" {
		p.Service = defaultSendService
	}
	switch p.Service {
	case "imessage", "sms", "auto":
	default:
		return &connectors.ValidationError{Message: "service must be imessage, sms, or auto"}
	}
	return nil
}

func (a *sendMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendMessageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: "invalid parameters: " + err.Error()}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	actionCtx, cancel := a.conn.actionTimeout(ctx)
	defer cancel()

	if params.RetryGUID != "" {
		if existing, ok, err := tryIdempotentSend(actionCtx, a.conn.client, req.Credentials, params.RetryGUID); err != nil {
			return nil, err
		} else if ok {
			return connectors.JSONResult(existing)
		}
	}

	rpcParams := buildSendRPCParams(params)
	var result sendResult
	if err := a.conn.client.rpcCall(actionCtx, req.Credentials, "send", rpcParams, &result); err != nil {
		return nil, err
	}

	response := map[string]any{
		"ok":   result.OK,
		"id":   result.ID,
		"guid": result.GUID,
	}
	if result.GUID != "" {
		delivery, err := verifySendDelivery(actionCtx, a.conn.client, req.Credentials, result.GUID)
		if err != nil {
			return nil, err
		}
		if delivery != nil {
			response["send_state"] = delivery.SendState
			response["service"] = delivery.Service
			response["delivery"] = delivery
		}
	}
	return connectors.JSONResult(response)
}

func buildSendRPCParams(p sendMessageParams) map[string]any {
	params := map[string]any{}
	if p.ChatID > 0 {
		params["chat_id"] = p.ChatID
	} else if p.ChatGUID != "" {
		params["chat_guid"] = p.ChatGUID
	} else if p.ChatIdentifier != "" {
		params["chat_identifier"] = p.ChatIdentifier
	} else if len(p.To) == 1 {
		params["to"] = p.To[0].Value
	}
	if p.Text != "" {
		params["text"] = p.Text
	}
	if p.File != "" {
		params["file"] = p.File
	}
	service := p.Service
	if service == "" {
		service = defaultSendService
	}
	params["service"] = service
	if p.NoSMSFallback != nil && *p.NoSMSFallback {
		params["no_sms_fallback"] = true
	}
	return params
}

// sendPreviewTarget returns a human-readable target for approval previews.
func sendPreviewTarget(p sendMessageParams) string {
	if p.ChatID > 0 {
		return fmt.Sprintf("chat:%d", p.ChatID)
	}
	if p.ChatGUID != "" {
		return p.ChatGUID
	}
	if p.ChatIdentifier != "" {
		return p.ChatIdentifier
	}
	if len(p.To) == 1 {
		return p.To[0].Value
	}
	return ""
}
