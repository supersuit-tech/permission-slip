package connectors

import "testing"

func TestInferOperationType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		actionType string
		want       OperationType
	}{
		{"github.list_repos", OperationRead},
		{"github.get_repo", OperationRead},
		{"github.search_code", OperationRead},
		{"github.create_issue", OperationWrite},
		{"github.merge_pr", OperationWrite},
		{"github.close_issue", OperationDelete},
		{"expedia.price_check", OperationRead},
		{"google.read_email", OperationRead},
		{"google.archive_email", OperationDelete},
		{"docusign.void_envelope", OperationDelete},
		{"no_dot", OperationWrite},
	}
	for _, tc := range cases {
		got := InferOperationType(tc.actionType)
		if got != tc.want {
			t.Errorf("InferOperationType(%q) = %q, want %q", tc.actionType, got, tc.want)
		}
	}
}
