package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	// storyContainerPollInterval is how long to wait between polling for
	// container status during Instagram story publishing.
	storyContainerPollInterval = 2 * time.Second

	// maxStoryContainerPolls is the maximum number of times to poll for
	// container readiness before giving up.
	maxStoryContainerPolls = 15
)

// createInstagramStoryAction implements connectors.Action for meta.create_instagram_story.
// Instagram Stories are published using the same two-step media container process
// as regular posts but with media_type=STORIES.
type createInstagramStoryAction struct {
	conn *MetaConnector
}

type createInstagramStoryParams struct {
	InstagramAccountID string `json:"instagram_account_id"`
	ImageURL           string `json:"image_url"`
}

func (p *createInstagramStoryParams) validate() error {
	if p.InstagramAccountID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: instagram_account_id"}
	}
	if !isValidGraphID(p.InstagramAccountID) {
		return &connectors.ValidationError{Message: "instagram_account_id contains invalid characters"}
	}
	if p.ImageURL == "" {
		return &connectors.ValidationError{Message: "missing required parameter: image_url"}
	}
	u, err := url.Parse(p.ImageURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return &connectors.ValidationError{Message: "image_url must be a valid HTTPS URL (Instagram requires HTTPS)"}
	}
	return nil
}

type createStoryContainerRequest struct {
	ImageURL  string `json:"image_url"`
	MediaType string `json:"media_type"`
}

func (a *createInstagramStoryAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createInstagramStoryParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Step 1: Create media container with media_type=STORIES.
	containerBody := createStoryContainerRequest{
		ImageURL:  params.ImageURL,
		MediaType: "STORIES",
	}
	var containerResp createContainerResponse
	containerURL := fmt.Sprintf("%s/%s/media", a.conn.baseURL, params.InstagramAccountID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, containerURL, containerBody, &containerResp); err != nil {
		return nil, err
	}

	// Step 2: Poll for container readiness.
	if err := a.waitForContainer(ctx, req.Credentials, containerResp.ID); err != nil {
		return nil, err
	}

	// Step 3: Publish the container.
	publishBody := publishRequest{CreationID: containerResp.ID}
	var pubResp publishResponse
	publishURL := fmt.Sprintf("%s/%s/media_publish", a.conn.baseURL, params.InstagramAccountID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, publishURL, publishBody, &pubResp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":           pubResp.ID,
		"container_id": containerResp.ID,
	})
}

// waitForContainer polls the container status until it reaches FINISHED or an error state.
func (a *createInstagramStoryAction) waitForContainer(ctx context.Context, creds connectors.Credentials, containerID string) error {
	statusURL := fmt.Sprintf("%s/%s?fields=status_code", a.conn.baseURL, containerID)

	for i := 0; i < maxStoryContainerPolls; i++ {
		var status containerStatusResponse
		if err := a.conn.doGet(ctx, creds, statusURL, &status); err != nil {
			return err
		}

		switch strings.ToUpper(status.StatusCode) {
		case "FINISHED":
			return nil
		case "ERROR":
			return &connectors.ExternalError{
				Message: "Instagram story container processing failed",
			}
		case "EXPIRED":
			return &connectors.ExternalError{
				Message: "Instagram story container expired before publishing",
			}
		}

		select {
		case <-ctx.Done():
			return &connectors.TimeoutError{Message: "timed out waiting for Instagram story container to finish processing"}
		case <-time.After(storyContainerPollInterval):
		}
	}

	return &connectors.TimeoutError{
		Message: fmt.Sprintf("Instagram story container still processing after %d polls", maxStoryContainerPolls),
	}
}
