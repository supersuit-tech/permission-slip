package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListChatSpaces_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/spaces" {
			t.Errorf("expected path /v1/spaces, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("pageSize"); got != "20" {
			t.Errorf("expected pageSize=20, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatSpacesResponse{
			Spaces: []chatSpace{
				{
					Name:        "spaces/AAAA1234",
					DisplayName: "Engineering",
					Type:        "ROOM",
					SpaceType:   "SPACE",
				},
				{
					Name:        "spaces/BBBB5678",
					DisplayName: "General",
					Type:        "ROOM",
					SpaceType:   "SPACE",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTestWithChat(srv.Client(), "", "", srv.URL)
	action := &listChatSpacesAction{conn: conn}

	params, _ := json.Marshal(listChatSpacesParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_chat_spaces",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Spaces []chatSpaceSummary `json:"spaces"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Spaces) != 2 {
		t.Fatalf("expected 2 spaces, got %d", len(data.Spaces))
	}
	if data.Spaces[0].Name != "spaces/AAAA1234" {
		t.Errorf("expected first space name 'spaces/AAAA1234', got %q", data.Spaces[0].Name)
	}
	if data.Spaces[0].DisplayName != "Engineering" {
		t.Errorf("expected display_name 'Engineering', got %q", data.Spaces[0].DisplayName)
	}
}

func TestListChatSpaces_CustomPageSize(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("pageSize"); got != "5" {
			t.Errorf("expected pageSize=5, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatSpacesResponse{Spaces: []chatSpace{}})
	}))
	defer srv.Close()

	conn := newForTestWithChat(srv.Client(), "", "", srv.URL)
	action := &listChatSpacesAction{conn: conn}

	params, _ := json.Marshal(listChatSpacesParams{PageSize: 5})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_chat_spaces",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListChatSpaces_PageSizeClamped(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("pageSize"); got != "100" {
			t.Errorf("expected pageSize clamped to 100, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(chatSpacesResponse{Spaces: []chatSpace{}})
	}))
	defer srv.Close()

	conn := newForTestWithChat(srv.Client(), "", "", srv.URL)
	action := &listChatSpacesAction{conn: conn}

	params, _ := json.Marshal(listChatSpacesParams{PageSize: 999})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_chat_spaces",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListChatSpaces_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    401,
				"message": "Invalid Credentials",
			},
		})
	}))
	defer srv.Close()

	conn := newForTestWithChat(srv.Client(), "", "", srv.URL)
	action := &listChatSpacesAction{conn: conn}

	params, _ := json.Marshal(listChatSpacesParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_chat_spaces",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T (%v)", err, err)
	}
}

func TestListChatSpaces_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listChatSpacesAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_chat_spaces",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
