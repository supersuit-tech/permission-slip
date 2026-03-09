package dropbox

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	// maxUploadBytes caps file content at 150 MB (Dropbox simple upload limit).
	maxUploadBytes = 150 << 20 // 150 MB
)

// uploadFileAction implements connectors.Action for dropbox.upload_file.
type uploadFileAction struct {
	conn *DropboxConnector
}

type uploadFileParams struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	Mode       string `json:"mode,omitempty"`
	Autorename bool   `json:"autorename"`
}

func (p *uploadFileParams) validate() error {
	if err := validatePath(p.Path, "path"); err != nil {
		return err
	}
	if p.Content == "" {
		return &connectors.ValidationError{Message: "missing required parameter: content"}
	}
	decoded, err := base64.StdEncoding.DecodeString(p.Content)
	if err != nil {
		return &connectors.ValidationError{Message: "content must be valid base64-encoded data"}
	}
	if len(decoded) > maxUploadBytes {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("file content exceeds maximum size of %d MB", maxUploadBytes>>20),
		}
	}
	if p.Mode != "" && p.Mode != "add" && p.Mode != "overwrite" {
		return &connectors.ValidationError{Message: "mode must be \"add\" or \"overwrite\""}
	}
	return nil
}

type uploadAPIArg struct {
	Path       string `json:"path"`
	Mode       string `json:"mode"`
	Autorename bool   `json:"autorename"`
}

type uploadResponse struct {
	Name        string `json:"name"`
	PathDisplay string `json:"path_display"`
	ID          string `json:"id"`
	Size        int64  `json:"size"`
}

func (a *uploadFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params uploadFileParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	decoded, _ := base64.StdEncoding.DecodeString(params.Content)

	mode := params.Mode
	if mode == "" {
		mode = "add"
	}

	apiArg := uploadAPIArg{
		Path:       params.Path,
		Mode:       mode,
		Autorename: params.Autorename,
	}

	var resp uploadResponse
	if err := a.conn.doContentUpload(ctx, "files/upload", req.Credentials, apiArg, bytes.NewReader(decoded), &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"name":         resp.Name,
		"path_display": resp.PathDisplay,
		"id":           resp.ID,
		"size":         resp.Size,
	})
}
