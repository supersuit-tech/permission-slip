package salesforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDescribeObject_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/services/data/v62.0/sobjects/Opportunity/describe" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"name":   "Opportunity",
			"label":  "Opportunity",
			"fields": []map[string]any{{"name": "Id", "type": "id"}},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &describeObjectAction{conn: conn}

	params, _ := json.Marshal(describeObjectParams{SObjectType: "Opportunity"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.describe_object",
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
	if data["name"] != "Opportunity" {
		t.Errorf("expected name 'Opportunity', got %v", data["name"])
	}
}

func TestDescribeObject_MissingSObjectType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &describeObjectAction{conn: conn}

	params, _ := json.Marshal(map[string]any{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.describe_object",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing sobject_type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDescribeObject_InvalidSObjectType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &describeObjectAction{conn: conn}

	tests := []struct {
		name        string
		sobjectType string
	}{
		{"path traversal", "../../../etc/passwd"},
		{"starts with number", "1Lead"},
		{"contains slash", "Lead/Account"},
		{"too long", "ANameThatIsWayTooLongForASalesforceObjectTypeNameBecauseItExceedsTheLimitOf50Characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := json.Marshal(map[string]any{"sobject_type": tt.sobjectType})
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "salesforce.describe_object",
				Parameters:  params,
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatalf("expected error for invalid sobject_type %q", tt.sobjectType)
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got: %T", err)
			}
		})
	}
}
