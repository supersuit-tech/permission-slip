package dropbox

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
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
	Autorename *bool  `json:"autorename,omitempty"` // defaults to true when omitted

	// decoded holds the base64-decoded content after validation.
	// Avoids decoding twice (once in validate, once in Execute).
	decoded []byte
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
			Message: fmt.Sprintf("file content is %d MB, which exceeds the maximum of %d MB", len(decoded)>>20, maxUploadBytes>>20),
		}
	}
	if p.Mode != "" && p.Mode != "add" && p.Mode != "overwrite" {
		return &connectors.ValidationError{Message: "mode must be \"add\" or \"overwrite\""}
	}
	p.decoded = decoded
	return nil
}

type uploadAPIArg struct {
	Path       string `json:"path"`
	Mode       string `json:"mode"`
	Autorename bool   `json:"autorename"`
}

type uploadResponse struct {
	Name           string `json:"name"`
	PathDisplay    string `json:"path_display"`
	ID             string `json:"id"`
	Size           int64  `json:"size"`
	ContentHash    string `json:"content_hash"`
	ServerModified string `json:"server_modified"`
}

func (a *uploadFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params uploadFileParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	mode := params.Mode
	if mode == "" {
		mode = "add"
	}

	// Default autorename to true when omitted (matches manifest schema default).
	autorename := true
	if params.Autorename != nil {
		autorename = *params.Autorename
	}

	apiArg := uploadAPIArg{
		Path:       params.Path,
		Mode:       mode,
		Autorename: autorename,
	}

	var resp uploadResponse
	if err := a.conn.doContentUpload(ctx, "files/upload", req.Credentials, apiArg, bytes.NewReader(params.decoded), &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"name":            resp.Name,
		"path_display":    resp.PathDisplay,
		"id":              resp.ID,
		"size":            resp.Size,
		"content_hash":    resp.ContentHash,
		"server_modified": resp.ServerModified,
	})
}
