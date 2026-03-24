package ticketmaster

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{"api_key": "test_key"})
}

func TestTicketmasterConnector_ID(t *testing.T) {
	t.Parallel()
	if got := New().ID(); got != "ticketmaster" {
		t.Errorf("ID() = %q, want ticketmaster", got)
	}
}

func TestTicketmasterConnector_Actions(t *testing.T) {
	t.Parallel()
	n := len(New().Actions())
	if n != 8 {
		t.Errorf("len(Actions()) = %d, want 8", n)
	}
}

func TestTicketmasterConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(t.Context(), connectors.NewCredentials(map[string]string{"api_key": "x"}))
	if err != nil {
		t.Errorf("ValidateCredentials valid: %v", err)
	}
	err = c.ValidateCredentials(t.Context(), connectors.NewCredentials(map[string]string{}))
	if err == nil || !connectors.IsValidationError(err) {
		t.Errorf("ValidateCredentials empty: %v", err)
	}
}

func TestTicketmasterConnector_Manifest(t *testing.T) {
	t.Parallel()
	m := New().Manifest()
	if m.ID != "ticketmaster" {
		t.Errorf("Manifest ID = %q", m.ID)
	}
	if len(m.Actions) != 8 {
		t.Fatalf("want 8 actions, got %d", len(m.Actions))
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest.Validate: %v", err)
	}
}

func TestTicketmasterConnector_ActionsMatchManifest(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	manifest := c.Manifest()
	got := make(map[string]bool, len(actions))
	for k := range actions {
		got[k] = true
	}
	for _, a := range manifest.Actions {
		if !got[a.ActionType] {
			t.Errorf("manifest action %q missing from Actions()", a.ActionType)
		}
	}
	for k := range actions {
		found := false
		for _, a := range manifest.Actions {
			if a.ActionType == k {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Actions() has %q not in manifest", k)
		}
	}
}

func TestTicketmasterConnector_DoGET_AppendsAPIKey(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("apikey") != "test_key" {
			http.Error(w, "no key", 400)
			return
		}
		if r.URL.Path != "/discovery/v2/events.json" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"page":{}}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL+"/discovery/v2")
	var raw json.RawMessage
	err := conn.doGET(t.Context(), validCreds(), "events.json", nil, &raw)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTicketmasterConnector_ThrottlesPerSecond(t *testing.T) {
	t.Parallel()
	var first atomic.Int64
	var second atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().UnixNano()
		if first.Load() == 0 {
			first.Store(now)
		} else if second.Load() == 0 {
			second.Store(now)
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL+"/discovery/v2")
	ctx := t.Context()
	var out json.RawMessage
	if err := conn.doGET(ctx, validCreds(), "events.json", nil, &out); err != nil {
		t.Fatal(err)
	}
	if err := conn.doGET(ctx, validCreds(), "events.json", nil, &out); err != nil {
		t.Fatal(err)
	}
	delta := time.Duration(second.Load() - first.Load())
	if delta < minInterval-20*time.Millisecond {
		t.Fatalf("expected ~%v between requests, got %v", minInterval, delta)
	}
}

func TestSearchEventsAction_Execute(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("keyword") != "jazz" {
			t.Errorf("keyword = %q", r.URL.Query().Get("keyword"))
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"_embedded":{"events":[]}}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL+"/discovery/v2")
	action := conn.Actions()["ticketmaster.search_events"]
	params, _ := json.Marshal(map[string]string{"keyword": "jazz"})
	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "ticketmaster.search_events",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Data) == 0 {
		t.Fatal("empty data")
	}
}

func TestGetEventAction_InvalidID(t *testing.T) {
	t.Parallel()
	conn := New()
	action := conn.Actions()["ticketmaster.get_event"]
	params, _ := json.Marshal(map[string]string{"event_id": "../x"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "ticketmaster.get_event",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("got %v", err)
	}
}

func TestSuggestAction_MissingKeyword(t *testing.T) {
	t.Parallel()
	conn := New()
	action := conn.Actions()["ticketmaster.suggest"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "ticketmaster.suggest",
		Parameters:  []byte(`{}`),
		Credentials: validCreds(),
	})
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("got %v", err)
	}
}
