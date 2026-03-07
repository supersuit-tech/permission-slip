package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateEmailCampaign_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/marketing/v3/emails" {
			t.Errorf("expected path /marketing/v3/emails, got %s", r.URL.Path)
		}

		var body emailCampaignRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Name != "March Newsletter" {
			t.Errorf("expected name March Newsletter, got %q", body.Name)
		}
		if body.Subject != "Big News" {
			t.Errorf("expected subject Big News, got %q", body.Subject)
		}
		if body.SendNow {
			t.Error("expected sendNow to be false")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(emailCampaignResponse{
			ID:      "email-1",
			Name:    body.Name,
			Subject: body.Subject,
			State:   "DRAFT",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createEmailCampaignAction{conn: conn}

	params, _ := json.Marshal(createEmailCampaignParams{
		Name:    "March Newsletter",
		Subject: "Big News",
		Content: "<h1>Hello</h1>",
		ListIDs: []string{"10", "20"},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_email_campaign",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data emailCampaignResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.ID != "email-1" {
		t.Errorf("expected id email-1, got %q", data.ID)
	}
	if data.State != "DRAFT" {
		t.Errorf("expected state DRAFT, got %q", data.State)
	}
}

func TestCreateEmailCampaign_SendNow(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body emailCampaignRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if !body.SendNow {
			t.Error("expected sendNow to be true")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(emailCampaignResponse{
			ID:    "email-2",
			State: "SENT",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createEmailCampaignAction{conn: conn}

	params, _ := json.Marshal(createEmailCampaignParams{
		Name:    "Urgent Update",
		Subject: "Action Required",
		Content: "<p>Please read</p>",
		ListIDs: []string{"10"},
		SendNow: true,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_email_campaign",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateEmailCampaign_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createEmailCampaignAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"subject": "Hi", "content": "Hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_email_campaign",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateEmailCampaign_MissingSubject(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createEmailCampaignAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"name": "Test", "content": "Hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_email_campaign",
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

func TestCreateEmailCampaign_SendNowWithoutListIDs(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createEmailCampaignAction{conn: conn}

	params, _ := json.Marshal(createEmailCampaignParams{
		Name:    "Test",
		Subject: "Hi",
		Content: "Hello",
		SendNow: true,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_email_campaign",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when send_now is true without list_ids")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateEmailCampaign_MissingContent(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createEmailCampaignAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"name": "Test", "subject": "Hi"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_email_campaign",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing content")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateEmailCampaign_InvalidListID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createEmailCampaignAction{conn: conn}

	params, _ := json.Marshal(createEmailCampaignParams{
		Name:    "Test",
		Subject: "Hi",
		Content: "Hello",
		ListIDs: []string{"10", "not-a-number"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_email_campaign",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid list ID")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
