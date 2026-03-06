package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listDocumentsAction implements connectors.Action for google.list_documents.
// It lists Google Docs via the Drive API GET /drive/v3/files with a
// mimeType filter for Google Docs.
type listDocumentsAction struct {
	conn *GoogleConnector
}

// listDocumentsParams is the user-facing parameter schema.
type listDocumentsParams struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
}

func (p *listDocumentsParams) normalize() {
	if p.MaxResults <= 0 {
		p.MaxResults = 10
	}
	if p.MaxResults > 100 {
		p.MaxResults = 100
	}
}

// driveListResponse is the Google Drive API response from files.list.
type driveListResponse struct {
	Files []driveFile `json:"files"`
}

type driveFile struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CreatedTime  string `json:"createdTime"`
	ModifiedTime string `json:"modifiedTime"`
	WebViewLink  string `json:"webViewLink"`
}

// documentSummary is the shape returned to the agent.
type documentSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	CreatedTime  string `json:"created_time,omitempty"`
	ModifiedTime string `json:"modified_time,omitempty"`
	WebViewLink  string `json:"web_view_link,omitempty"`
}

// Execute lists Google Docs from Drive and returns their metadata.
func (a *listDocumentsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listDocumentsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	params.normalize()

	q := "mimeType='application/vnd.google-apps.document'"
	if params.Query != "" {
		q += " and name contains '" + escapeDriveQuery(params.Query) + "'"
	}

	queryParams := url.Values{}
	queryParams.Set("q", q)
	queryParams.Set("pageSize", strconv.Itoa(params.MaxResults))
	queryParams.Set("fields", "files(id,name,createdTime,modifiedTime,webViewLink)")
	queryParams.Set("orderBy", "modifiedTime desc")

	var resp driveListResponse
	listURL := a.conn.driveBaseURL + "/drive/v3/files?" + queryParams.Encode()
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, listURL, nil, &resp); err != nil {
		return nil, err
	}

	documents := make([]documentSummary, 0, len(resp.Files))
	for _, f := range resp.Files {
		documents = append(documents, documentSummary{
			ID:           f.ID,
			Name:         f.Name,
			CreatedTime:  f.CreatedTime,
			ModifiedTime: f.ModifiedTime,
			WebViewLink:  f.WebViewLink,
		})
	}

	return connectors.JSONResult(map[string]any{
		"documents": documents,
	})
}

// escapeDriveQuery escapes single quotes and backslashes in a Drive query
// string value to prevent query injection.
func escapeDriveQuery(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			result = append(result, '\\', '\\')
		case '\'':
			result = append(result, '\\', '\'')
		default:
			result = append(result, s[i])
		}
	}
	return string(result)
}
