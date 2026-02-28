package api

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidateStruct_Required(t *testing.T) {
	t.Parallel()
	type req struct {
		Name string `json:"name" validate:"required"`
	}

	// Missing required field.
	err := ValidateStruct(&req{})
	if err == nil {
		t.Fatal("expected error for missing required field")
	}
	if got := err.Error(); got != "name is required" {
		t.Errorf("got %q, want %q", got, "name is required")
	}

	// Present.
	err = ValidateStruct(&req{Name: "hello"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateStruct_GT(t *testing.T) {
	t.Parallel()
	type req struct {
		AgentID int64 `json:"agent_id" validate:"gt=0"`
	}

	tests := []struct {
		name    string
		val     int64
		wantErr string
	}{
		{"zero", 0, "agent_id must be a positive integer"},
		{"negative", -1, "agent_id must be a positive integer"},
		{"valid", 1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateStruct(&req{AgentID: tt.val})
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			if got := err.Error(); got != tt.wantErr {
				t.Errorf("got %q, want %q", got, tt.wantErr)
			}
		})
	}
}

func TestValidateStruct_GTE(t *testing.T) {
	t.Parallel()
	type req struct {
		MaxExec *int `json:"max_executions" validate:"omitempty,gte=1"`
	}

	// nil pointer (omitempty): valid.
	err := ValidateStruct(&req{})
	if err != nil {
		t.Errorf("unexpected error for nil pointer: %v", err)
	}

	// Zero value: invalid.
	zero := 0
	err = ValidateStruct(&req{MaxExec: &zero})
	if err == nil {
		t.Fatal("expected error for gte=1 with 0")
	}
	if got := err.Error(); got != "max_executions must be at least 1" {
		t.Errorf("got %q, want %q", got, "max_executions must be at least 1")
	}

	// Valid.
	one := 1
	err = ValidateStruct(&req{MaxExec: &one})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateStruct_Oneof(t *testing.T) {
	t.Parallel()
	type req struct {
		Status *string `json:"status" validate:"omitempty,oneof=active disabled"`
	}

	// nil (omitempty): valid.
	err := ValidateStruct(&req{})
	if err != nil {
		t.Errorf("unexpected error for nil: %v", err)
	}

	// Valid value.
	active := "active"
	err = ValidateStruct(&req{Status: &active})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Invalid value.
	bad := "deleted"
	err = ValidateStruct(&req{Status: &bad})
	if err == nil {
		t.Fatal("expected error for invalid oneof")
	}
	if got := err.Error(); got != "status must be one of: 'active', 'disabled'" {
		t.Errorf("got %q, want %q", got, "status must be one of: 'active', 'disabled'")
	}
}

func TestValidateStruct_RequiredTime(t *testing.T) {
	t.Parallel()
	type req struct {
		ExpiresAt time.Time `json:"expires_at" validate:"required"`
	}

	// Zero time.
	err := ValidateStruct(&req{})
	if err == nil {
		t.Fatal("expected error for zero time")
	}
	if got := err.Error(); got != "expires_at is required" {
		t.Errorf("got %q, want %q", got, "expires_at is required")
	}

	// Valid.
	err = ValidateStruct(&req{ExpiresAt: time.Now()})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateStruct_RequiredRawMessage(t *testing.T) {
	t.Parallel()
	type req struct {
		Metadata json.RawMessage `json:"metadata" validate:"required"`
	}

	// nil RawMessage.
	err := ValidateStruct(&req{})
	if err == nil {
		t.Fatal("expected error for nil RawMessage")
	}
	if got := err.Error(); got != "metadata is required" {
		t.Errorf("got %q, want %q", got, "metadata is required")
	}

	// Non-nil RawMessage.
	err = ValidateStruct(&req{Metadata: json.RawMessage(`{"key": "value"}`)})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateStruct_MinMap(t *testing.T) {
	t.Parallel()
	type req struct {
		Creds map[string]any `json:"credentials" validate:"required,min=1"`
	}

	// nil map: required fails.
	err := ValidateStruct(&req{})
	if err == nil {
		t.Fatal("expected error for nil map")
	}
	if got := err.Error(); got != "credentials is required" {
		t.Errorf("got %q, want %q", got, "credentials is required")
	}

	// Empty map: min fails.
	err = ValidateStruct(&req{Creds: map[string]any{}})
	if err == nil {
		t.Fatal("expected error for empty map")
	}
	if got := err.Error(); got != "credentials is required and must be non-empty" {
		t.Errorf("got %q, want %q", got, "credentials is required and must be non-empty")
	}

	// Non-empty map: valid.
	err = ValidateStruct(&req{Creds: map[string]any{"key": "val"}})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateStruct_MultipleFieldsReportsFirst(t *testing.T) {
	t.Parallel()
	type req struct {
		Name string `json:"name" validate:"required"`
		ID   int64  `json:"id" validate:"gt=0"`
	}

	// Both invalid — validator should report the first field in struct order.
	err := ValidateStruct(&req{})
	if err == nil {
		t.Fatal("expected error")
	}
	// The first field alphabetically by struct order is Name.
	if got := err.Error(); got != "name is required" {
		t.Errorf("got %q, want %q", got, "name is required")
	}
}

func TestValidateStruct_LTEOnOptionalInt(t *testing.T) {
	t.Parallel()
	type req struct {
		TTL *int `json:"expires_in" validate:"omitempty,gte=60,lte=86400"`
	}

	// nil: valid.
	err := ValidateStruct(&req{})
	if err != nil {
		t.Errorf("unexpected error for nil: %v", err)
	}

	// Too small.
	small := 30
	err = ValidateStruct(&req{TTL: &small})
	if err == nil {
		t.Fatal("expected error for too small")
	}
	if got := err.Error(); got != "expires_in must be at least 60" {
		t.Errorf("got %q, want %q", got, "expires_in must be at least 60")
	}

	// Too large.
	big := 100000
	err = ValidateStruct(&req{TTL: &big})
	if err == nil {
		t.Fatal("expected error for too large")
	}
	if got := err.Error(); got != "expires_in must be at most 86400" {
		t.Errorf("got %q, want %q", got, "expires_in must be at most 86400")
	}

	// Just right.
	valid := 900
	err = ValidateStruct(&req{TTL: &valid})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
