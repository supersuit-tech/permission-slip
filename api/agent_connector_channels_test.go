package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/oauth"
	"github.com/supersuit-tech/permission-slip/vault"
)

type stubChannelConnector struct {
	id       string
	listFunc func(context.Context, connectors.Credentials, string) ([]connectors.ChannelListItem, error)
}

func (s *stubChannelConnector) ID() string { return s.id }

func (s *stubChannelConnector) Actions() map[string]connectors.Action {
	return nil
}

func (s *stubChannelConnector) ValidateCredentials(_ context.Context, _ connectors.Credentials) error {
	return nil
}

func (s *stubChannelConnector) ListChannels(ctx context.Context, creds connectors.Credentials, userEmail string) ([]connectors.ChannelListItem, error) {
	if s.listFunc != nil {
		return s.listFunc(ctx, creds, userEmail)
	}
	return []connectors.ChannelListItem{
		{ID: "C1", DisplayLabel: "#general", NumMembers: 10},
	}, nil
}

func (s *stubChannelConnector) ChannelListCredentialActionType() string {
	return s.id + ".list_channels"
}

func decodeChannelList(t *testing.T, body []byte) channelListResponse {
	t.Helper()
	var resp channelListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal channel list: %v", err)
	}
	return resp
}

func TestListAgentConnectorChannels_NoCredentialBinding(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

	connID := "chstub"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".list_channels", "List channels")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, connID, connID, nil)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	reg := connectors.NewRegistry()
	reg.Register(&stubChannelConnector{id: connID})

	v := vault.NewMockVaultStore()
	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID:           connID,
		AuthorizeURL: "https://example.com/oauth/authorize",
		TokenURL:     "https://example.com/oauth/token",
		ClientID:     "test",
		ClientSecret: "test",
		Source:       oauth.SourceBuiltIn,
	})
	deps := &Deps{
		DB:                tx,
		Vault:             v,
		Connectors:        reg,
		OAuthProviders:    oauthReg,
		SupabaseJWTSecret: testJWTSecret,
	}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/connectors/%s/channels", agentID, connID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListAgentConnectorChannels_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

	connID := "chstub"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".list_channels", "List channels")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, connID, connID, nil)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	oauthConnID := testhelper.GenerateID(t, "oconn_")
	testhelper.InsertOAuthConnection(t, tx, oauthConnID, uid, connID)
	oauthRow, err := db.GetOAuthConnectionByProvider(t.Context(), tx, uid, connID)
	if err != nil {
		t.Fatalf("GetOAuthConnectionByProvider: %v", err)
	}
	bindingID := testhelper.GenerateID(t, "accr_")
	if _, err := db.UpsertAgentConnectorCredential(t.Context(), tx, db.UpsertAgentConnectorCredentialParams{
		ID: bindingID, AgentID: agentID, ConnectorID: connID,
		ApproverID: uid, OAuthConnectionID: &oauthRow.ID,
	}); err != nil {
		t.Fatalf("UpsertAgentConnectorCredential: %v", err)
	}

	reg := connectors.NewRegistry()
	reg.Register(&stubChannelConnector{
		id: connID,
		listFunc: func(_ context.Context, creds connectors.Credentials, email string) ([]connectors.ChannelListItem, error) {
			tok, _ := creds.Get("access_token")
			if tok == "" {
				t.Error("expected non-empty access_token in credentials")
			}
			if email != "test@example.com" {
				t.Errorf("expected user email test@example.com, got %q", email)
			}
			return []connectors.ChannelListItem{
				{ID: "C111", DisplayLabel: "#announcements", IsPrivate: false, NumMembers: 5},
				{ID: "G222", DisplayLabel: "private-team", IsPrivate: true},
			}, nil
		},
	})

	v := vault.NewMockVaultStore()
	v.SeedSecretForTest(oauthRow.AccessTokenVaultID, []byte(`{"access_token":"tok-test"}`))

	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID:           connID,
		AuthorizeURL: "https://example.com/oauth/authorize",
		TokenURL:     "https://example.com/oauth/token",
		ClientID:     "test",
		ClientSecret: "test",
		Source:       oauth.SourceBuiltIn,
	})
	deps := &Deps{
		DB:                tx,
		Vault:             v,
		Connectors:        reg,
		OAuthProviders:    oauthReg,
		SupabaseJWTSecret: testJWTSecret,
	}
	router := NewRouter(deps)

	r := authenticatedRequestWithEmail(t, http.MethodGet, fmt.Sprintf("/agents/%d/connectors/%s/channels", agentID, connID), uid, "test@example.com")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeChannelList(t, w.Body.Bytes())
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 channels, got %d", len(resp.Data))
	}
	if resp.Data[0].ID != "C111" || resp.Data[0].DisplayLabel != "#announcements" {
		t.Errorf("unexpected first item: %+v", resp.Data[0])
	}
}
