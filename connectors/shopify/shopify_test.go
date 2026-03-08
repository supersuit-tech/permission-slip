package shopify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestShopifyConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "shopify" {
		t.Errorf("ID() = %q, want %q", got, "shopify")
	}
}

func TestShopifyConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	wantActions := []string{
		"shopify.get_orders",
		"shopify.get_order",
		"shopify.update_order",
		"shopify.create_product",
		"shopify.update_inventory",
		"shopify.create_discount",
		"shopify.fulfill_order",
		"shopify.cancel_order",
		"shopify.update_product",
		"shopify.create_collection",
		"shopify.get_analytics",
		"shopify.list_customers",
		"shopify.get_customer",
		"shopify.create_customer",
		"shopify.list_products",
		"shopify.get_product",
		"shopify.create_draft_order",
	}

	if len(actions) != len(wantActions) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(wantActions))
	}
	for _, actionType := range wantActions {
		if _, ok := actions[actionType]; !ok {
			t.Errorf("Actions() missing %q", actionType)
		}
	}
}

func TestShopifyConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid credentials",
			creds:   connectors.NewCredentials(map[string]string{"shop_domain": "mystore", "access_token": "shpat_abc123"}),
			wantErr: false,
		},
		{
			name:    "valid with full domain",
			creds:   connectors.NewCredentials(map[string]string{"shop_domain": "mystore.myshopify.com", "access_token": "shpat_abc123"}),
			wantErr: false,
		},
		{
			name:    "missing shop_domain",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "shpat_abc123"}),
			wantErr: true,
		},
		{
			name:    "empty shop_domain",
			creds:   connectors.NewCredentials(map[string]string{"shop_domain": "", "access_token": "shpat_abc123"}),
			wantErr: true,
		},
		{
			name:    "invalid shop_domain with custom domain",
			creds:   connectors.NewCredentials(map[string]string{"shop_domain": "shop.example.com", "access_token": "shpat_abc123"}),
			wantErr: true,
		},
		{
			name:    "missing access_token",
			creds:   connectors.NewCredentials(map[string]string{"shop_domain": "mystore"}),
			wantErr: true,
		},
		{
			name:    "empty access_token",
			creds:   connectors.NewCredentials(map[string]string{"shop_domain": "mystore", "access_token": ""}),
			wantErr: true,
		},
		{
			name:    "wrong key name",
			creds:   connectors.NewCredentials(map[string]string{"shop_domain": "mystore", "api_key": "shpat_abc123"}),
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

func TestShopifyConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "shopify" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "shopify")
	}
	if m.Name != "Shopify" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Shopify")
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "shopify" {
		t.Errorf("credential service = %q, want %q", cred.Service, "shopify")
	}
	if cred.AuthType != "api_key" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "api_key")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	if len(m.Actions) != 17 {
		t.Errorf("Manifest().Actions has %d items, want 17", len(m.Actions))
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v, want nil", err)
	}
}

func TestShopifyConnector_ActionsMatchManifest(t *testing.T) {
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

func TestShopifyConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*ShopifyConnector)(nil)
	var _ connectors.ManifestProvider = (*ShopifyConnector)(nil)
}

func TestShopifyConnector_Do_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header.
		if got := r.Header.Get("X-Shopify-Access-Token"); got != "shpat_test123" {
			t.Errorf("X-Shopify-Access-Token = %q, want %q", got, "shpat_test123")
		}
		// Verify no Authorization header (Shopify uses X-Shopify-Access-Token).
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization header should be empty, got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want %q", got, "application/json")
		}
		if r.Method != http.MethodGet {
			t.Errorf("Method = %q, want GET", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test.json", nil, &resp)
	if err != nil {
		t.Fatalf("do() error = %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("resp[status] = %q, want %q", resp["status"], "ok")
	}
}

func TestShopifyConnector_Do_PostWithBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Method = %q, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body["title"] != "Test Product" {
			t.Errorf("body[title] = %q, want %q", body["title"], "Test Product")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]int{"id": 123})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	reqBody := map[string]string{"title": "Test Product"}
	var resp map[string]int
	err := conn.do(t.Context(), validCreds(), http.MethodPost, "/products.json", reqBody, &resp)
	if err != nil {
		t.Fatalf("do() error = %v", err)
	}
	if resp["id"] != 123 {
		t.Errorf("resp[id] = %d, want 123", resp["id"])
	}
}

func TestShopifyConnector_Do_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"errors":"[API] Invalid API key or access token (unrecognized login or wrong password)"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/orders.json", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestShopifyConnector_Do_RateLimitError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"errors":"Exceeded 2 calls per second for api client."}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/orders.json", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 3*1e9 { // 3 seconds in nanoseconds
			t.Errorf("RetryAfter = %v, want 3s", rle.RetryAfter)
		}
	}
}

func TestShopifyConnector_Do_ValidationError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"errors":{"title":["can't be blank"]}}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodPost, "/products.json", map[string]any{}, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestShopifyConnector_Do_NotFoundError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":"Not Found"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/orders/999.json", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got %T: %v", err, err)
	}
}

func TestShopifyConnector_Do_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"errors":"Internal Server Error"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/orders.json", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestShopifyConnector_Do_MissingToken(t *testing.T) {
	t.Parallel()

	conn := newForTest(&http.Client{}, "http://localhost")
	creds := connectors.NewCredentials(map[string]string{"shop_domain": "mystore"})
	err := conn.do(t.Context(), creds, http.MethodGet, "/test.json", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestShopifyConnector_Do_CanceledContext(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately.

	err := conn.do(ctx, validCreds(), http.MethodGet, "/test.json", nil, nil)
	if err == nil {
		t.Fatal("do() expected error for canceled context, got nil")
	}
}

func TestShopBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		domain  string
		want    string
		wantErr bool
	}{
		{
			name:   "bare subdomain",
			domain: "mystore",
			want:   "https://mystore.myshopify.com/admin/api/2024-10",
		},
		{
			name:   "full domain",
			domain: "mystore.myshopify.com",
			want:   "https://mystore.myshopify.com/admin/api/2024-10",
		},
		{
			name:   "full domain with trailing slash",
			domain: "mystore.myshopify.com/",
			want:   "https://mystore.myshopify.com/admin/api/2024-10",
		},
		{
			name:   "subdomain with whitespace",
			domain: "  mystore  ",
			want:   "https://mystore.myshopify.com/admin/api/2024-10",
		},
		{
			name:   "subdomain with hyphen",
			domain: "my-store",
			want:   "https://my-store.myshopify.com/admin/api/2024-10",
		},
		{
			name:   "uppercase normalized to lowercase",
			domain: "MyStore",
			want:   "https://mystore.myshopify.com/admin/api/2024-10",
		},
		{
			name:    "custom domain rejected",
			domain:  "shop.example.com",
			wantErr: true,
		},
		{
			name:    "empty domain",
			domain:  "",
			wantErr: true,
		},
		{
			name:    "subdomain with path traversal",
			domain:  "evil/../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "subdomain with newline",
			domain:  "evil\nHost: attacker.com",
			wantErr: true,
		},
		{
			name:    "subdomain with special chars",
			domain:  "store@evil",
			wantErr: true,
		},
		{
			name:    "subdomain exceeding DNS label limit",
			domain:  "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmn",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := connectors.NewCredentials(map[string]string{
				"shop_domain":  tt.domain,
				"access_token": "shpat_test",
			})
			got, err := shopBaseURL(creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("shopBaseURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("shopBaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
