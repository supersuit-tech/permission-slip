package protonmail

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

type moveToFolderAction struct {
	conn *ProtonMailConnector
}

type moveToFolderParams struct {
	uidMessageParams
	TargetFolder string `json:"target_folder"`
}

func parseMoveToFolderParams(raw []byte) (*moveToFolderParams, error) {
	base, err := parseUIDMessageParams(raw)
	if err != nil {
		return nil, err
	}

	var extra struct {
		TargetFolder string `json:"target_folder"`
	}
	if err := json.Unmarshal(raw, &extra); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	return &moveToFolderParams{
		uidMessageParams: *base,
		TargetFolder:     extra.TargetFolder,
	}, nil
}

func (p *moveToFolderParams) validate() error {
	if err := p.uidMessageParams.validate(); err != nil {
		return err
	}
	if strings.TrimSpace(p.TargetFolder) == "" {
		return &connectors.ValidationError{Message: "missing required parameter: target_folder"}
	}
	if strings.EqualFold(p.Folder, p.TargetFolder) {
		return &connectors.ValidationError{Message: "target_folder must differ from source folder"}
	}
	return nil
}

func (a *moveToFolderAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseMoveToFolderParams(req.Parameters)
	if err != nil {
		return nil, err
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	err = executeUIDMessageAction(ctx, a.conn, req, &params.uidMessageParams, func(session *imapSession, uidSet imap.UIDSet) error {
		moveCmd := session.client.Move(uidSet, params.TargetFolder)
		if _, err := moveCmd.Wait(); err != nil {
			errMsg := err.Error()
			if strings.Contains(errMsg, "TRYCREATE") || strings.Contains(errMsg, "Mailbox doesn't exist") {
				return &connectors.ExternalError{
					Message: fmt.Sprintf("target folder %q not found on server: %v", params.TargetFolder, err),
				}
			}
			return mapUIDNotFoundError(err, params.Folder)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"status":        "moved",
		"folder":        params.Folder,
		"target_folder": params.TargetFolder,
		"moved":         len(params.MessageIDs),
		"message_ids":   params.MessageIDs,
	})
}
