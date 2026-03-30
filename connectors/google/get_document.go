package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getDocumentAction implements connectors.Action for google.get_document.
// It retrieves a Google Doc via the Docs API GET /v1/documents/{documentId}.
type getDocumentAction struct {
	conn *GoogleConnector
}

// getDocumentParams is the user-facing parameter schema.
type getDocumentParams struct {
	DocumentID string `json:"document_id"`
}

func (p *getDocumentParams) validate() error {
	if p.DocumentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: document_id"}
	}
	return nil
}

// docsGetResponse is the Google Docs API response from documents.get.
type docsGetResponse struct {
	DocumentID string  `json:"documentId"`
	Title      string  `json:"title"`
	Body       docsBody `json:"body"`
}

type docsBody struct {
	Content []docsStructuralElement `json:"content"`
}

type docsStructuralElement struct {
	Paragraph *docsParagraph `json:"paragraph,omitempty"`
}

type docsParagraph struct {
	Elements []docsParagraphElement `json:"elements"`
}

type docsParagraphElement struct {
	TextRun *docsTextRun `json:"textRun,omitempty"`
}

type docsTextRun struct {
	Content string `json:"content"`
}

// extractPlainText walks the structural content tree and extracts plain text.
func extractPlainText(content []docsStructuralElement) string {
	var sb strings.Builder
	for _, elem := range content {
		if elem.Paragraph == nil {
			continue
		}
		for _, pe := range elem.Paragraph.Elements {
			if pe.TextRun != nil {
				sb.WriteString(pe.TextRun.Content)
			}
		}
	}
	return sb.String()
}

// wordCount counts the number of whitespace-separated words in s.
func wordCount(s string) int {
	return len(strings.Fields(s))
}

// Execute retrieves a Google Doc and returns its metadata and plain text content.
func (a *getDocumentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getDocumentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp docsGetResponse
	docURL := a.conn.docsBaseURL + "/v1/documents/" + url.PathEscape(params.DocumentID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, docURL, nil, &resp); err != nil {
		return nil, err
	}

	bodyText := extractPlainText(resp.Body.Content)
	documentURL := documentEditURL(resp.DocumentID)

	return connectors.JSONResult(map[string]any{
		"document_id":  resp.DocumentID,
		"title":        resp.Title,
		"body_text":    bodyText,
		"document_url": documentURL,
		"word_count":   wordCount(bodyText),
	})
}
