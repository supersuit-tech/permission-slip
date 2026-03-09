package dropbox

import (
	"context"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// shareFileAction implements connectors.Action for dropbox.share_file.
type shareFileAction struct {
	conn *DropboxConnector
}

type shareFileParams struct {
	Path                string `json:"path"`
	RequestedVisibility string `json:"requested_visibility,omitempty"`
	LinkPassword        string `json:"link_password,omitempty"`
	Expires             string `json:"expires,omitempty"`
}

func (p *shareFileParams) validate() error {
	if err := validatePath(p.Path, "path"); err != nil {
		return err
	}
	if p.RequestedVisibility != "" && p.RequestedVisibility != "public" && p.RequestedVisibility != "team_only" && p.RequestedVisibility != "password" {
		return &connectors.ValidationError{Message: "requested_visibility must be \"public\", \"team_only\", or \"password\""}
	}
	if p.RequestedVisibility == "password" && p.LinkPassword == "" {
		return &connectors.ValidationError{Message: "link_password is required when requested_visibility is \"password\""}
	}
	return nil
}

type shareSettings struct {
	RequestedVisibility string `json:"requested_visibility,omitempty"`
	LinkPassword        string `json:"link_password,omitempty"`
	Expires             string `json:"expires,omitempty"`
}

type shareRequest struct {
	Path     string         `json:"path"`
	Settings *shareSettings `json:"settings,omitempty"`
}

type shareResponse struct {
	URL       string `json:"url"`
	PathLower string `json:"path_lower"`
	Name      string `json:"name"`
}

func (a *shareFileAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params shareFileParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := shareRequest{Path: params.Path}
	if params.RequestedVisibility != "" || params.LinkPassword != "" || params.Expires != "" {
		body.Settings = &shareSettings{
			RequestedVisibility: params.RequestedVisibility,
			LinkPassword:        params.LinkPassword,
			Expires:             params.Expires,
		}
	}

	var resp shareResponse
	if err := a.conn.doRPC(ctx, "sharing/create_shared_link_with_settings", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"url":        resp.URL,
		"path_lower": resp.PathLower,
		"name":       resp.Name,
	})
}
