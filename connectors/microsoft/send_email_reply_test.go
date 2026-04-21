package microsoft

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendEmailReply_HTMLBody(t *testing.T) {
	t.Parallel()
	var gotPath string
	var gotBody graphReplyRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendEmailReplyAction{conn: conn}
	params, _ := json.Marshal(map[string]string{
		"message_id": "msgA",
		"body":       "<p>Hi</p>",
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotPath != "/me/messages/msgA/reply" {
		t.Errorf("path: %s", gotPath)
	}
	if gotBody.Comment != "" {
		t.Errorf("expected message body for HTML, got comment %q", gotBody.Comment)
	}
	if gotBody.Message.Body.ContentType != "HTML" || gotBody.Message.Body.Content != "<p>Hi</p>" {
		t.Errorf("body: %+v", gotBody.Message.Body)
	}
}

func TestSendEmailReply_PlainUsesComment(t *testing.T) {
	t.Parallel()
	var gotBody graphReplyRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendEmailReplyAction{conn: conn}
	htmlFalse := false
	params, _ := json.Marshal(map[string]any{
		"message_id": "msgB",
		"body":       "plain",
		"html":       htmlFalse,
	})
	_, err := action.Execute(context.Background(), connectors.ActionRequest{
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody.Comment != "plain" {
		t.Errorf("comment: %q", gotBody.Comment)
	}
	if gotBody.Message.Body.Content != "" {
		t.Errorf("expected empty message.body for plain, got %+v", gotBody.Message.Body)
	}
}
