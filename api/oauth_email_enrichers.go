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

// fetchGoogleEmail fetches the user's email and name from Google's userinfo endpoint.
func fetchGoogleEmail(ctx context.Context, accessToken string) (map[string]string, error) {
	return fetchProfileFromJSON(ctx, accessToken, "https://www.googleapis.com/oauth2/v3/userinfo", "email", "name")
}

// fetchGitHubEmail fetches the user's email and username from GitHub's user
// endpoint. Falls back to the /user/emails endpoint for users with private
// emails. The "display_name" key in the returned map contains the GitHub login.
func fetchGitHubEmail(ctx context.Context, accessToken string) (map[string]string, error) {
	// First, fetch the /user endpoint which has both login and email.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("create github user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := emailHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github user request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github user returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("read github user response: %w", err)
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse github user: %w", err)
	}

	extra := make(map[string]string)
	if login, _ := data["login"].(string); login != "" {
		extra["display_name"] = login
	}
	if email, _ := data["email"].(string); email != "" {
		extra["email"] = email
		return extra, nil
	}

	// GitHub users can have a private email; fall back to /user/emails.
	fallbackExtra, fallbackErr := fetchGitHubEmailFallback(ctx, accessToken)
	if fallbackErr != nil {
		if len(extra) > 0 {
			log.Printf("oauth: github email fallback failed (non-fatal, have login): %v", fallbackErr)
			return extra, nil
		}
		return nil, fallbackErr
	}

	for k, v := range fallbackExtra {
		extra[k] = v
	}
	return extra, nil
}

// fetchGitHubEmailFallback fetches the primary email from GitHub's /user/emails
// endpoint. Used when the /user endpoint doesn't include an email.
func fetchGitHubEmailFallback(ctx context.Context, accessToken string) (map[string]string, error) {
	emailReq, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return nil, fmt.Errorf("create github emails request: %w", err)
	}
	emailReq.Header.Set("Authorization", "Bearer "+accessToken)
	emailReq.Header.Set("Accept", "application/vnd.github+json")

	emailResp, err := emailHTTPClient.Do(emailReq)
	if err != nil {
		return nil, fmt.Errorf("github emails request failed: %w", err)
	}
	defer emailResp.Body.Close()

	if emailResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github emails returned HTTP %d", emailResp.StatusCode)
	}

	emailBody, err := io.ReadAll(io.LimitReader(emailResp.Body, 64*1024))
	if err != nil {
		return nil, fmt.Errorf("read github emails response: %w", err)
	}

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
	}
	if err := json.Unmarshal(emailBody, &emails); err != nil {
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

	return nil, fmt.Errorf("no email found in GitHub /user/emails response")
}

// fetchMicrosoftEmail fetches the user's email and display name from Microsoft
// Graph. Stores displayName as "display_name", tries "mail" first for email,
// falling back to "userPrincipalName".
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

	extra := make(map[string]string)
	if name, _ := data["displayName"].(string); name != "" {
		extra["display_name"] = name
	}
	if mail, _ := data["mail"].(string); mail != "" {
		extra["email"] = mail
		return extra, nil
	}
	if upn, _ := data["userPrincipalName"].(string); upn != "" {
		extra["email"] = upn
		return extra, nil
	}
	if len(extra) > 0 {
		return extra, nil
	}
	return nil, fmt.Errorf("no email found in Microsoft profile")
}

// fetchSlackEmail fetches the user's email and name from Slack's OIDC userinfo endpoint.
func fetchSlackEmail(ctx context.Context, accessToken string) (map[string]string, error) {
	return fetchProfileFromJSON(ctx, accessToken, "https://slack.com/api/openid.connect.userInfo", "email", "name")
}

// fetchProfileFromJSON is a generic helper that calls a JSON endpoint with the
// access token and extracts an email and optional display name field. The name
// is stored under the "display_name" key in the returned map.
func fetchProfileFromJSON(ctx context.Context, accessToken, url, emailField, nameField string) (map[string]string, error) {
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

	extra := make(map[string]string)
	if name, _ := data[nameField].(string); name != "" {
		extra["display_name"] = name
	}
	if email, _ := data[emailField].(string); email != "" {
		extra["email"] = email
	}

	if len(extra) == 0 {
		return nil, fmt.Errorf("neither field %q nor %q found in response", emailField, nameField)
	}
	return extra, nil
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
