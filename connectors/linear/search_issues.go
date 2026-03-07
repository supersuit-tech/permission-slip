package linear

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchIssuesAction implements connectors.Action for linear.search_issues.
type searchIssuesAction struct {
	conn *LinearConnector
}

type searchIssuesParams struct {
	Query      string `json:"query"`
	TeamID     string `json:"team_id,omitempty"`
	AssigneeID string `json:"assignee_id,omitempty"`
	State      string `json:"state,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

const defaultSearchLimit = 50
const maxSearchLimit = 100

func (p *searchIssuesParams) validate() error {
	if p.Query == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query"}
	}
	if p.Limit < 0 {
		return &connectors.ValidationError{Message: "limit must be a non-negative integer"}
	}
	if p.Limit > maxSearchLimit {
		return &connectors.ValidationError{Message: fmt.Sprintf("limit must not exceed %d", maxSearchLimit)}
	}
	return nil
}

// searchIssuesQueryFiltered uses the issues endpoint with filters. Used when
// team, assignee, or state filters are specified alongside the query.
const searchIssuesQueryFiltered = `query SearchIssues($filter: IssueFilter, $first: Int) {
	issues(filter: $filter, first: $first) {
		nodes {
			id
			identifier
			title
			description
			priority
			url
			state {
				name
			}
			assignee {
				name
			}
		}
	}
}`

// searchIssuesQueryFullText uses Linear's issueSearch for full-text search
// across titles, descriptions, and comments. Used when no filters are specified.
const searchIssuesQueryFullText = `query IssueSearch($query: String!, $first: Int) {
	issueSearch(query: $query, first: $first) {
		nodes {
			id
			identifier
			title
			description
			priority
			url
			state {
				name
			}
			assignee {
				name
			}
		}
	}
}`

type searchIssueNode struct {
	ID          string `json:"id"`
	Identifier  string `json:"identifier"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	URL         string `json:"url"`
	State       struct {
		Name string `json:"name"`
	} `json:"state"`
	Assignee *struct {
		Name string `json:"name"`
	} `json:"assignee"`
}

type searchIssuesFilteredResponse struct {
	Issues struct {
		Nodes []searchIssueNode `json:"nodes"`
	} `json:"issues"`
}

type searchIssuesFullTextResponse struct {
	IssueSearch struct {
		Nodes []searchIssueNode `json:"nodes"`
	} `json:"issueSearch"`
}

type searchIssueResult struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	Priority   string `json:"priority"`
	URL        string `json:"url"`
	State      string `json:"state"`
	Assignee   string `json:"assignee"`
}

type searchIssuesResult struct {
	Issues     []searchIssueResult `json:"issues"`
	TotalCount int                 `json:"total_count"`
}

func (a *searchIssuesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchIssuesParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit == 0 {
		limit = defaultSearchLimit
	}

	hasFilters := params.TeamID != "" || params.AssigneeID != "" || params.State != ""
	var nodes []searchIssueNode

	if hasFilters {
		// Use the issues endpoint with filters when team/assignee/state
		// are specified. The query parameter is applied as a title filter.
		filter := map[string]any{}
		filter["title"] = map[string]any{"containsIgnoreCase": params.Query}
		if params.TeamID != "" {
			filter["team"] = map[string]any{"id": map[string]any{"eq": params.TeamID}}
		}
		if params.AssigneeID != "" {
			filter["assignee"] = map[string]any{"id": map[string]any{"eq": params.AssigneeID}}
		}
		if params.State != "" {
			filter["state"] = map[string]any{"name": map[string]any{"eqIgnoreCase": params.State}}
		}

		vars := map[string]any{"filter": filter, "first": limit}
		var resp searchIssuesFilteredResponse
		if err := a.conn.doGraphQL(ctx, req.Credentials, searchIssuesQueryFiltered, vars, &resp); err != nil {
			return nil, err
		}
		nodes = resp.Issues.Nodes
	} else {
		// Use issueSearch for full-text search across titles, descriptions,
		// and comments when no structured filters are applied.
		vars := map[string]any{"query": params.Query, "first": limit}
		var resp searchIssuesFullTextResponse
		if err := a.conn.doGraphQL(ctx, req.Credentials, searchIssuesQueryFullText, vars, &resp); err != nil {
			return nil, err
		}
		nodes = resp.IssueSearch.Nodes
	}

	issues := make([]searchIssueResult, 0, len(nodes))
	for _, node := range nodes {
		assignee := ""
		if node.Assignee != nil {
			assignee = node.Assignee.Name
		}
		issues = append(issues, searchIssueResult{
			ID:         node.ID,
			Identifier: node.Identifier,
			Title:      node.Title,
			Priority:   strconv.Itoa(node.Priority),
			URL:        node.URL,
			State:      node.State.Name,
			Assignee:   assignee,
		})
	}

	return connectors.JSONResult(searchIssuesResult{
		Issues:     issues,
		TotalCount: len(issues),
	})
}
