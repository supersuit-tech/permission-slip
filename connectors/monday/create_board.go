package monday

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type createBoardAction struct {
	conn *MondayConnector
}

type createBoardParams struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	FolderID  string `json:"folder_id"`
	WorkspaceID string `json:"workspace_id"`
}

func (p *createBoardParams) validate() error {
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if p.Kind != "" && !validBoardKinds[p.Kind] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid kind %q: must be one of public, private, share", p.Kind)}
	}
	return nil
}

func (a *createBoardAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createBoardParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	kind := params.Kind
	if kind == "" {
		kind = "public"
	}

	query := `mutation ($board_name: String!, $board_kind: BoardKind!, $folder_id: ID, $workspace_id: ID) {
		create_board(board_name: $board_name, board_kind: $board_kind, folder_id: $folder_id, workspace_id: $workspace_id) {
			id
			name
			board_kind
			url
		}
	}`

	variables := map[string]any{
		"board_name": params.Name,
		"board_kind": kind,
	}
	if params.FolderID != "" {
		variables["folder_id"] = params.FolderID
	}
	if params.WorkspaceID != "" {
		variables["workspace_id"] = params.WorkspaceID
	}

	var data struct {
		CreateBoard struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			BoardKind string `json:"board_kind"`
			URL       string `json:"url"`
		} `json:"create_board"`
	}

	if err := a.conn.query(ctx, req.Credentials, query, variables, &data); err != nil {
		return nil, err
	}

	return connectors.JSONResult(data.CreateBoard)
}
