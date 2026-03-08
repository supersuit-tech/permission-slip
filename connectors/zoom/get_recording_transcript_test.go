package zoom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetRecordingTranscript_Success(t *testing.T) {
	t.Parallel()

	transcriptContent := "WEBVTT\n\n00:00:01.000 --> 00:00:05.000\nHello everyone, welcome to the meeting.\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/meetings/98765432100/recordings":
			json.NewEncoder(w).Encode(zoomMeetingRecordingsResponse{
				ID:        "98765432100",
				Topic:     "Team Standup",
				StartTime: "2024-01-15T09:00:00Z",
				Duration:  30,
				RecordingFiles: []zoomTranscriptFile{
					{
						ID:            "file-mp4",
						FileType:      "MP4",
						FileSize:      1024000,
						DownloadURL:   "http://" + r.Host + "/download/mp4",
						RecordingType: "shared_screen_with_speaker_view",
						Status:        "completed",
					},
					{
						ID:            "file-vtt",
						FileType:      "TRANSCRIPT",
						FileSize:      512,
						DownloadURL:   "http://" + r.Host + "/download/transcript",
						RecordingType: "audio_transcript",
						Status:        "completed",
					},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/download/transcript":
			w.Header().Set("Content-Type", "text/vtt")
			w.Write([]byte(transcriptContent))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getRecordingTranscriptAction{conn: conn}

	params, _ := json.Marshal(getRecordingTranscriptParams{MeetingID: "98765432100"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.get_recording_transcript",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["topic"] != "Team Standup" {
		t.Errorf("expected topic 'Team Standup', got %v", data["topic"])
	}
	transcript, ok := data["transcript"].(string)
	if !ok {
		t.Fatalf("expected transcript to be a string, got %T", data["transcript"])
	}
	if transcript != transcriptContent {
		t.Errorf("transcript content mismatch: got %q", transcript)
	}
}

func TestGetRecordingTranscript_NoTranscript(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(zoomMeetingRecordingsResponse{
			ID:        "98765432100",
			Topic:     "Team Standup",
			StartTime: "2024-01-15T09:00:00Z",
			RecordingFiles: []zoomTranscriptFile{
				{
					ID:       "file-mp4",
					FileType: "MP4",
					Status:   "completed",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getRecordingTranscriptAction{conn: conn}

	params, _ := json.Marshal(getRecordingTranscriptParams{MeetingID: "98765432100"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.get_recording_transcript",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["transcript"] != nil {
		t.Errorf("expected transcript to be nil when no transcript file exists")
	}
	if _, ok := data["message"]; !ok {
		t.Error("expected 'message' field when no transcript found")
	}
}

func TestGetRecordingTranscript_MissingMeetingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getRecordingTranscriptAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zoom.get_recording_transcript",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}
