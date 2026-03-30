package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestInviteToChannel_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.invite" {
			t.Errorf("expected path /conversations.invite, got %s", r.URL.Path)
		}

		var body inviteToChannelRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Channel != "C01234567" {
			t.Errorf("expected channel C01234567, got %q", body.Channel)
		}
		if body.Users != "U111,U222" {
			t.Errorf("expected users 'U111,U222', got %q", body.Users)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"channel": map[string]string{
				"id":   "C01234567",
				"name": "test-channel",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &inviteToChannelAction{conn: conn}

	params, _ := json.Marshal(inviteToChannelParams{
		Channel: "C01234567",
		Users:   "U111, U222",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.invite_to_channel",
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
	if data["channel"] != "C01234567" {
		t.Errorf("expected channel 'C01234567', got %q", data["channel"])
	}
	if data["channel_name"] != "test-channel" {
		t.Errorf("expected channel_name 'test-channel', got %q", data["channel_name"])
	}
}

func TestInviteToChannel_MissingChannel(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &inviteToChannelAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"users": "U111",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.invite_to_channel",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing channel")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestInviteToChannel_MissingUsers(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &inviteToChannelAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"channel": "C01234567",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.invite_to_channel",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing users")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestInviteToChannel_SlackAPIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "cant_invite_self",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &inviteToChannelAction{conn: conn}

	params, _ := json.Marshal(inviteToChannelParams{
		Channel: "C01234567",
		Users:   "U111",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.invite_to_channel",
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
