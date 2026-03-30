package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateChannel_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/conversations.create" {
			t.Errorf("expected path /conversations.create, got %s", r.URL.Path)
		}

		var body createChannelParams
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Name != "new-channel" {
			t.Errorf("expected name 'new-channel', got %q", body.Name)
		}
		if body.IsPrivate {
			t.Error("expected is_private to be false")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channel": map[string]string{
				"id":   "C09876543",
				"name": "new-channel",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createChannelAction{conn: conn}

	params, _ := json.Marshal(createChannelParams{
		Name: "new-channel",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.create_channel",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "C09876543" {
		t.Errorf("expected id 'C09876543', got %q", data["id"])
	}
	if data["name"] != "new-channel" {
		t.Errorf("expected name 'new-channel', got %q", data["name"])
	}
}

func TestCreateChannel_PrivateChannel(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body createChannelParams
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if !body.IsPrivate {
			t.Error("expected is_private to be true")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channel": map[string]string{
				"id":   "C11111111",
				"name": "secret-channel",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createChannelAction{conn: conn}

	params, _ := json.Marshal(createChannelParams{
		Name:      "secret-channel",
		IsPrivate: true,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.create_channel",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["name"] != "secret-channel" {
		t.Errorf("expected name 'secret-channel', got %q", data["name"])
	}
}

func TestCreateChannel_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createChannelAction{conn: conn}

	params, _ := json.Marshal(map[string]bool{
		"is_private": true,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.create_channel",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateChannel_NameTaken(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "name_taken",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createChannelAction{conn: conn}

	params, _ := json.Marshal(createChannelParams{
		Name: "existing-channel",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.create_channel",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for name_taken")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError for name_taken, got: %T", err)
	}
}

func TestCreateChannel_SlackAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "restricted_action",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createChannelAction{conn: conn}

	params, _ := json.Marshal(createChannelParams{
		Name: "restricted",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.create_channel",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for Slack API error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}

func TestCreateChannel_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createChannelAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.create_channel",
		Parameters:  []byte(`not json`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
