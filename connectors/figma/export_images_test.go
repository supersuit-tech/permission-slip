package figma

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestExportImages_Success(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/images/abc123" {
			t.Errorf("expected path /images/abc123, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("ids"); got != "1:2" {
			t.Errorf("expected ids=1:2, got %q", got)
		}
		if got := r.URL.Query().Get("format"); got != "png" {
			t.Errorf("expected format=png, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"images": map[string]any{
				"1:2": "https://figma-alpha-api.s3.us-west-2.amazonaws.com/images/test.png",
			},
		})
	})

	action := &exportImagesAction{conn: conn}
	params, _ := json.Marshal(exportImagesParams{
		FileKey: "abc123",
		NodeIDs: "1:2",
		Format:  "png",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.export_images",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	images, ok := data["images"].(map[string]any)
	if !ok {
		t.Fatal("expected images map in response")
	}
	if images["1:2"] == nil {
		t.Error("expected image URL for node 1:2")
	}
}

func TestExportImages_WithScale(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("scale"); got != "2" {
			t.Errorf("expected scale=2, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"images": map[string]any{}})
	})

	action := &exportImagesAction{conn: conn}
	scale := 2.0
	params, _ := json.Marshal(exportImagesParams{
		FileKey: "abc123",
		NodeIDs: "1:2",
		Format:  "png",
		Scale:   &scale,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.export_images",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExportImages_InvalidFormat(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &exportImagesAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"file_key": "abc123",
		"node_ids": "1:2",
		"format":   "gif",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.export_images",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestExportImages_InvalidScale(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &exportImagesAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"file_key": "abc123",
		"node_ids": "1:2",
		"format":   "png",
		"scale":    5.0,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.export_images",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for scale > 4")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestExportImages_ZeroScale(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &exportImagesAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"file_key": "abc123",
		"node_ids": "1:2",
		"format":   "png",
		"scale":    0,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.export_images",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for scale=0 (below minimum 0.01)")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestExportImages_MissingNodeIDs(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &exportImagesAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"file_key": "abc123",
		"format":   "png",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.export_images",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing node_ids")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestExportImages_MissingFormat(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &exportImagesAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"file_key": "abc123",
		"node_ids": "1:2",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.export_images",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing format")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
