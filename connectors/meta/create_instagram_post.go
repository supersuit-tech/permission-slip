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
	// maxCaptionLength is Instagram's maximum caption length.
	maxCaptionLength = 2200

	// containerPollInterval is how long to wait between polling for
	// container status during Instagram media publishing.
	containerPollInterval = 2 * time.Second

	// maxContainerPolls is the maximum number of times to poll for
	// container readiness before giving up.
	maxContainerPolls = 15
)

// createInstagramPostAction implements connectors.Action for meta.create_instagram_post.
// Instagram Content Publishing is a two-step process:
// 1. POST /{ig_account_id}/media — create a media container
// 2. POST /{ig_account_id}/media_publish — publish the container
type createInstagramPostAction struct {
	conn *MetaConnector
}

type createInstagramPostParams struct {
	InstagramAccountID string `json:"instagram_account_id"`
	ImageURL           string `json:"image_url"`
	Caption            string `json:"caption"`
	Hashtags           string `json:"hashtags,omitempty"`
}

func (p *createInstagramPostParams) validate() error {
	if p.InstagramAccountID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: instagram_account_id"}
	}
	if !isValidGraphID(p.InstagramAccountID) {
		return &connectors.ValidationError{Message: "instagram_account_id contains invalid characters"}
	}
	if p.ImageURL == "" {
		return &connectors.ValidationError{Message: "missing required parameter: image_url"}
	}
	// Instagram requires HTTPS URLs for media container creation.
	u, err := url.Parse(p.ImageURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return &connectors.ValidationError{Message: "image_url must be a valid HTTPS URL (Instagram requires HTTPS)"}
	}
	if p.Caption == "" {
		return &connectors.ValidationError{Message: "missing required parameter: caption"}
	}
	// Build the full caption to check length.
	caption := p.Caption
	if p.Hashtags != "" {
		caption = caption + " " + p.Hashtags
	}
	if len(caption) > maxCaptionLength {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("caption exceeds maximum length of %d characters (got %d)", maxCaptionLength, len(caption)),
		}
	}
	return nil
}

type createContainerRequest struct {
	ImageURL string `json:"image_url"`
	Caption  string `json:"caption"`
}

type createContainerResponse struct {
	ID string `json:"id"`
}

type containerStatusResponse struct {
	StatusCode string `json:"status_code"`
}

type publishRequest struct {
	CreationID string `json:"creation_id"`
}

type publishResponse struct {
	ID string `json:"id"`
}

func (a *createInstagramPostAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createInstagramPostParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build caption with optional hashtags.
	caption := params.Caption
	if params.Hashtags != "" {
		caption = caption + " " + params.Hashtags
	}

	// Step 1: Create media container.
	containerBody := createContainerRequest{
		ImageURL: params.ImageURL,
		Caption:  caption,
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

// waitForContainer polls the container status until it reaches FINISHED
// or an error state. Instagram media processing is asynchronous — the
// container may not be ready for publishing immediately after creation.
func (a *createInstagramPostAction) waitForContainer(ctx context.Context, creds connectors.Credentials, containerID string) error {
	statusURL := fmt.Sprintf("%s/%s?fields=status_code", a.conn.baseURL, containerID)

	for i := 0; i < maxContainerPolls; i++ {
		var status containerStatusResponse
		if err := a.conn.doGet(ctx, creds, statusURL, &status); err != nil {
			return err
		}

		switch strings.ToUpper(status.StatusCode) {
		case "FINISHED":
			return nil
		case "ERROR":
			return &connectors.ExternalError{
				Message: "Instagram media container processing failed",
			}
		case "EXPIRED":
			return &connectors.ExternalError{
				Message: "Instagram media container expired before publishing",
			}
		}

		// Still processing — wait before next poll.
		select {
		case <-ctx.Done():
			return &connectors.TimeoutError{Message: "timed out waiting for Instagram media container to finish processing"}
		case <-time.After(containerPollInterval):
		}
	}

	return &connectors.TimeoutError{
		Message: fmt.Sprintf("Instagram media container still processing after %d polls", maxContainerPolls),
	}
}
