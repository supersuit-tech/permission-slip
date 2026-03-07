package monday

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchItemsAction implements connectors.Action for monday.search_items.
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
	if p.Limit < 0 {
		return &connectors.ValidationError{Message: "limit must be non-negative"}
	}
	return nil
}

type searchItemResult struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	ColumnValues []searchItemColumn `json:"column_values"`
}

type searchItemColumn struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

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

	// Build the query based on whether we have filter criteria.
	var query string
	variables := map[string]any{
		"board_id": params.BoardID,
		"limit":    limit,
	}

	if params.ColumnID != "" && params.ColumnValue != "" {
		// Filter by column value.
		query = `query ($board_id: [ID!]!, $limit: Int!) {
			boards(ids: $board_id) {
				items_page(limit: $limit, query_params: {rules: [{column_id: "` + params.ColumnID + `", compare_value: ["` + params.ColumnValue + `"]}]}) {
					items {
						id
						name
						column_values {
							id
							title: column {
								title
							}
							text
						}
					}
				}
			}
		}`
	} else if params.Query != "" {
		// Text search - use items_page with query_params.
		query = `query ($board_id: [ID!]!, $limit: Int!, $query: String!) {
			boards(ids: $board_id) {
				items_page(limit: $limit, query_params: {rules: [{column_id: "name", compare_value: [$query]}]}) {
					items {
						id
						name
						column_values {
							id
							title: column {
								title
							}
							text
						}
					}
				}
			}
		}`
		variables["query"] = params.Query
	} else {
		// No filter - return all items.
		query = `query ($board_id: [ID!]!, $limit: Int!) {
			boards(ids: $board_id) {
				items_page(limit: $limit) {
					items {
						id
						name
						column_values {
							id
							title: column {
								title
							}
							text
						}
					}
				}
			}
		}`
	}

	// The board_id variable needs to be an array for boards(ids:).
	variables["board_id"] = []string{params.BoardID}

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

	// Flatten the response.
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
