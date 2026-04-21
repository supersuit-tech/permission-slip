package google

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func httptestNewJSONServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))
}

func TestBuildGmailEmailThread_MultipartAndAttachments(t *testing.T) {
	threadJSON := `{
		"id": "th1",
		"messages": [
			{
				"id": "m1",
				"internalDate": "1000",
				"snippet": "snip",
				"payload": {
					"mimeType": "multipart/alternative",
					"headers": [
						{"name": "Subject", "value": "Thread Subj"},
						{"name": "From", "value": "a@x.com"},
						{"name": "To", "value": "b@x.com"},
						{"name": "Cc", "value": "c@x.com"}
					],
					"parts": [
						{
							"mimeType": "text/plain",
							"body": {"data": "VGV4dA=="}
						},
						{
							"mimeType": "text/html",
							"body": {"data": "PHA+SFRNTDwvcD4="}
						}
					]
				}
			},
			{
				"id": "m2",
				"internalDate": "2000",
				"snippet": "later",
				"payload": {
					"mimeType": "multipart/mixed",
					"headers": [
						{"name": "From", "value": "b@x.com"}
					],
					"parts": [
						{
							"mimeType": "multipart/alternative",
							"parts": [
								{"mimeType": "text/plain", "body": {"data": "TGF0ZXI="}}
							]
						},
						{
							"mimeType": "application/pdf",
							"filename": "f.pdf",
							"body": {"attachmentId": "att1", "size": 42}
						}
					]
				}
			}
		]
	}`

	srv := httptestNewJSONServer(t, threadJSON)
	defer srv.Close()

	c := newGmailForTest(srv.Client(), srv.URL)
	th, err := c.buildGmailEmailThread(context.Background(), validCreds(), "th1")
	if err != nil {
		t.Fatalf("buildGmailEmailThread: %v", err)
	}
	if th.Subject != "Thread Subj" {
		t.Errorf("subject: got %q", th.Subject)
	}
	if len(th.Messages) != 2 {
		t.Fatalf("messages: got %d", len(th.Messages))
	}
	if th.Messages[0].MessageID != "m1" || th.Messages[1].MessageID != "m2" {
		t.Errorf("order: %#v", th.Messages)
	}
	if th.Messages[0].BodyText != "Text" {
		t.Errorf("m1 text: %q", th.Messages[0].BodyText)
	}
	if th.Messages[0].BodyHTML != "<p>HTML</p>" {
		t.Errorf("m1 html: %q", th.Messages[0].BodyHTML)
	}
	if len(th.Messages[1].Attachments) != 1 || th.Messages[1].Attachments[0].Filename != "f.pdf" {
		t.Errorf("attachments: %+v", th.Messages[1].Attachments)
	}
}

func TestBuildGmailEmailThread_Truncation(t *testing.T) {
	longBody := strings.Repeat("z", connectors.MaxEmailThreadBodyRunes+100)
	plainB64 := base64.StdEncoding.EncodeToString([]byte(longBody))
	threadJSON := `{
		"id": "th2",
		"messages": [{
			"id": "m1",
			"internalDate": "1",
			"payload": {
				"mimeType": "text/plain",
				"headers": [{"name": "Subject", "value": "S"}],
				"body": {"data": "` + plainB64 + `"}
			}
		}]
	}`
	srv := httptestNewJSONServer(t, threadJSON)
	defer srv.Close()
	c := newGmailForTest(srv.Client(), srv.URL)
	th, err := c.buildGmailEmailThread(context.Background(), validCreds(), "th2")
	if err != nil {
		t.Fatal(err)
	}
	if !th.Messages[0].Truncated {
		t.Fatal("expected truncated")
	}
	if len(th.Messages[0].BodyText) != connectors.MaxEmailThreadBodyRunes {
		t.Fatalf("len %d", len(th.Messages[0].BodyText))
	}
}
