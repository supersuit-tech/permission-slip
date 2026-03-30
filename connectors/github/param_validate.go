package github

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *GitHubConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"github.add_comment": makeParamValidator[addCommentParams](),
	"github.add_label": makeParamValidator[addLabelParams](),
	"github.add_reviewer": makeParamValidator[addReviewerParams](),
	"github.close_issue": makeParamValidator[closeIssueParams](),
	"github.create_branch": makeParamValidator[createBranchParams](),
	"github.create_issue": makeParamValidator[createIssueParams](),
	"github.create_or_update_file": makeParamValidator[createOrUpdateFileParams](),
	"github.create_pr": makeParamValidator[createPRParams](),
	"github.create_release": makeParamValidator[createReleaseParams](),
	"github.create_repo": makeParamValidator[createRepoParams](),
	"github.create_webhook": makeParamValidator[createWebhookParams](),
	"github.get_file_contents": makeParamValidator[getFileContentsParams](),
	"github.get_repo": makeParamValidator[getRepoParams](),
	"github.list_pull_requests": makeParamValidator[listPullRequestsParams](),
	"github.list_repos": makeParamValidator[listReposParams](),
	"github.list_workflow_runs": makeParamValidator[listWorkflowRunsParams](),
	"github.merge_pr": makeParamValidator[mergePRParams](),
	"github.search_code": makeParamValidator[searchCodeParams](),
	"github.search_issues": makeParamValidator[searchIssuesParams](),
	"github.trigger_workflow": makeParamValidator[triggerWorkflowParams](),
}

func makeParamValidator[T any, PT interface {
	*T
	validate() error
}]() connectors.ParamValidatorFunc {
	return func(params json.RawMessage) error {
		p := PT(new(T))
		if err := json.Unmarshal(params, p); err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
		}
		return p.validate()
	}
}
