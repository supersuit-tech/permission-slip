package context

import "testing"

func TestExtractFiles(t *testing.T) {
	t.Parallel()
	msg := slackMessage{
		Files: []slackFile{
			{Name: "a.png", Size: 12},
			{Name: "", Size: 3},
		},
	}
	fs := ExtractFiles(msg)
	if len(fs) != 2 || fs[0].Filename != "a.png" || fs[1].Filename != "file" {
		t.Fatalf("ExtractFiles = %#v", fs)
	}
}
