package linkedin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAddComment_Success(t *testing.T) {
	t.Parallel()

	var gotBody linkedInCommentRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/userinfo":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userinfoResponse{Sub: "person123"})
		case strings.HasPrefix(r.URL.Path, "/socialActions/"):
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if got := r.Header.Get("LinkedIn-Version"); got != linkedInVersion {
				t.Errorf("expected LinkedIn-Version %q, got %q", linkedInVersion, got)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Errorf("failed to decode request body: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &addCommentAction{conn: conn}

	params, _ := json.Marshal(addCommentParams{
		PostURN: "urn:li:share:123456",
		Text:    "Great post!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.add_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotBody.Actor != "urn:li:person:person123" {
		t.Errorf("expected actor 'urn:li:person:person123', got %q", gotBody.Actor)
	}
	if gotBody.Message.Text != "Great post!" {
		t.Errorf("expected message text 'Great post!', got %q", gotBody.Message.Text)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["status"] != "created" {
		t.Errorf("expected status 'created', got %q", data["status"])
	}
}

func TestAddComment_MissingPostURN(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addCommentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"text": "Hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.add_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing post_urn")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddComment_MissingText(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addCommentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"post_urn": "urn:li:share:123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.add_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing text")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddComment_TextTooLong(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addCommentAction{conn: conn}

	params, _ := json.Marshal(addCommentParams{
		PostURN: "urn:li:share:123",
		Text:    strings.Repeat("a", maxCommentTextLen+1),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.add_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for text too long")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddComment_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addCommentAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.add_comment",
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
