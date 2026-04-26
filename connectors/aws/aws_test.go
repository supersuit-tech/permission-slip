package aws

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestAWSConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "aws" {
		t.Errorf("ID() = %q, want %q", got, "aws")
	}
}

func TestAWSConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"aws.describe_instances",
		"aws.start_instance",
		"aws.stop_instance",
		"aws.restart_instance",
		"aws.get_metrics",
		"aws.list_s3_objects",
		"aws.create_presigned_url",
		"aws.describe_rds_instances",
	}
	for _, at := range want {
		if _, ok := actions[at]; !ok {
			t.Errorf("Actions() missing %q", at)
		}
	}
	if len(actions) != len(want) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(want))
	}
}

func TestAWSConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid credentials",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "missing access_key_id",
			creds:   connectors.NewCredentials(map[string]string{"secret_access_key": "secret"}),
			wantErr: true,
		},
		{
			name:    "missing secret_access_key",
			creds:   connectors.NewCredentials(map[string]string{"access_key_id": "AKID"}),
			wantErr: true,
		},
		{
			name:    "missing region",
			creds:   connectors.NewCredentials(map[string]string{"access_key_id": "AKID", "secret_access_key": "secret"}),
			wantErr: true,
		},
		{
			name:    "empty access_key_id",
			creds:   connectors.NewCredentials(map[string]string{"access_key_id": "", "secret_access_key": "secret"}),
			wantErr: true,
		},
		{
			name:    "empty secret_access_key",
			creds:   connectors.NewCredentials(map[string]string{"access_key_id": "AKID", "secret_access_key": ""}),
			wantErr: true,
		},
		{
			name:    "empty credentials",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "zero-value credentials",
			creds:   connectors.Credentials{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := c.ValidateCredentials(t.Context(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("ValidateCredentials() returned %T, want *connectors.ValidationError", err)
			}
		})
	}
}

func TestAWSConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "aws" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "aws")
	}
	if m.Name != "AWS" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "AWS")
	}
	if len(m.Actions) != 8 {
		t.Fatalf("Manifest().Actions has %d items, want 8", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"aws.describe_instances",
		"aws.start_instance",
		"aws.stop_instance",
		"aws.restart_instance",
		"aws.get_metrics",
		"aws.list_s3_objects",
		"aws.create_presigned_url",
		"aws.describe_rds_instances",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "aws" {
		t.Errorf("credential service = %q, want %q", cred.Service, "aws")
	}
	if cred.AuthType != "custom" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "custom")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	// Validate the manifest passes validation.
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestAWSConnector_ActionsMatchManifest(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	manifest := c.Manifest()

	manifestTypes := make(map[string]bool, len(manifest.Actions))
	for _, a := range manifest.Actions {
		manifestTypes[a.ActionType] = true
	}

	for actionType := range actions {
		if !manifestTypes[actionType] {
			t.Errorf("Actions() has %q but Manifest() does not", actionType)
		}
	}
	for _, a := range manifest.Actions {
		if _, ok := actions[a.ActionType]; !ok {
			t.Errorf("Manifest() has %q but Actions() does not", a.ActionType)
		}
	}
}

func TestAWSConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*AWSConnector)(nil)
	var _ connectors.ManifestProvider = (*AWSConnector)(nil)
}

func TestParseServiceHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		host        string
		wantService string
		wantRegion  string
	}{
		{"ec2.us-east-1.amazonaws.com", "ec2", "us-east-1"},
		{"s3.us-west-2.amazonaws.com", "s3", "us-west-2"},
		{"monitoring.eu-west-1.amazonaws.com", "monitoring", "eu-west-1"},
		{"rds.ap-southeast-1.amazonaws.com", "rds", "ap-southeast-1"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			t.Parallel()
			service, region := parseServiceHost(tt.host)
			if service != tt.wantService {
				t.Errorf("service = %q, want %q", service, tt.wantService)
			}
			if region != tt.wantRegion {
				t.Errorf("region = %q, want %q", region, tt.wantRegion)
			}
		})
	}
}
