package microsoft

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGraphMessageToThreadMessage_HTML(t *testing.T) {
	var from graphMailAddress
	from.EmailAddress.Address = "a@b.com"
	var to graphMailAddress
	to.EmailAddress.Address = "c@d.com"
	m := &graphFullMessage{
		ID:               "1",
		Subject:          "S",
		ReceivedDateTime: "2020-01-01T00:00:00Z",
		BodyPreview:      "pv",
		From:             &from,
		ToRecipients:     []*graphMailAddress{&to},
		Body:             graphEmailBody{ContentType: "HTML", Content: "<p>x</p>"},
	}
	em := graphMessageToThreadMessage(m)
	if em.BodyHTML != "<p>x</p>" {
		t.Fatalf("html: %q", em.BodyHTML)
	}
	if em.BodyText == "" {
		t.Fatal("expected body_text from html")
	}
}

func TestBuildMicrosoftEmailThread_Conversation(t *testing.T) {
	msg1 := `{"id":"m1","subject":"Subj","conversationId":"conv1","receivedDateTime":"2020-01-01T00:00:00Z","bodyPreview":"p1","from":{"emailAddress":{"address":"a@b.com"}},"toRecipients":[{"emailAddress":{"address":"c@d.com"}}],"body":{"contentType":"text","content":"hello"},"attachments":[{"name":"f.txt","size":3}]}`
	msg2 := `{"id":"m2","subject":"Subj","conversationId":"conv1","receivedDateTime":"2020-01-02T00:00:00Z","bodyPreview":"p2","from":{"emailAddress":{"address":"c@d.com"}},"body":{"contentType":"HTML","content":"<b>late</b>"}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/me/messages/m2" {
			w.Write([]byte(msg2))
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/me/messages" {
			w.Write([]byte(`{"value":[` + msg1 + `,` + msg2 + `]}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	th, err := c.buildMicrosoftEmailThread(context.Background(), validCreds(), "m2")
	if err != nil {
		t.Fatal(err)
	}
	if th.Subject != "Subj" {
		t.Errorf("subject %q", th.Subject)
	}
	if len(th.Messages) != 2 {
		t.Fatalf("messages %d", len(th.Messages))
	}
	if th.Messages[1].BodyHTML != "<b>late</b>" {
		t.Errorf("last html %q", th.Messages[1].BodyHTML)
	}
	if len(th.Messages[0].Attachments) != 1 || th.Messages[0].Attachments[0].Filename != "f.txt" {
		t.Errorf("att %+v", th.Messages[0].Attachments)
	}
}

func TestBuildMicrosoftEmailThread_Truncation(t *testing.T) {
	long := strings.Repeat("w", connectors.MaxEmailThreadBodyRunes+50)
	msg := map[string]any{
		"id": "m1", "subject": "S", "conversationId": "c",
		"receivedDateTime": "2020-01-01T00:00:00Z",
		"body":             map[string]string{"contentType": "text", "content": long},
	}
	b, _ := json.Marshal(msg)
	page := `{"value":[` + string(b) + `]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/me/messages/m1" {
			w.Write(b)
			return
		}
		if r.URL.Path == "/me/messages" {
			w.Write([]byte(page))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c := newForTest(srv.Client(), srv.URL)
	th, err := c.buildMicrosoftEmailThread(context.Background(), validCreds(), "m1")
	if err != nil {
		t.Fatal(err)
	}
	if len(th.Messages) != 1 || !th.Messages[0].Truncated {
		t.Fatalf("trunc: %+v", th.Messages)
	}
}
