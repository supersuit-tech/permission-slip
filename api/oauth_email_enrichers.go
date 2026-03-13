// OAuth email enrichers fetch the authenticated user's email from provider
// userinfo endpoints after a successful token exchange. Unlike hard enrichers
// (postOAuthEnrichers), these are best-effort: a failure to fetch the email
// does not block storing the OAuth connection.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// softOAuthEnrichers maps provider IDs to best-effort enrichers that fetch
// the authenticated user's email after OAuth token exchange. Failures are
// logged but do not block the connection from being stored.
//
// Add a new entry here when a provider exposes a userinfo or profile endpoint
// that includes the user's email address.
var softOAuthEnrichers = map[string]postOAuthEnricher{
	"google":    fetchGoogleEmail,
	"github":    fetchGitHubEmail,
	"microsoft": fetchMicrosoftEmail,
	"slack":     fetchSlackEmail,
}

// emailHTTPClient is a shared HTTP client for email enrichers.
var emailHTTPClient = &http.Client{Timeout: 10 * time.Second}

// fetchGoogleEmail fetches the user's email from Google's userinfo endpoint.
func fetchGoogleEmail(ctx context.Context, accessToken string) (map[string]string, error) {
	return fetchEmailFromJSON(ctx, accessToken, "https://www.googleapis.com/oauth2/v3/userinfo", "email")
}

// fetchGitHubEmail fetches the user's email from GitHub's user endpoint.
// Falls back to the /user/emails endpoint for users with private emails.
func fetchGitHubEmail(ctx context.Context, accessToken string) (map[string]string, error) {
	extra, err := fetchEmailFromJSON(ctx, accessToken, "https://api.github.com/user", "email")
	if err == nil && extra["email"] != "" {
		return extra, nil
	}

	// GitHub users can have a private email; fall back to /user/emails.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, fmt.Errorf("create github emails request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := emailHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github emails request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github emails returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("read github emails response: %w", err)
	}

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.Unmarshal(body, &emails); err != nil {
		return nil, fmt.Errorf("parse github emails: %w", err)
	}

	for _, e := range emails {
		if e.Primary && e.Email != "" {
			return map[string]string{"email": e.Email}, nil
		}
	}
	if len(emails) > 0 && emails[0].Email != "" {
		return map[string]string{"email": emails[0].Email}, nil
	}

	return nil, fmt.Errorf("no email found in GitHub response")
}

// fetchMicrosoftEmail fetches the user's email from Microsoft Graph.
// Tries the "mail" field first, falling back to "userPrincipalName".
func fetchMicrosoftEmail(ctx context.Context, accessToken string) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := emailHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if mail, _ := data["mail"].(string); mail != "" {
		return map[string]string{"email": mail}, nil
	}
	if upn, _ := data["userPrincipalName"].(string); upn != "" {
		return map[string]string{"email": upn}, nil
	}
	return nil, fmt.Errorf("no email found in Microsoft profile")
}

// fetchSlackEmail fetches the user's email from Slack's OIDC userinfo endpoint.
func fetchSlackEmail(ctx context.Context, accessToken string) (map[string]string, error) {
	return fetchEmailFromJSON(ctx, accessToken, "https://slack.com/api/openid.connect.userInfo", "email")
}

// fetchEmailFromJSON is a generic helper that calls a JSON endpoint with the
// access token and extracts a string field into {"email": value}.
func fetchEmailFromJSON(ctx context.Context, accessToken, url, field string) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := emailHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	email, _ := data[field].(string)
	if email == "" {
		return nil, fmt.Errorf("field %q not found or empty", field)
	}

	return map[string]string{"email": email}, nil
}

// runSoftEnrichers runs the best-effort email enrichers for a provider.
// Returns extra data to merge into the connection's extra_data, or nil
// if no enricher exists or the enricher fails.
func runSoftEnrichers(ctx context.Context, providerID, accessToken string) map[string]string {
	enricher, ok := softOAuthEnrichers[providerID]
	if !ok {
		return nil
	}
	extra, err := enricher(ctx, accessToken)
	if err != nil {
		log.Printf("oauth: soft enricher for %q failed (non-fatal): %v", providerID, err)
		return nil
	}
	return extra
}
