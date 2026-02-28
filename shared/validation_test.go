package shared

import (
	"testing"
	"time"
)

func TestValidationConfigLoads(t *testing.T) {
	tests := []struct {
		name string
		got  int
		want int
	}{
		{"UsernameMinLength", UsernameMinLength, 3},
		{"UsernameMaxLength", UsernameMaxLength, 32},
		{"AgentNameMaxLength", AgentNameMaxLength, 256},
		{"ActionConfigNameMaxLength", ActionConfigNameMaxLength, 255},
		{"ActionConfigDescMaxLength", ActionConfigDescMaxLength, 4096},
		{"CredentialServiceMaxLength", CredentialServiceMaxLength, 128},
		{"CredentialLabelMaxLength", CredentialLabelMaxLength, 255},
		{"ActionTypeMaxLength", ActionTypeMaxLength, 128},
		{"ActionVersionMaxLength", ActionVersionMaxLength, 10},
		{"ConfirmationCodeLength", ConfirmationCodeLength, 6},
		{"MaxConstraintsBytes", MaxConstraintsBytes, 16384},
		{"MaxParametersBytes", MaxParametersBytes, 16384},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}

	wantDuration := 90 * 24 * time.Hour
	if StandingApprovalMaxDuration != wantDuration {
		t.Errorf("StandingApprovalMaxDuration = %v, want %v", StandingApprovalMaxDuration, wantDuration)
	}
}
