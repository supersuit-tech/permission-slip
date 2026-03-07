package linkedin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetPostAnalytics_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("LinkedIn-Version"); got != linkedInVersion {
			t.Errorf("expected LinkedIn-Version %q, got %q", linkedInVersion, got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(socialActionsResponse{
			LikesSummary: likesSummary{
				TotalLikes:         42,
				LikedByCurrentUser: true,
			},
			CommentsSummary: commentsSummary{
				TotalComments: 7,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &getPostAnalyticsAction{conn: conn}

	params, _ := json.Marshal(getPostAnalyticsParams{PostURN: "urn:li:share:123456"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.get_post_analytics",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["likes"] != float64(42) {
		t.Errorf("expected likes 42, got %v", data["likes"])
	}
	if data["comments"] != float64(7) {
		t.Errorf("expected comments 7, got %v", data["comments"])
	}
	if data["liked_by_current_user"] != true {
		t.Errorf("expected liked_by_current_user true, got %v", data["liked_by_current_user"])
	}
	if data["post_urn"] != "urn:li:share:123456" {
		t.Errorf("expected post_urn 'urn:li:share:123456', got %v", data["post_urn"])
	}
}

func TestGetPostAnalytics_MissingPostURN(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getPostAnalyticsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.get_post_analytics",
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

func TestGetPostAnalytics_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(linkedInErrorResponse{
			Status:  401,
			Message: "Invalid access token",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &getPostAnalyticsAction{conn: conn}

	params, _ := json.Marshal(getPostAnalyticsParams{PostURN: "urn:li:share:123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.get_post_analytics",
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

func TestGetPostAnalytics_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getPostAnalyticsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.get_post_analytics",
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
