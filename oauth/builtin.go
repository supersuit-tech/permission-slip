package oauth

import (
	"fmt"
	"os"
	"strings"

	slackconnector "github.com/supersuit-tech/permission-slip-web/connectors/slack"
)

// BuiltInProviders returns the platform's pre-configured OAuth providers.
// Client credentials are read from environment variables; if not set, the
// providers are still registered (so manifest validation passes) but cannot
// initiate OAuth flows until BYOA credentials are supplied.
func BuiltInProviders() []Provider {
	return []Provider{
		{
			ID:           "google",
			AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			Scopes: []string{
				"openid",
				"https://www.googleapis.com/auth/userinfo.email",
			},
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "meta",
			AuthorizeURL: "https://www.facebook.com/v19.0/dialog/oauth",
			TokenURL:     "https://graph.facebook.com/v19.0/oauth/access_token",
			Scopes: []string{
				"pages_manage_posts",
				"pages_read_engagement",
				"pages_read_user_content",
				"instagram_basic",
				"instagram_content_publish",
				"instagram_manage_insights",
			},
			ClientID:     os.Getenv("META_CLIENT_ID"),
			ClientSecret: os.Getenv("META_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "microsoft",
			AuthorizeURL: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			Scopes: []string{
				"openid",
				"offline_access",
				"User.Read",
			},
			ClientID:     os.Getenv("MICROSOFT_CLIENT_ID"),
			ClientSecret: os.Getenv("MICROSOFT_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "linkedin",
			AuthorizeURL: "https://www.linkedin.com/oauth/v2/authorization",
			TokenURL:     "https://www.linkedin.com/oauth/v2/accessToken",
			Scopes: []string{
				"openid",
				"profile",
				"w_member_social",
				"r_organization_social",
				"w_organization_social",
			},
			ClientID:     os.Getenv("LINKEDIN_CLIENT_ID"),
			ClientSecret: os.Getenv("LINKEDIN_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "salesforce",
			AuthorizeURL: "https://login.salesforce.com/services/oauth2/authorize",
			TokenURL:     "https://login.salesforce.com/services/oauth2/token",
			Scopes: []string{
				"api",
				"refresh_token",
			},
			ClientID:     os.Getenv("SALESFORCE_CLIENT_ID"),
			ClientSecret: os.Getenv("SALESFORCE_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "zoom",
			AuthorizeURL: "https://zoom.us/oauth/authorize",
			TokenURL:     "https://zoom.us/oauth/token",
			Scopes: []string{
				"meeting:read",
				"meeting:write",
				"recording:read",
				"user:read",
			},
			ClientID:     os.Getenv("ZOOM_CLIENT_ID"),
			ClientSecret: os.Getenv("ZOOM_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "figma",
			AuthorizeURL: "https://www.figma.com/oauth",
			TokenURL:     "https://api.figma.com/v1/oauth/token",
			Scopes: []string{
				"files:read",
				"file_comments:write",
			},
			ClientID:     os.Getenv("FIGMA_CLIENT_ID"),
			ClientSecret: os.Getenv("FIGMA_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "github",
			AuthorizeURL: "https://github.com/login/oauth/authorize",
			TokenURL:     "https://github.com/login/oauth/access_token",
			Scopes: []string{
				"repo",
			},
			ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
			ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "hubspot",
			AuthorizeURL: "https://app.hubspot.com/oauth/authorize",
			TokenURL:     "https://api.hubapi.com/oauth/v1/token",
			Scopes: []string{
				"crm.objects.contacts.read",
				"crm.objects.contacts.write",
				"crm.objects.deals.read",
				"crm.objects.deals.write",
				"crm.objects.companies.read",
				"tickets",
				"automation",
				"content",
				"analytics.read",
			},
			ClientID:     os.Getenv("HUBSPOT_CLIENT_ID"),
			ClientSecret: os.Getenv("HUBSPOT_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "kroger",
			AuthorizeURL: "https://api.kroger.com/v1/connect/oauth2/authorize",
			TokenURL:     "https://api.kroger.com/v1/connect/oauth2/token",
			Scopes: []string{
				"product.compact",
				"cart.basic:write",
			},
			ClientID:     os.Getenv("KROGER_CLIENT_ID"),
			ClientSecret: os.Getenv("KROGER_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "linear",
			AuthorizeURL: "https://linear.app/oauth/authorize",
			TokenURL:     "https://api.linear.app/oauth/token",
			Scopes: []string{
				"read",
				"write",
			},
			ClientID:     os.Getenv("LINEAR_CLIENT_ID"),
			ClientSecret: os.Getenv("LINEAR_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "netlify",
			AuthorizeURL: "https://app.netlify.com/authorize",
			TokenURL:     "https://api.netlify.com/oauth/token",
			ClientID:     os.Getenv("NETLIFY_CLIENT_ID"),
			ClientSecret: os.Getenv("NETLIFY_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "notion",
			AuthorizeURL: "https://api.notion.com/v1/oauth/authorize",
			TokenURL:     "https://api.notion.com/v1/oauth/token",
			Scopes:       []string{},
			ClientID:     os.Getenv("NOTION_CLIENT_ID"),
			ClientSecret: os.Getenv("NOTION_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "pagerduty",
			AuthorizeURL: "https://identity.pagerduty.com/oauth/authorize",
			TokenURL:     "https://identity.pagerduty.com/oauth/token",
			Scopes: []string{
				"read",
				"write",
			},
			ClientID:     os.Getenv("PAGERDUTY_CLIENT_ID"),
			ClientSecret: os.Getenv("PAGERDUTY_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			// Shopify uses per-shop OAuth URLs. The {shop} placeholder is
			// replaced at authorize/callback time with the user's shop
			// subdomain (e.g. "mystore"). See api/oauth.go for resolution.
			ID:           "shopify",
			AuthorizeURL: "https://{shop}.myshopify.com/admin/oauth/authorize",
			TokenURL:     "https://{shop}.myshopify.com/admin/oauth/access_token",
			Scopes: []string{
				"write_orders",
				"write_products",
				"write_inventory",
				"write_discounts",
				"read_reports",
				"read_all_orders",
			},
			ClientID:     os.Getenv("SHOPIFY_CLIENT_ID"),
			ClientSecret: os.Getenv("SHOPIFY_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "slack",
			AuthorizeURL: "https://slack.com/oauth/v2/authorize",
			TokenURL:     "https://slack.com/api/oauth.v2.access",
			Scopes:       slackconnector.OAuthScopes,
			// Slack V2 OAuth requires comma-separated scopes instead of the
			// standard space-separated format.
			AuthorizeParams: map[string]string{
				"scope": strings.Join(slackconnector.OAuthScopes, ","),
			},
			ClientID:     os.Getenv("SLACK_CLIENT_ID"),
			ClientSecret: os.Getenv("SLACK_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "square",
			AuthorizeURL: "https://connect.squareup.com/oauth2/authorize",
			TokenURL:     "https://connect.squareup.com/oauth2/token",
			Scopes: []string{
				"ORDERS_READ",
				"ORDERS_WRITE",
				"PAYMENTS_READ",
				"PAYMENTS_WRITE",
				"ITEMS_READ",
				"ITEMS_WRITE",
				"CUSTOMERS_READ",
				"CUSTOMERS_WRITE",
				"APPOINTMENTS_READ",
				"APPOINTMENTS_WRITE",
				"INVOICES_READ",
				"INVOICES_WRITE",
				"INVENTORY_READ",
				"INVENTORY_WRITE",
			},
			ClientID:     os.Getenv("SQUARE_CLIENT_ID"),
			ClientSecret: os.Getenv("SQUARE_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "stripe",
			AuthorizeURL: "https://connect.stripe.com/oauth/authorize",
			TokenURL:     "https://connect.stripe.com/oauth/token",
			Scopes: []string{
				"read_write",
			},
			ClientID:     os.Getenv("STRIPE_CLIENT_ID"),
			ClientSecret: os.Getenv("STRIPE_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			ID:           "calendly",
			AuthorizeURL: "https://auth.calendly.com/oauth/authorize",
			TokenURL:     "https://auth.calendly.com/oauth/token",
			Scopes:       []string{},
			ClientID:     os.Getenv("CALENDLY_CLIENT_ID"),
			ClientSecret: os.Getenv("CALENDLY_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
		{
			// Zendesk uses per-subdomain OAuth URLs. The {subdomain} placeholder
			// is replaced at authorize/callback time with the user's Zendesk
			// subdomain (e.g. "mycompany"). See api/oauth.go for resolution.
			ID:           "zendesk",
			AuthorizeURL: "https://{subdomain}.zendesk.com/oauth/authorizations/new",
			TokenURL:     "https://{subdomain}.zendesk.com/oauth/tokens",
			Scopes: []string{
				"read",
				"write",
			},
			ClientID:     os.Getenv("ZENDESK_CLIENT_ID"),
			ClientSecret: os.Getenv("ZENDESK_CLIENT_SECRET"),
			Source:       SourceBuiltIn,
		},
	}
}

// NewRegistryWithBuiltIns creates a new provider registry pre-populated with
// the platform's built-in providers. Panics if a built-in provider has an
// invalid ID (programming error).
func NewRegistryWithBuiltIns() *Registry {
	r := NewRegistry()
	for _, p := range BuiltInProviders() {
		if err := r.Register(p); err != nil {
			panic(fmt.Sprintf("failed to register built-in OAuth provider %q: %v", p.ID, err))
		}
	}
	return r
}

// RegisterFromManifest registers providers declared in a connector manifest's
// oauth_providers section. These are external providers that the platform
// doesn't have built-in support for. They are registered without client
// credentials — users must supply those via BYOA.
// Returns an error if any provider fails validation.
func RegisterFromManifest(r *Registry, providers []ManifestProvider) error {
	for _, mp := range providers {
		if err := r.Register(Provider{
			ID:              mp.ID,
			AuthorizeURL:    mp.AuthorizeURL,
			TokenURL:        mp.TokenURL,
			Scopes:          mp.Scopes,
			AuthorizeParams: mp.AuthorizeParams,
			Source:          SourceManifest,
		}); err != nil {
			return fmt.Errorf("registering manifest OAuth provider %q: %w", mp.ID, err)
		}
	}
	return nil
}

// ManifestProvider mirrors the OAuth provider declaration in a connector
// manifest. This avoids a circular import between oauth/ and connectors/.
type ManifestProvider struct {
	ID              string
	AuthorizeURL    string
	TokenURL        string
	Scopes          []string
	AuthorizeParams map[string]string
}
