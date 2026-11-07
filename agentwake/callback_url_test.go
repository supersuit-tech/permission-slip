package agentwake

import (
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestValidateCallbackURL_OpenClawPrivate(t *testing.T) {
	if err := connectors.ValidatePrivateNetworkURL("http://127.0.0.1:1/hooks", "callback_url"); err != nil {
		t.Fatal(err)
	}
	provider, err := ValidateCallbackURL("http://127.0.0.1:18789/hooks")
	if err != nil {
		t.Fatal(err)
	}
	if provider != ProviderOpenClaw {
		t.Fatalf("provider = %q", provider)
	}
}

func TestValidateCallbackURL_RejectsCredentialsInURL(t *testing.T) {
	_, err := ValidateCallbackURL("http://user:pass@127.0.0.1/hooks")
	if err == nil || !strings.Contains(err.Error(), "credentials") {
		t.Fatalf("err = %v", err)
	}
}
