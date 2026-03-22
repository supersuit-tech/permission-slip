package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	googleconnector "github.com/supersuit-tech/permission-slip-web/connectors/google"
	"github.com/supersuit-tech/permission-slip-web/connectors/microsoft"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
	"github.com/supersuit-tech/permission-slip-web/vault"
)

func TestListAgentConnectorCalendars_Google(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/calendar/v3/users/me/calendarList" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"id":"primary","summary":"Primary","primary":true},{"id":"cal_other","summary":"Work"}]}`))
	}))
	t.Cleanup(srv.Close)

	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := "google"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	mockVault := vault.NewMockVaultStore()
	oauthID := testhelper.GenerateID(t, "oac_")
	accessVaultID, err := mockVault.CreateSecret(context.Background(), tx, "tok", []byte("test-access-token"))
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	testhelper.InsertOAuthConnectionFull(t, tx, oauthID, uid, "google", testhelper.OAuthConnectionOpts{
		AccessTokenVaultID: accessVaultID,
		Scopes:             []string{"https://www.googleapis.com/auth/calendar.events"},
	})

	bindingID := testhelper.GenerateID(t, "accr_")
	_, bindErr := db.UpsertAgentConnectorCredential(t.Context(), tx, db.UpsertAgentConnectorCredentialParams{
		ID:                 bindingID,
		AgentID:            agentID,
		ConnectorID:        connID,
		ApproverID:         uid,
		OAuthConnectionID:  &oauthID,
	})
	if bindErr != nil {
		t.Fatalf("upsert binding: %v", bindErr)
	}

	gc := googleconnector.NewCalendarForTest(srv.Client(), srv.URL+"/calendar/v3")

	reg := connectors.NewRegistry()
	reg.Register(gc)

	deps := &Deps{DB: tx, Vault: mockVault, Connectors: reg, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet,
		fmt.Sprintf("/agents/%d/connectors/google/calendars", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentConnectorCalendarsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 calendars, got %d", len(resp.Data))
	}
	if resp.Data[0].ID != "primary" || !resp.Data[0].IsPrimary {
		t.Errorf("first calendar: %+v", resp.Data[0])
	}
}

func TestListAgentConnectorCalendars_Microsoft(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1.0/me/calendars" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"value":[{"id":"cal-1","name":"Calendar","isDefaultCalendar":true}]}`))
	}))
	t.Cleanup(srv.Close)

	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])

	connID := "microsoft"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	mockVault := vault.NewMockVaultStore()
	oauthID := testhelper.GenerateID(t, "oac_")
	accessVaultID, err := mockVault.CreateSecret(context.Background(), tx, "tok", []byte("ms-token"))
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	testhelper.InsertOAuthConnectionFull(t, tx, oauthID, uid, "microsoft", testhelper.OAuthConnectionOpts{
		AccessTokenVaultID: accessVaultID,
		Scopes:             []string{"Calendars.ReadWrite"},
	})

	bindingID := testhelper.GenerateID(t, "accr_")
	_, bindErr := db.UpsertAgentConnectorCredential(t.Context(), tx, db.UpsertAgentConnectorCredentialParams{
		ID:                bindingID,
		AgentID:           agentID,
		ConnectorID:       connID,
		ApproverID:        uid,
		OAuthConnectionID: &oauthID,
	})
	if bindErr != nil {
		t.Fatalf("upsert binding: %v", bindErr)
	}

	mc := microsoft.NewForTest(srv.Client(), srv.URL+"/v1.0")
	reg := connectors.NewRegistry()
	reg.Register(mc)

	deps := &Deps{DB: tx, Vault: mockVault, Connectors: reg, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet,
		fmt.Sprintf("/agents/%d/connectors/microsoft/calendars", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp agentConnectorCalendarsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].ID != "cal-1" || !resp.Data[0].IsPrimary {
		t.Errorf("unexpected data: %+v", resp.Data)
	}
}

func TestListAgentConnectorCalendars_NoCredential(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	agentID := testhelper.InsertUserWithAgent(t, tx, uid, "u_"+uid[:8])
	testhelper.InsertConnector(t, tx, "google")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, "google")

	reg := connectors.NewRegistry()
	reg.Register(googleconnector.New())

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), Connectors: reg, SupabaseJWTSecret: testJWTSecret}
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet,
		fmt.Sprintf("/agents/%d/connectors/google/calendars", agentID), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
