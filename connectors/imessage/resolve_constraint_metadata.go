package imessage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ResolveConstraintMetadata returns verified from/to handles for permission matching.
func (c *IMessageConnector) ResolveConstraintMetadata(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	switch actionType {
	case "imessage.send_message":
		return c.resolveSendConstraintMetadata(ctx, params, creds)
	default:
		return nil, connectors.ErrConstraintMetadataUnavailable
	}
}

func (c *IMessageConnector) resolveSendConstraintMetadata(ctx context.Context, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	var sendParams sendMessageParams
	if err := json.Unmarshal(params, &sendParams); err != nil {
		return nil, connectors.ErrConstraintMetadataUnavailable
	}
	if err := sendParams.validate(); err != nil {
		return nil, connectors.ErrConstraintMetadataUnavailable
	}

	actionCtx, cancel := c.actionTimeout(ctx)
	defer cancel()

	fromHandles, err := resolveFromHandles(actionCtx, c.client, creds)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve from account: %v", connectors.ErrConstraintMetadataUnavailable, err)
	}

	toHandles, err := resolveToHandles(actionCtx, c.client, creds, sendParams)
	if err != nil {
		return nil, fmt.Errorf("%w: resolve to handles: %v", connectors.ErrConstraintMetadataUnavailable, err)
	}

	chatObj, _ := resolveChatForSend(actionCtx, c.client, creds, sendParams)
	deliveryService, deliveryDisclosure := resolveDeliveryDisclosure(sendParams, chatObj)

	result := map[string]any{
		"from":                handlesToMaps(fromHandles),
		"to":                  handlesToMaps(toHandles),
		"delivery_service":    deliveryService,
		"delivery_disclosure": deliveryDisclosure,
	}
	if len(fromHandles) == 1 {
		result["from_handle"] = fromHandles[0].Value
	}
	if len(toHandles) == 1 {
		result["to_handle"] = toHandles[0].Value
	}
	return result, nil
}

func resolveFromHandles(ctx context.Context, client *imsgClient, creds connectors.Credentials) ([]Handle, error) {
	lines, err := client.runCLI(ctx, creds, "account", "--json")
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, fmt.Errorf("no account info")
	}
	var info accountInfo
	if err := json.Unmarshal(lines[0], &info); err != nil {
		return nil, err
	}
	if info.AccountLogin != "" {
		return HandlesFromRaws([]string{info.AccountLogin})
	}
	if len(info.Aliases) > 0 {
		return HandlesFromRaws(info.Aliases[:1])
	}
	return nil, fmt.Errorf("account login unavailable")
}

func resolveToHandles(ctx context.Context, client *imsgClient, creds connectors.Credentials, p sendMessageParams) ([]Handle, error) {
	if len(p.To) > 0 {
		return NormalizeHandles(p.To)
	}

	chatObj, err := resolveChatForSend(ctx, client, creds, p)
	if err != nil {
		return nil, err
	}
	if chatObj == nil {
		return nil, fmt.Errorf("chat not found")
	}

	if chatObj.IsGroup {
		handles, err := HandlesFromRaws(chatObj.Participants)
		if err != nil {
			return nil, err
		}
		if chatObj.GUID != "" {
			groupHandle, err := HandleFromRaw(chatObj.GUID)
			if err == nil {
				handles = append(handles, groupHandle)
			}
		}
		return handles, nil
	}
	if len(chatObj.Participants) > 0 {
		return HandlesFromRaws(chatObj.Participants[:1])
	}
	if chatObj.Identifier != "" {
		return HandlesFromRaws([]string{chatObj.Identifier})
	}
	return nil, fmt.Errorf("chat has no participants")
}

func handlesToMaps(handles []Handle) []map[string]string {
	out := make([]map[string]string, 0, len(handles))
	for _, h := range handles {
		out = append(out, map[string]string{"type": h.Type, "value": h.Value})
	}
	return out
}

var _ connectors.ConstraintMetadataResolver = (*IMessageConnector)(nil)
