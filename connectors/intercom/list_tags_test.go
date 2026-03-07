package intercom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListTags_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/tags" {
			t.Errorf("expected path /tags, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tagsListResponse{
			Type: "list",
			Data: []tag{
				{Type: "tag", ID: "1", Name: "billing"},
				{Type: "tag", ID: "2", Name: "vip"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listTagsAction{conn: conn}

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.list_tags",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data tagsListResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Data) != 2 {
		t.Errorf("expected 2 tags, got %d", len(data.Data))
	}
	if data.Data[0].Name != "billing" {
		t.Errorf("expected first tag 'billing', got %q", data.Data[0].Name)
	}
}
