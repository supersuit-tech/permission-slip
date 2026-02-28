package main

import "testing"

func TestRepoName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{"https", "https://github.com/acme/ps-jira-connector", "ps-jira-connector"},
		{"https with .git", "https://github.com/acme/ps-jira-connector.git", "ps-jira-connector"},
		{"https trailing slash", "https://github.com/acme/ps-jira-connector/", "ps-jira-connector"},
		{"https .git and slash", "https://github.com/acme/ps-jira-connector.git/", "ps-jira-connector"},
		{"ssh colon style", "git@github.com:acme/ps-jira-connector.git", "ps-jira-connector"},
		{"ssh colon no .git", "git@github.com:acme/ps-jira-connector", "ps-jira-connector"},
		{"bare name", "my-connector", "my-connector"},
		{"bare name with .git", "my-connector.git", "my-connector"},
		{"deeply nested path", "https://gitlab.com/org/group/subgroup/connector.git", "connector"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := repoName(tt.url)
			if got != tt.want {
				t.Errorf("repoName(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
