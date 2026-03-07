package airtable

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateRecords_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/appABC123/Tasks" {
			t.Errorf("expected path /appABC123/Tasks, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("expected JSON content type, got %q", got)
		}

		var body createRecordsRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if len(body.Records) != 1 {
			t.Fatalf("expected 1 record, got %d", len(body.Records))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"records": []map[string]any{
				{
					"id":          "recNEW001",
					"createdTime": "2024-01-15T12:00:00.000Z",
					"fields":      body.Records[0].Fields,
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createRecordsAction{conn: conn}

	params, _ := json.Marshal(createRecordsParams{
		BaseID: "appABC123",
		Table:  "Tasks",
		Records: []createRecordInput{
			{Fields: map[string]any{"Name": "New Task", "Status": "Todo"}},
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.create_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listRecordsResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(data.Records))
	}
	if data.Records[0].ID != "recNEW001" {
		t.Errorf("expected record ID 'recNEW001', got %q", data.Records[0].ID)
	}
}

func TestCreateRecords_MissingRecords(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createRecordsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"base_id": "appABC123",
		"table":   "Tasks",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.create_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing records")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateRecords_TooManyRecords(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createRecordsAction{conn: conn}

	records := make([]createRecordInput, 11)
	for i := range records {
		records[i] = createRecordInput{Fields: map[string]any{"Name": "Task"}}
	}

	params, _ := json.Marshal(createRecordsParams{
		BaseID:  "appABC123",
		Table:   "Tasks",
		Records: records,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.create_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for too many records")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateRecords_MissingFields(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createRecordsAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"base_id": "appABC123",
		"table":   "Tasks",
		"records": []map[string]any{{}},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.create_records",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateRecords_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createRecordsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "airtable.create_records",
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
