package jira

import "strings"

// plainTextToADF wraps plain text in minimal Atlassian Document Format.
// Newlines are converted to separate paragraphs for readability.
func plainTextToADF(text string) map[string]interface{} {
	lines := strings.Split(text, "\n")
	var paragraphs []interface{}

	for _, line := range lines {
		paragraphs = append(paragraphs, map[string]interface{}{
			"type": "paragraph",
			"content": []interface{}{
				map[string]interface{}{
					"type": "text",
					"text": line,
				},
			},
		})
	}

	return map[string]interface{}{
		"type":    "doc",
		"version": 1,
		"content": paragraphs,
	}
}
