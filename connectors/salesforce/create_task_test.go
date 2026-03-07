package salesforce

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateTask_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/services/data/v62.0/sobjects/Task/" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["Subject"] != "Follow up with prospect" {
			t.Errorf("expected Subject 'Follow up with prospect', got %q", body["Subject"])
		}
		if body["Status"] != "Not Started" {
			t.Errorf("expected default Status 'Not Started', got %q", body["Status"])
		}
		if body["WhoId"] != "003xx0000000001" {
			t.Errorf("expected WhoId '003xx0000000001', got %q", body["WhoId"])
		}
		if body["Priority"] != "High" {
			t.Errorf("expected Priority 'High', got %q", body["Priority"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sfCreateResponse{
			ID:      "00Txx0000000001",
			Success: true,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createTaskAction{conn: conn}

	params, _ := json.Marshal(createTaskParams{
		Subject:  "Follow up with prospect",
		WhoID:    "003xx0000000001",
		Priority: "High",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_task",
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
	if data["id"] != "00Txx0000000001" {
		t.Errorf("expected id '00Txx0000000001', got %v", data["id"])
	}
}

func TestCreateTask_CustomStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["Status"] != "In Progress" {
			t.Errorf("expected Status 'In Progress', got %q", body["Status"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sfCreateResponse{ID: "00Txx0000000002", Success: true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createTaskAction{conn: conn}

	params, _ := json.Marshal(createTaskParams{
		Subject: "Review proposal",
		Status:  "In Progress",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_task",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateTask_MissingSubject(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createTaskAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"who_id": "003xx"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_task",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing subject")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateTask_AllOptionalFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["WhatId"] != "006xx0000000001" {
			t.Errorf("expected WhatId '006xx0000000001', got %q", body["WhatId"])
		}
		if body["ActivityDate"] != "2024-06-15" {
			t.Errorf("expected ActivityDate '2024-06-15', got %q", body["ActivityDate"])
		}
		if body["Description"] != "Call the customer" {
			t.Errorf("expected Description 'Call the customer', got %q", body["Description"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sfCreateResponse{ID: "00Txx0000000003", Success: true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createTaskAction{conn: conn}

	params, _ := json.Marshal(createTaskParams{
		Subject:     "Follow up",
		WhatID:      "006xx0000000001",
		WhoID:       "003xx0000000001",
		Status:      "Not Started",
		Priority:    "Normal",
		DueDate:     "2024-06-15",
		Description: "Call the customer",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "salesforce.create_task",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
