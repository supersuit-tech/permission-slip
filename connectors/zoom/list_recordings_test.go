package zoom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListRecordings_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/users/me/recordings" {
			t.Errorf("expected path /users/me/recordings, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("from") != "2024-01-01" {
			t.Errorf("expected from=2024-01-01, got %s", r.URL.Query().Get("from"))
		}
		if r.URL.Query().Get("to") != "2024-01-31" {
			t.Errorf("expected to=2024-01-31, got %s", r.URL.Query().Get("to"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(zoomRecordingsResponse{
			Meetings: []zoomRecordingMeeting{
				{
					ID:        123456,
					UUID:      "uuid-rec-1",
					Topic:     "Team Standup",
					StartTime: "2024-01-15T09:00:00Z",
					Duration:  30,
					TotalSize: 1024000,
					RecordingFiles: []zoomRecordingFile{
						{
							ID:            "file-1",
							FileType:      "MP4",
							FileSize:      512000,
							DownloadURL:   "https://zoom.us/rec/download/file-1",
							PlayURL:       "https://zoom.us/rec/play/file-1",
							RecordingType: "shared_screen_with_speaker_view",
							Status:        "completed",
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listRecordingsAction{conn: conn}

	params, _ := json.Marshal(listRecordingsParams{
		From: "2024-01-01",
		To:   "2024-01-31",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.list_recordings",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Recordings []recordingItem `json:"recordings"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Recordings) != 1 {
		t.Fatalf("expected 1 recording, got %d", len(data.Recordings))
	}
	if data.Recordings[0].Topic != "Team Standup" {
		t.Errorf("expected topic 'Team Standup', got %q", data.Recordings[0].Topic)
	}
	if len(data.Recordings[0].RecordingFiles) != 1 {
		t.Fatalf("expected 1 recording file, got %d", len(data.Recordings[0].RecordingFiles))
	}
	if data.Recordings[0].RecordingFiles[0].FileType != "MP4" {
		t.Errorf("expected file type 'MP4', got %q", data.Recordings[0].RecordingFiles[0].FileType)
	}
}

func TestListRecordings_MissingFrom(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listRecordingsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"to": "2024-01-31"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.list_recordings",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing from")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListRecordings_MissingTo(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listRecordingsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"from": "2024-01-01"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.list_recordings",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing to")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListRecordings_InvalidDateFormat(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listRecordingsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"from": "01/01/2024", "to": "2024-01-31"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.list_recordings",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid date format")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListRecordings_FromAfterTo(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listRecordingsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"from": "2024-02-01", "to": "2024-01-01"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.list_recordings",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for from after to")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListRecordings_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listRecordingsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.list_recordings",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
