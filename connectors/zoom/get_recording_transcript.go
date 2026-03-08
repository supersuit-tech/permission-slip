package zoom

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getRecordingTranscriptAction implements connectors.Action for zoom.get_recording_transcript.
// It retrieves the transcript for a recorded meeting by:
// 1. Getting the list of recording files for the meeting.
// 2. Finding the VTT transcript file.
// 3. Downloading and returning its content.
type getRecordingTranscriptAction struct {
	conn *ZoomConnector
}

type getRecordingTranscriptParams struct {
	MeetingID string `json:"meeting_id"`
}

func (p *getRecordingTranscriptParams) validate() error {
	p.MeetingID = strings.TrimSpace(p.MeetingID)
	if p.MeetingID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: meeting_id"}
	}
	return nil
}

// zoomMeetingRecordingsResponse is the response from GET /meetings/{meeting_id}/recordings.
type zoomMeetingRecordingsResponse struct {
	ID             string                    `json:"id"`
	UUID           string                    `json:"uuid"`
	Topic          string                    `json:"topic"`
	StartTime      string                    `json:"start_time"`
	Duration       int                       `json:"duration"`
	RecordingFiles []zoomTranscriptFile      `json:"recording_files"`
}

type zoomTranscriptFile struct {
	ID             string `json:"id"`
	FileType       string `json:"file_type"`
	FileSize       int64  `json:"file_size"`
	DownloadURL    string `json:"download_url"`
	RecordingType  string `json:"recording_type"`
	Status         string `json:"status"`
}

func (a *getRecordingTranscriptAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getRecordingTranscriptParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Step 1: Get the list of recording files.
	var recordings zoomMeetingRecordingsResponse
	recordingsURL := fmt.Sprintf("%s/meetings/%s/recordings", a.conn.baseURL, params.MeetingID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, recordingsURL, nil, &recordings); err != nil {
		return nil, err
	}

	// Step 2: Find the transcript (VTT) file.
	var transcriptFile *zoomTranscriptFile
	for i := range recordings.RecordingFiles {
		f := &recordings.RecordingFiles[i]
		if strings.EqualFold(f.FileType, "TRANSCRIPT") {
			transcriptFile = f
			break
		}
	}

	if transcriptFile == nil {
		return connectors.JSONResult(map[string]interface{}{
			"meeting_id":  params.MeetingID,
			"topic":       recordings.Topic,
			"start_time":  recordings.StartTime,
			"transcript":  nil,
			"message":     "No transcript file found for this recording. Transcripts are only available when automatic transcription was enabled.",
		})
	}

	// Step 3: Download the transcript content using the access token.
	token, ok := req.Credentials.Get(credKeyAccessToken)
	if !ok || token == "" {
		return nil, &connectors.ValidationError{Message: "access_token credential is missing or empty"}
	}

	downloadURL := transcriptFile.DownloadURL + "?access_token=" + token
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("creating transcript download request: %v", err)}
	}

	resp, err := a.conn.client.Do(httpReq)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: "Zoom transcript download timed out"}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("downloading transcript: %v", err)}
	}
	defer resp.Body.Close()

	transcriptBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("reading transcript content: %v", err)}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("Zoom transcript download failed with status %d", resp.StatusCode),
		}
	}

	return connectors.JSONResult(map[string]interface{}{
		"meeting_id":      params.MeetingID,
		"topic":           recordings.Topic,
		"start_time":      recordings.StartTime,
		"duration":        recordings.Duration,
		"transcript":      string(transcriptBytes),
		"transcript_size": len(transcriptBytes),
	})
}
