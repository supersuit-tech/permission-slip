package linkedin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDeletePost_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		// The URN is URL-encoded in the request; the server sees either
		// the decoded path or the raw path depending on Go version.
		expectedDecoded := "/posts/urn:li:share:123456"
		expectedEncoded := "/posts/urn%3Ali%3Ashare%3A123456"
		if r.URL.Path != expectedDecoded && r.URL.RawPath != expectedEncoded {
			t.Errorf("expected path %q or raw path %q, got path=%q raw=%q", expectedDecoded, expectedEncoded, r.URL.Path, r.URL.RawPath)
		}
		if got := r.Header.Get("LinkedIn-Version"); got != linkedInVersion {
			t.Errorf("expected LinkedIn-Version %q, got %q", linkedInVersion, got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &deletePostAction{conn: conn}

	params, _ := json.Marshal(deletePostParams{PostURN: "urn:li:share:123456"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.delete_post",
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
	if data["status"] != "deleted" {
		t.Errorf("expected status 'deleted', got %q", data["status"])
	}
	if data["post_urn"] != "urn:li:share:123456" {
		t.Errorf("expected post_urn 'urn:li:share:123456', got %q", data["post_urn"])
	}
}

func TestDeletePost_MissingPostURN(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deletePostAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.delete_post",
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

func TestDeletePost_Forbidden(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(linkedInErrorResponse{
			Status:  403,
			Message: "Not authorized to delete this post",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &deletePostAction{conn: conn}

	params, _ := json.Marshal(deletePostParams{PostURN: "urn:li:share:999"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.delete_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for forbidden")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T (%v)", err, err)
	}
}

func TestDeletePost_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deletePostAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.delete_post",
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
