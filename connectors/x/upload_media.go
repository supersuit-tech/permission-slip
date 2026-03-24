package x

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const uploadBaseURL = "https://upload.twitter.com/1.1"

// maxMediaDataBytes caps the base64-encoded media_data string length at ~27 MB,
// which covers the X API's 15 MB GIF limit plus encoding overhead (~33%).
// Requests above this limit are rejected early without touching the network.
const maxMediaDataBytes = 27 * 1024 * 1024

// maxAltTextBytes matches the X API's documented 1000-character alt-text limit.
const maxAltTextBytes = 1000

// uploadMediaAction implements connectors.Action for x.upload_media.
// It uploads media via POST https://upload.twitter.com/1.1/media/upload.json
// using the simple upload method (base64-encoded media data).
type uploadMediaAction struct {
	conn *XConnector
}

// uploadMediaParams are the parameters parsed from ActionRequest.Parameters.
type uploadMediaParams struct {
	// MediaData is the base64-encoded media content.
	MediaData     string `json:"media_data"`
	MediaCategory string `json:"media_category"`
	AltText       string `json:"alt_text"`
}

func (p *uploadMediaParams) validate() error {
	if p.MediaData == "" {
		return errMissingParam("media_data")
	}
	if len(p.MediaData) > maxMediaDataBytes {
		return errInvalidParam(fmt.Sprintf("media_data exceeds maximum allowed size (%d MB)", maxMediaDataBytes/(1024*1024)))
	}
	if p.MediaCategory != "" {
		valid := map[string]bool{
			"tweet_image": true,
			"tweet_gif":   true,
			"tweet_video": true,
		}
		if !valid[p.MediaCategory] {
			return errInvalidParam("media_category must be one of: tweet_image, tweet_gif, tweet_video")
		}
	}
	if len(p.AltText) > maxAltTextBytes {
		return errInvalidParam(fmt.Sprintf("alt_text exceeds maximum length of %d characters", maxAltTextBytes))
	}
	return nil
}

// Execute uploads media and returns the media_id for use in post_tweet.
func (a *uploadMediaAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params uploadMediaParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, errBadJSON(err)
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	token, ok := req.Credentials.Get(credKeyToken)
	if !ok || token == "" {
		return nil, errMissingParam("access_token")
	}

	// Build form-encoded body for the v1.1 simple upload endpoint.
	form := url.Values{}
	form.Set("media_data", params.MediaData)
	if params.MediaCategory != "" {
		form.Set("media_category", params.MediaCategory)
	}

	// Use the connector's uploadBaseURL if set (for tests), otherwise the default.
	uploadURL := a.conn.uploadBaseURL
	if uploadURL == "" {
		uploadURL = uploadBaseURL
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		uploadURL+"/media/upload.json",
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating upload request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.conn.client.Do(httpReq)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("media upload request timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("media upload request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading upload response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return nil, err
	}

	var uploadResp struct {
		MediaID    int64  `json:"media_id"`
		MediaIDStr string `json:"media_id_string"`
		Size       int    `json:"size"`
		ExpiresAfterSecs int `json:"expires_after_secs"`
		Image      *struct {
			ImageType string `json:"image_type"`
			W         int    `json:"w"`
			H         int    `json:"h"`
		} `json:"image"`
	}
	if err := json.Unmarshal(respBytes, &uploadResp); err != nil {
		return nil, &connectors.ExternalError{Message: "failed to decode media upload response"}
	}

	// If alt_text is provided, set it via the metadata endpoint.
	if params.AltText != "" && uploadResp.MediaIDStr != "" {
		_ = a.conn.setMediaAltText(ctx, req.Credentials, uploadResp.MediaIDStr, params.AltText)
		// Alt text failure is non-fatal — the media is already uploaded.
	}

	return connectors.JSONResult(uploadResp)
}

// setMediaAltText sets alt text on an uploaded media item via
// POST /1.1/media/metadata/create.json.
func (c *XConnector) setMediaAltText(ctx context.Context, creds connectors.Credentials, mediaIDStr, altText string) error {
	uploadURL := c.uploadBaseURL
	if uploadURL == "" {
		uploadURL = uploadBaseURL
	}

	body := map[string]any{
		"media_id":  mediaIDStr,
		"alt_text":  map[string]string{"text": altText},
	}
	return c.doUpload(ctx, creds, uploadURL+"/media/metadata/create.json", body)
}

// doUpload sends a JSON POST to an arbitrary URL using Bearer auth.
// Used for upload.twitter.com endpoints that don't share the v2 base URL.
func (c *XConnector) doUpload(ctx context.Context, creds connectors.Credentials, fullURL string, body any) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling upload body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating upload request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("upload request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.CanceledError{Message: "upload request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("upload request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading upload response: %v", err)}
	}

	return checkResponse(resp.StatusCode, resp.Header, respBytes)
}
