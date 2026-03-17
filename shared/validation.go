// Package shared provides validation constants shared between the Go backend
// and the React frontend. The single source of truth is validation.json in
// this directory; the frontend imports it directly and this package embeds it
// at compile time so both sides stay in sync.
package shared

import (
	_ "embed"
	"encoding/json"
)

//go:embed validation.json
var validationJSON []byte

// fieldLimits mirrors a JSON entry that has a maxLength key.
type fieldLimits struct {
	MinLength *int `json:"minLength,omitempty"`
	MaxLength *int `json:"maxLength,omitempty"`
	Length    *int `json:"length,omitempty"`
	Max      *int `json:"max,omitempty"`
}

type validationConfig struct {
	Username                 fieldLimits `json:"username"`
	AgentName                fieldLimits `json:"agentName"`
	ActionConfigName         fieldLimits `json:"actionConfigName"`
	ActionConfigDescription  fieldLimits `json:"actionConfigDescription"`
	CredentialService        fieldLimits `json:"credentialService"`
	CredentialLabel          fieldLimits `json:"credentialLabel"`
	ActionType               fieldLimits `json:"actionType"`
	ActionVersion            fieldLimits `json:"actionVersion"`
	ConfirmationCode         fieldLimits `json:"confirmationCode"`
	ConstraintsBytes         fieldLimits `json:"constraintsBytes"`
	ParametersBytes          fieldLimits `json:"parametersBytes"`
}

// --- Exported constants ---
// Populated from validation.json at init time so callers get plain int values.

var (
	UsernameMinLength           int
	UsernameMaxLength           int
	AgentNameMaxLength          int
	ActionConfigNameMaxLength   int
	ActionConfigDescMaxLength   int
	CredentialServiceMaxLength  int
	CredentialLabelMaxLength    int
	ActionTypeMaxLength         int
	ActionVersionMaxLength      int
	ConfirmationCodeLength      int
	MaxConstraintsBytes         int
	MaxParametersBytes          int
)

// mustInt dereferences a *int parsed from validation.json, panicking with a
// clear message if the field is missing so misconfigured JSON is caught
// immediately at startup rather than at request time.
func mustInt(path string, v *int) int {
	if v == nil {
		panic("shared: " + path + " is required in validation.json")
	}
	return *v
}

func init() {
	var cfg validationConfig
	if err := json.Unmarshal(validationJSON, &cfg); err != nil {
		panic("shared: failed to parse validation.json: " + err.Error())
	}

	UsernameMinLength = mustInt("username.minLength", cfg.Username.MinLength)
	UsernameMaxLength = mustInt("username.maxLength", cfg.Username.MaxLength)
	AgentNameMaxLength = mustInt("agentName.maxLength", cfg.AgentName.MaxLength)
	ActionConfigNameMaxLength = mustInt("actionConfigName.maxLength", cfg.ActionConfigName.MaxLength)
	ActionConfigDescMaxLength = mustInt("actionConfigDescription.maxLength", cfg.ActionConfigDescription.MaxLength)
	CredentialServiceMaxLength = mustInt("credentialService.maxLength", cfg.CredentialService.MaxLength)
	CredentialLabelMaxLength = mustInt("credentialLabel.maxLength", cfg.CredentialLabel.MaxLength)
	ActionTypeMaxLength = mustInt("actionType.maxLength", cfg.ActionType.MaxLength)
	ActionVersionMaxLength = mustInt("actionVersion.maxLength", cfg.ActionVersion.MaxLength)
	ConfirmationCodeLength = mustInt("confirmationCode.length", cfg.ConfirmationCode.Length)
	MaxConstraintsBytes = mustInt("constraintsBytes.max", cfg.ConstraintsBytes.Max)
	MaxParametersBytes = mustInt("parametersBytes.max", cfg.ParametersBytes.Max)
}
