package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
	"github.com/supersuit-tech/permission-slip-web/oauth"
	"github.com/supersuit-tech/permission-slip-web/vault"
)

type stubCalendarConnector struct {
	id       string
	listFunc func(context.Context, connectors.Credentials) ([]connectors.CalendarListItem, error)
}

func (s *stubCalendarConnector) ID() string { return s.id }

func (s *stubCalendarConnector) Actions() map[string]connectors.Action {
	return nil
}

func (s *stubCalendarConnector) ValidateCredentials(_ context.Context, _ connectors.Credentials) error {
	return nil
}

func (s *stubCalendarConnector) ListCalendars(ctx context.Context, creds connectors.Credentials) ([]connectors.CalendarListItem, error) {
	if s.listFunc != nil {
		return s.listFunc(ctx, creds)
	}
	return []connectors.CalendarListItem{
		{ID: "cal-1", Summary: "Work", Primary: true},
	}, nil
}

func (s *stubCalendarConnector) CalendarListCredentialActionType() string {
	return s.id + ".list_calendar_events"
}

func decodeCalendarList(t *testing.T, body []byte) calendarListResponse {
	t.Helper()
	var resp calendarListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal calendar list: %v", err)
	}
	return resp
}

func TestListAgentConnectorCalendars_NoCredentialBinding(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

	connID := "calstub"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".list_calendar_events", "List events")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, connID, connID, nil)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	reg := connectors.NewRegistry()
	reg.Register(&stubCalendarConnector{id: connID})

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
		DB:             tx,
		Vault:          v,
		Connectors:     reg,
		OAuthProviders: oauthReg,
		SupabaseJWTSecret: testJWTSecret,
	}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/connectors/%s/calendars", agentID, connID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListAgentConnectorCalendars_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

	connID := "calstub"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".list_calendar_events", "List events")
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
	reg.Register(&stubCalendarConnector{
		id: connID,
		listFunc: func(_ context.Context, creds connectors.Credentials) ([]connectors.CalendarListItem, error) {
			tok, _ := creds.Get("access_token")
			if tok == "" {
				t.Error("expected non-empty access_token in credentials")
			}
			return []connectors.CalendarListItem{
				{ID: "primary", Summary: "Primary", Primary: true},
				{ID: "work@group.calendar.google.com", Summary: "Team"},
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
		DB:             tx,
		Vault:          v,
		Connectors:     reg,
		OAuthProviders: oauthReg,
		SupabaseJWTSecret: testJWTSecret,
	}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, fmt.Sprintf("/agents/%d/connectors/%s/calendars", agentID, connID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	resp := decodeCalendarList(t, w.Body.Bytes())
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 calendars, got %d", len(resp.Data))
	}
	if resp.Data[0].ID != "primary" || !resp.Data[0].Primary {
		t.Errorf("unexpected first item: %+v", resp.Data[0])
	}
}
