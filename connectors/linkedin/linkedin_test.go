package linkedin

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "linkedin" {
		t.Errorf("expected ID 'linkedin', got %q", c.ID())
	}
}

func TestActions_AllRegistered(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"linkedin.create_post",
		"linkedin.delete_post",
		"linkedin.add_comment",
		"linkedin.get_profile",
		"linkedin.get_post_analytics",
		"linkedin.create_company_post",
		"linkedin.send_message",
		"linkedin.search_people",
		"linkedin.search_companies",
		"linkedin.get_company",
		"linkedin.list_connections",
	}

	for _, name := range expected {
		if _, ok := actions[name]; !ok {
			t.Errorf("expected action %q to be registered", name)
		}
	}

	if len(actions) != len(expected) {
		t.Errorf("expected %d actions, got %d", len(expected), len(actions))
	}
}

func TestValidateCredentials_Valid(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(context.Background(), validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCredentials_MissingToken(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(context.Background(), connectors.NewCredentials(map[string]string{}))
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestValidateCredentials_EmptyToken(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(context.Background(), connectors.NewCredentials(map[string]string{
		"access_token": "",
	}))
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestManifest_Valid(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()
	if err := m.Validate(); err != nil {
		t.Fatalf("manifest validation failed: %v", err)
	}
}
