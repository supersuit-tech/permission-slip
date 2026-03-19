package slack

import (
	"encoding/json"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// This file implements connectors.RequestValidator for Slack actions that have
// parameter format checks (channel ID, user ID, message timestamp). These
// validations run at approval request time so the agent gets an immediate error
// instead of the approval failing after the user approves it.

// channelParams is a minimal struct for extracting the channel field.
type channelParams struct {
	Channel string `json:"channel"`
}

// channelAndTSParams extracts channel + ts fields.
type channelAndTSParams struct {
	Channel string `json:"channel"`
	TS      string `json:"ts"`
}

// inviteParams extracts channel + users fields for invite_to_channel.
type inviteParams struct {
	Channel string `json:"channel"`
	Users   string `json:"users"`
}

// userIDParams extracts the user_id field.
type userIDParams struct {
	UserID string `json:"user_id"`
}

// validateChannelFromRaw unmarshals params and validates the channel ID format.
func validateChannelFromRaw(params json.RawMessage) error {
	var p channelParams
	if err := json.Unmarshal(params, &p); err != nil {
		return &connectors.ValidationError{Message: "invalid parameters: " + err.Error()}
	}
	if p.Channel == "" {
		return nil // required-field check is handled by schema validation
	}
	return validateChannelID(p.Channel)
}

// validateChannelAndTSFromRaw validates both channel ID and message timestamp.
func validateChannelAndTSFromRaw(params json.RawMessage) error {
	var p channelAndTSParams
	if err := json.Unmarshal(params, &p); err != nil {
		return &connectors.ValidationError{Message: "invalid parameters: " + err.Error()}
	}
	if p.Channel != "" {
		if err := validateChannelID(p.Channel); err != nil {
			return err
		}
	}
	if p.TS != "" {
		if err := validateMessageTS(p.TS); err != nil {
			return err
		}
	}
	return nil
}

// --- Channel-only actions ---

func (a *readChannelMessagesAction) ValidateRequest(params json.RawMessage) error {
	return validateChannelFromRaw(params)
}

func (a *sendMessageAction) ValidateRequest(params json.RawMessage) error {
	return validateChannelFromRaw(params)
}

func (a *scheduleMessageAction) ValidateRequest(params json.RawMessage) error {
	return validateChannelFromRaw(params)
}

func (a *inviteToChannelAction) ValidateRequest(params json.RawMessage) error {
	var p inviteParams
	if err := json.Unmarshal(params, &p); err != nil {
		return &connectors.ValidationError{Message: "invalid parameters: " + err.Error()}
	}
	if p.Channel != "" {
		if err := validateChannelID(p.Channel); err != nil {
			return err
		}
	}
	if p.Users != "" {
		for _, uid := range strings.Split(p.Users, ",") {
			uid = strings.TrimSpace(uid)
			if uid == "" {
				continue
			}
			if err := validateUserID(uid); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *setTopicAction) ValidateRequest(params json.RawMessage) error {
	return validateChannelFromRaw(params)
}

func (a *readThreadAction) ValidateRequest(params json.RawMessage) error {
	return validateChannelFromRaw(params)
}

func (a *uploadFileAction) ValidateRequest(params json.RawMessage) error {
	return validateChannelFromRaw(params)
}

// --- Channel + timestamp actions ---

func (a *addReactionAction) ValidateRequest(params json.RawMessage) error {
	return validateChannelAndTSFromRaw(params)
}

func (a *updateMessageAction) ValidateRequest(params json.RawMessage) error {
	return validateChannelAndTSFromRaw(params)
}

func (a *deleteMessageAction) ValidateRequest(params json.RawMessage) error {
	return validateChannelAndTSFromRaw(params)
}

// --- User ID actions ---

func (a *sendDMAction) ValidateRequest(params json.RawMessage) error {
	var p userIDParams
	if err := json.Unmarshal(params, &p); err != nil {
		return &connectors.ValidationError{Message: "invalid parameters: " + err.Error()}
	}
	if p.UserID == "" {
		return nil // required-field check is handled by schema validation
	}
	return validateUserID(p.UserID)
}
