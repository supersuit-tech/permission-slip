package slack

import (
	"encoding/json"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// validCreds returns a Credentials value with a valid user OAuth token for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "xoxp-test-token-123",
	})
}

// testFullSlackScopes is a comma-separated list of user-token scopes that
// satisfies every scope check in this package. Tests that don't care about
// scope validation pass this to writeAuthTestResponse so the scope probe
// (see #1033) doesn't false-positive and block their code path.
const testFullSlackScopes = "channels:read,channels:history,groups:read,groups:history,im:read,im:history,mpim:read,mpim:history,users:read,users:read.email"

// writeAuthTestResponse writes a minimal auth.test response with the given
// comma-separated X-OAuth-Scopes header. Tests mock /auth.test with this
// helper so the scope probe added for #1033 sees the expected granted
// scopes and proceeds to the code path under test.
func writeAuthTestResponse(w http.ResponseWriter, scopes string) {
	if scopes != "" {
		w.Header().Set("X-OAuth-Scopes", scopes)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"url":     "https://example.slack.com/",
		"team":    "Example",
		"user":    "caller",
		"team_id": "T_TEST",
		"user_id": "U_CALLER",
	})
}
