// Package ticketmaster implements the Ticketmaster Discovery API connector for
// the Permission Slip connector execution layer. It uses plain net/http with
// API key query-parameter authentication (no OAuth for read-only Discovery).
//
// Docs: https://developer.ticketmaster.com/products-and-docs/apis/discovery-api/v2/
package ticketmaster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://app.ticketmaster.com/discovery/v2"
	defaultTimeout = 30 * time.Second

	// maxResponseBytes caps Discovery API response size (10 MB).
	maxResponseBytes = 10 * 1024 * 1024

	// minInterval is the minimum spacing between requests per API key to stay
	// under Ticketmaster's documented 5 requests/second limit.
	minInterval = 210 * time.Millisecond

	// maxDailyRequests is Ticketmaster's documented free-tier daily cap.
	maxDailyRequests = 5000
)

// TicketmasterConnector owns the shared HTTP client and per-key rate limits.
type TicketmasterConnector struct {
	client  *http.Client
	baseURL string

	dayMu   sync.Mutex
	dayKey  string // UTC date YYYYMMDD for daily counter reset
	dayUsed map[string]int

	secondMu sync.Mutex
	// lastSecondCall maps api_key -> last request time (process-local).
	lastSecondCall map[string]time.Time
}

// New creates a TicketmasterConnector with production defaults.
func New() *TicketmasterConnector {
	return &TicketmasterConnector{
		client:         &http.Client{Timeout: defaultTimeout},
		baseURL:        defaultBaseURL,
		dayUsed:        make(map[string]int),
		lastSecondCall: make(map[string]time.Time),
	}
}

func newForTest(client *http.Client, baseURL string) *TicketmasterConnector {
	return &TicketmasterConnector{
		client:         client,
		baseURL:        baseURL,
		dayUsed:        make(map[string]int),
		lastSecondCall: make(map[string]time.Time),
	}
}

// ID returns "ticketmaster", matching connectors.id in the database.
func (c *TicketmasterConnector) ID() string { return "ticketmaster" }

// Actions returns registered action handlers keyed by action_type.
func (c *TicketmasterConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"ticketmaster.search_events":        &searchEventsAction{conn: c},
		"ticketmaster.get_event":            &getEventAction{conn: c},
		"ticketmaster.search_venues":        &searchVenuesAction{conn: c},
		"ticketmaster.get_venue":            &getVenueAction{conn: c},
		"ticketmaster.search_attractions":   &searchAttractionsAction{conn: c},
		"ticketmaster.get_attraction":       &getAttractionAction{conn: c},
		"ticketmaster.list_classifications": &listClassificationsAction{conn: c},
		"ticketmaster.suggest":              &suggestAction{conn: c},
	}
}

// ValidateCredentials ensures a non-empty API key is present.
func (c *TicketmasterConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	key, ok := creds.Get("api_key")
	if !ok || key == "" {
		return &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	return nil
}

func (c *TicketmasterConnector) apiKey(creds connectors.Credentials) (string, error) {
	key, _ := creds.Get("api_key")
	if key == "" {
		return "", &connectors.ValidationError{Message: "missing required credential: api_key"}
	}
	return key, nil
}

// throttlePerSecond waits until at least minInterval has passed since the last
// request for this API key (process-local best-effort enforcement).
func (c *TicketmasterConnector) throttlePerSecond(ctx context.Context, apiKey string) error {
	c.secondMu.Lock()
	defer c.secondMu.Unlock()

	last := c.lastSecondCall[apiKey]
	wait := time.Until(last.Add(minInterval))
	if wait > 0 {
		select {
		case <-ctx.Done():
			return &connectors.TimeoutError{Message: "Ticketmaster API request canceled"}
		case <-time.After(wait):
		}
	}
	c.lastSecondCall[apiKey] = time.Now()
	return nil
}

// checkDailyQuota returns RateLimitError when the process-local daily counter
// for this key reaches maxDailyRequests (resets at UTC midnight).
func (c *TicketmasterConnector) checkDailyQuota(apiKey string) error {
	today := time.Now().UTC().Format("20060102")

	c.dayMu.Lock()
	defer c.dayMu.Unlock()

	if c.dayKey != today {
		c.dayKey = today
		c.dayUsed = make(map[string]int)
	}

	if c.dayUsed[apiKey] >= maxDailyRequests {
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Ticketmaster Discovery API daily quota exceeded (%d requests/day per key; counter resets at UTC midnight on this server)", maxDailyRequests),
			RetryAfter: time.Until(nextUTCMidnight()),
		}
	}
	c.dayUsed[apiKey]++
	return nil
}

func nextUTCMidnight() time.Time {
	now := time.Now().UTC()
	y, m, d := now.Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, time.UTC)
}

// doGET performs a throttled GET to the Discovery API, appends apikey, and
// unmarshals JSON into respBody when non-nil.
func (c *TicketmasterConnector) doGET(ctx context.Context, creds connectors.Credentials, path string, query url.Values, respBody interface{}) error {
	apiKey, err := c.apiKey(creds)
	if err != nil {
		return err
	}

	if err := c.checkDailyQuota(apiKey); err != nil {
		return err
	}
	if err := c.throttlePerSecond(ctx, apiKey); err != nil {
		return err
	}

	if query == nil {
		query = url.Values{}
	}
	query.Set("apikey", apiKey)

	base := strings.TrimSuffix(c.baseURL, "/")
	rel := strings.TrimPrefix(path, "/")
	full := base + "/" + rel
	if strings.Contains(full, "\x00") {
		return &connectors.ValidationError{Message: "invalid request path"}
	}
	u, err := url.Parse(full)
	if err != nil {
		return fmt.Errorf("parsing request URL: %w", err)
	}
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Ticketmaster API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Ticketmaster API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Ticketmaster API request failed: %v", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, body); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(body, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing Ticketmaster response: %v", err)}
		}
	}
	return nil
}

func parseParams(data json.RawMessage, dest interface{}) error {
	if err := json.Unmarshal(data, dest); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	return nil
}

// validateTicketmasterID ensures path segments are safe (alphanumeric + TM ids).
func validateTicketmasterID(id, field string) error {
	if id == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", field)}
	}
	if len(id) > 128 {
		return &connectors.ValidationError{Message: fmt.Sprintf("%s exceeds maximum length of 128 characters", field)}
	}
	for _, ch := range id {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		case ch == '_', ch == '-', ch == '.':
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: only letters, digits, hyphen, underscore, and period are allowed", field)}
		}
	}
	return nil
}

func appendNonEmpty(q url.Values, apiName, val string) {
	if val != "" {
		q.Set(apiName, val)
	}
}

func trimString(s string) string { return strings.TrimSpace(s) }
