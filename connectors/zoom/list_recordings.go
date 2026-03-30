package zoom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listRecordingsAction implements connectors.Action for zoom.list_recordings.
// It lists cloud recordings via GET /users/me/recordings.
type listRecordingsAction struct {
	conn *ZoomConnector
}

type listRecordingsParams struct {
	From     string `json:"from"`
	To       string `json:"to"`
	PageSize int    `json:"page_size"`
}

// datePattern matches YYYY-MM-DD format.
var datePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func (p *listRecordingsParams) validate() error {
	if p.From == "" {
		return &connectors.ValidationError{Message: "missing required parameter: from"}
	}
	if p.To == "" {
		return &connectors.ValidationError{Message: "missing required parameter: to"}
	}
	if !datePattern.MatchString(p.From) {
		return &connectors.ValidationError{Message: "from must be in YYYY-MM-DD format"}
	}
	if !datePattern.MatchString(p.To) {
		return &connectors.ValidationError{Message: "to must be in YYYY-MM-DD format"}
	}
	if p.From > p.To {
		return &connectors.ValidationError{Message: "from date must be before or equal to to date"}
	}
	return nil
}

func (p *listRecordingsParams) normalize() {
	if p.PageSize <= 0 {
		p.PageSize = 30
	}
	if p.PageSize > 300 {
		p.PageSize = 300
	}
}

// zoomRecordingsResponse is the Zoom API response from GET /users/me/recordings.
type zoomRecordingsResponse struct {
	Meetings []zoomRecordingMeeting `json:"meetings"`
}

type zoomRecordingMeeting struct {
	ID             int64                `json:"id"`
	UUID           string               `json:"uuid"`
	Topic          string               `json:"topic"`
	StartTime      string               `json:"start_time"`
	Duration       int                  `json:"duration"`
	TotalSize      int64                `json:"total_size"`
	RecordingFiles []zoomRecordingFile  `json:"recording_files"`
}

type zoomRecordingFile struct {
	ID             string `json:"id"`
	FileType       string `json:"file_type"`
	FileSize       int64  `json:"file_size"`
	DownloadURL    string `json:"download_url"`
	PlayURL        string `json:"play_url"`
	RecordingStart string `json:"recording_start"`
	RecordingEnd   string `json:"recording_end"`
	RecordingType  string `json:"recording_type"`
	Status         string `json:"status"`
}

type recordingItem struct {
	MeetingID      int64               `json:"meeting_id"`
	UUID           string              `json:"uuid"`
	Topic          string              `json:"topic"`
	StartTime      string              `json:"start_time"`
	Duration       int                 `json:"duration"`
	TotalSize      int64               `json:"total_size"`
	RecordingFiles []recordingFileItem `json:"recording_files,omitempty"`
}

type recordingFileItem struct {
	ID            string `json:"id"`
	FileType      string `json:"file_type"`
	FileSize      int64  `json:"file_size"`
	DownloadURL   string `json:"download_url"`
	PlayURL       string `json:"play_url,omitempty"`
	RecordingType string `json:"recording_type"`
	Status        string `json:"status"`
}

func (a *listRecordingsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listRecordingsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	q := url.Values{}
	q.Set("from", params.From)
	q.Set("to", params.To)
	q.Set("page_size", strconv.Itoa(params.PageSize))

	var resp zoomRecordingsResponse
	recordingsURL := a.conn.baseURL + "/users/me/recordings?" + q.Encode()
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, recordingsURL, nil, &resp); err != nil {
		return nil, err
	}

	recordings := make([]recordingItem, 0, len(resp.Meetings))
	for _, m := range resp.Meetings {
		item := recordingItem{
			MeetingID: m.ID,
			UUID:      m.UUID,
			Topic:     m.Topic,
			StartTime: m.StartTime,
			Duration:  m.Duration,
			TotalSize: m.TotalSize,
		}
		files := make([]recordingFileItem, 0, len(m.RecordingFiles))
		for _, f := range m.RecordingFiles {
			files = append(files, recordingFileItem{
				ID:            f.ID,
				FileType:      f.FileType,
				FileSize:      f.FileSize,
				DownloadURL:   f.DownloadURL,
				PlayURL:       f.PlayURL,
				RecordingType: f.RecordingType,
				Status:        f.Status,
			})
		}
		if len(files) > 0 {
			item.RecordingFiles = files
		}
		recordings = append(recordings, item)
	}

	return connectors.JSONResult(map[string]any{
		"total_recordings": len(recordings),
		"recordings":       recordings,
	})
}
