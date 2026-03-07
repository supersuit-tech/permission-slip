package salesforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestQuery_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.URL.Query().Get("q"); got != "SELECT Id, Name FROM Lead WHERE Status = 'Open'" {
			t.Errorf("unexpected SOQL query: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"totalSize": 2,
			"done":      true,
			"records": []map[string]any{
				{"Id": "00Qxx0000000001", "Name": "Alice"},
				{"Id": "00Qxx0000000002", "Name": "Bob"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &queryAction{conn: conn}

	params, _ := json.Marshal(queryParams{
		SOQL: "SELECT Id, Name FROM Lead WHERE Status = 'Open'",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.query",
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
	if data["total_size"] != float64(2) {
		t.Errorf("expected total_size 2, got %v", data["total_size"])
	}
	if data["done"] != true {
		t.Errorf("expected done true, got %v", data["done"])
	}
	records, ok := data["records"].([]any)
	if !ok || len(records) != 2 {
		t.Errorf("expected 2 records, got %v", data["records"])
	}
}

func TestQuery_MaxRecordsTruncation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"totalSize": 3,
			"done":      true,
			"records": []map[string]any{
				{"Id": "001"},
				{"Id": "002"},
				{"Id": "003"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &queryAction{conn: conn}

	params, _ := json.Marshal(queryParams{
		SOQL:       "SELECT Id FROM Account",
		MaxRecords: 2,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.query",
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
	records, ok := data["records"].([]any)
	if !ok || len(records) != 2 {
		t.Errorf("expected 2 records after truncation, got %d", len(records))
	}
}

func TestQuery_MissingSOQL(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &queryAction{conn: conn}

	params, _ := json.Marshal(map[string]any{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.query",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing soql")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestQuery_MalformedQuery(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode([]sfAPIError{{ErrorCode: "MALFORMED_QUERY", Message: "unexpected token: SLECT"}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &queryAction{conn: conn}

	params, _ := json.Marshal(queryParams{SOQL: "SLECT Id FROM Lead"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.query",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for malformed query")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T (%v)", err, err)
	}
}

func TestQuery_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &queryAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.query",
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
