package slack

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestMarkRead_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/conversations.mark" {
			t.Errorf("unexpected path %s", r.URL.Path)
			return
		}
		var body markReadRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if body.Channel != "C01234567" || body.TS != "1678900000.000100" {
			t.Errorf("unexpected body %+v", body)
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &markReadAction{conn: conn}

	params, _ := json.Marshal(markReadParams{
		ChannelID: "C01234567",
		TS:        "1678900000.000100",
	})

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.mark_read",
		Parameters:  params,
		Credentials: validCreds(),
		UserEmail:   "user@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(res.Data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out["ok"] != true {
		t.Fatalf("expected ok true, got %+v", out)
	}
}

func TestMarkRead_MissingTS(t *testing.T) {
	t.Parallel()

	conn := newForTest(http.DefaultClient, "http://unused.example")
	action := &markReadAction{conn: conn}

	params, _ := json.Marshal(markReadParams{
		ChannelID: "C01234567",
		TS:        "",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.mark_read",
		Parameters:  params,
		Credentials: validCreds(),
		UserEmail:   "user@example.com",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestMarkRead_InvalidChannelID(t *testing.T) {
	t.Parallel()

	conn := newForTest(http.DefaultClient, "http://unused.example")
	action := &markReadAction{conn: conn}

	params, _ := json.Marshal(markReadParams{
		ChannelID: "not-a-channel-id",
		TS:        "1678900000.000100",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.mark_read",
		Parameters:  params,
		Credentials: validCreds(),
		UserEmail:   "user@example.com",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestMarkRead_ChannelNotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/conversations.mark" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "channel_not_found"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &markReadAction{conn: conn}

	params, _ := json.Marshal(markReadParams{
		ChannelID: "C01234567",
		TS:        "1678900000.000100",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.mark_read",
		Parameters:  params,
		Credentials: validCreds(),
		UserEmail:   "user@example.com",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMarkRead_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.mark" {
			t.Errorf("unexpected path %s", r.URL.Path)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "ratelimited"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &markReadAction{conn: conn}

	params, _ := json.Marshal(markReadParams{
		ChannelID: "C01234567",
		TS:        "1678900000.000100",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "slack.mark_read",
		Parameters:  params,
		Credentials: validCreds(),
		UserEmail:   "user@example.com",
	})
	if err == nil {
		t.Fatal("expected rate limit error")
	}
	if !connectors.IsRateLimitError(err) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
}
