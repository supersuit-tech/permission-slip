package imessage

import (
	"context"
	"encoding/json"
	"log"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ResolveResourceDetails enriches send approvals with predicted delivery service.
func (c *IMessageConnector) ResolveResourceDetails(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	if actionType != "imessage.send_message" {
		return nil, nil
	}

	var sendParams sendMessageParams
	if err := json.Unmarshal(params, &sendParams); err != nil {
		log.Printf("imessage: resolve send_message details: parse params: %v", err)
		return nil, nil
	}
	if err := sendParams.validate(); err != nil {
		log.Printf("imessage: resolve send_message details: invalid params: %v", err)
		return nil, nil
	}

	actionCtx, cancel := c.actionTimeout(ctx)
	defer cancel()

	chatObj, err := resolveChatForSend(actionCtx, c.client, creds, sendParams)
	if err != nil {
		log.Printf("imessage: resolve send_message details: chat lookup: %v", err)
	}

	service, disclosure := resolveDeliveryDisclosure(sendParams, chatObj)
	details := map[string]any{
		"to":                  sendPreviewTarget(sendParams),
		"delivery_service":    service,
		"delivery_disclosure": disclosure,
	}
	if chatObj != nil && chatObj.Service != "" {
		details["chat_service"] = chatObj.Service
	}
	if chatObj != nil && chatObj.DisplayName != "" {
		details["chat_name"] = chatObj.DisplayName
	} else if chatObj != nil && chatObj.Name != "" {
		details["chat_name"] = chatObj.Name
	}
	return details, nil
}

var _ connectors.ResourceDetailResolver = (*IMessageConnector)(nil)
