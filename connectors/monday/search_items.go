package monday

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// searchItemsAction implements connectors.Action for monday.search_items.
// It queries boards(ids:) with optional items_page filtering via query_params.
type searchItemsAction struct {
	conn *MondayConnector
}

type searchItemsParams struct {
	BoardID     string `json:"board_id"`
	Query       string `json:"query,omitempty"`
	ColumnID    string `json:"column_id,omitempty"`
	ColumnValue string `json:"column_value,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

func (p *searchItemsParams) validate() error {
	if p.BoardID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: board_id"}
	}
	if !isValidMondayID(p.BoardID) {
		return &connectors.ValidationError{Message: "board_id must be a numeric string"}
	}
	if (p.ColumnID == "" && p.ColumnValue != "") || (p.ColumnID != "" && p.ColumnValue == "") {
		return &connectors.ValidationError{Message: "column_id and column_value must be provided together"}
	}
	if p.Limit < 0 {
		return &connectors.ValidationError{Message: "limit must be non-negative"}
	}
	return nil
}

type searchItemResult struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	ColumnValues []searchItemColumn `json:"column_values"`
}

type searchItemColumn struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// itemsFragment is the shared GraphQL selection set for items.
const itemsFragment = `
	items {
		id
		name
		column_values {
			id
			title: column { title }
			text
		}
	}
`

func (a *searchItemsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchItemsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit == 0 {
		limit = 20
	}

	variables := map[string]any{
		"board_id": []string{params.BoardID},
		"limit":    limit,
	}

	// Build the query. Column filter values are passed as GraphQL variables
	// to prevent injection — never interpolate user input into the query string.
	var query string
	if params.ColumnID != "" && params.ColumnValue != "" {
		query = `query ($board_id: [ID!]!, $limit: Int!, $column_id: String!, $column_value: String!) {
			boards(ids: $board_id) {
				items_page(limit: $limit, query_params: {rules: [{column_id: $column_id, compare_value: [$column_value]}]}) {` + itemsFragment + `
				}
			}
		}`
		variables["column_id"] = params.ColumnID
		variables["column_value"] = params.ColumnValue
	} else if params.Query != "" {
		query = `query ($board_id: [ID!]!, $limit: Int!, $query: String!) {
			boards(ids: $board_id) {
				items_page(limit: $limit, query_params: {rules: [{column_id: "name", compare_value: [$query]}]}) {` + itemsFragment + `
				}
			}
		}`
		variables["query"] = params.Query
	} else {
		query = `query ($board_id: [ID!]!, $limit: Int!) {
			boards(ids: $board_id) {
				items_page(limit: $limit) {` + itemsFragment + `
				}
			}
		}`
	}

	var data struct {
		Boards []struct {
			ItemsPage struct {
				Items []struct {
					ID           string `json:"id"`
					Name         string `json:"name"`
					ColumnValues []struct {
						ID    string `json:"id"`
						Title struct {
							Title string `json:"title"`
						} `json:"title"`
						Text string `json:"text"`
					} `json:"column_values"`
				} `json:"items"`
			} `json:"items_page"`
		} `json:"boards"`
	}

	if err := a.conn.query(ctx, req.Credentials, query, variables, &data); err != nil {
		return nil, err
	}

	// Flatten the response into a simpler structure.
	var items []searchItemResult
	if len(data.Boards) > 0 {
		for _, item := range data.Boards[0].ItemsPage.Items {
			result := searchItemResult{
				ID:   item.ID,
				Name: item.Name,
			}
			for _, cv := range item.ColumnValues {
				result.ColumnValues = append(result.ColumnValues, searchItemColumn{
					ID:    cv.ID,
					Title: cv.Title.Title,
					Text:  cv.Text,
				})
			}
			items = append(items, result)
		}
	}

	return connectors.JSONResult(map[string]any{
		"items": items,
		"count": len(items),
	})
}
