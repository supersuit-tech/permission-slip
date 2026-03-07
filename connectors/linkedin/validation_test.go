package linkedin

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestValidatePostURN_Invalid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		urn  string
	}{
		{"no_prefix", "share:123456"},
		{"wrong_separator", "urn-li-share-123456"},
		{"missing_id", "urn:li:share:"},
		{"non_numeric_id", "urn:li:share:abc"},
		{"path_traversal", "urn:li:share:123/../../admin"},
		{"spaces", "urn:li:share:123 456"},
		{"empty_type", "urn:li::123456"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validatePostURN(tc.urn)
			if err == nil {
				t.Errorf("expected error for URN %q", tc.urn)
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got: %T", err)
			}
		})
	}
}

func TestValidatePostURN_Valid(t *testing.T) {
	t.Parallel()

	cases := []string{
		"urn:li:share:123456",
		"urn:li:ugcPost:789",
		"urn:li:activity:999999",
	}

	for _, urn := range cases {
		t.Run(urn, func(t *testing.T) {
			t.Parallel()
			if err := validatePostURN(urn); err != nil {
				t.Errorf("unexpected error for URN %q: %v", urn, err)
			}
		})
	}
}

func TestValidateOrganizationID_Invalid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		id   string
	}{
		{"letters", "abc"},
		{"mixed", "123abc"},
		{"special_chars", "12-34"},
		{"spaces", "12 34"},
		{"path_traversal", "123/../../admin"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateOrganizationID(tc.id)
			if err == nil {
				t.Errorf("expected error for org ID %q", tc.id)
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got: %T", err)
			}
		})
	}
}

func TestValidateOrganizationID_Valid(t *testing.T) {
	t.Parallel()

	for _, id := range []string{"1", "12345", "999999999"} {
		t.Run(id, func(t *testing.T) {
			t.Parallel()
			if err := validateOrganizationID(id); err != nil {
				t.Errorf("unexpected error for org ID %q: %v", id, err)
			}
		})
	}
}

func TestValidateArticleURL_UnsafeScheme(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		url  string
	}{
		{"javascript", "javascript:alert(1)"},
		{"data", "data:text/html,<script>alert(1)</script>"},
		{"ftp", "ftp://example.com/file"},
		{"file", "file:///etc/passwd"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateArticleURL(tc.url)
			if err == nil {
				t.Errorf("expected error for URL %q", tc.url)
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got: %T", err)
			}
		})
	}
}

func TestValidateArticleURL_Valid(t *testing.T) {
	t.Parallel()

	cases := []string{
		"https://example.com/article",
		"http://example.com/article",
		"https://blog.example.com/2024/post?utm_source=linkedin",
	}

	for _, u := range cases {
		t.Run(u, func(t *testing.T) {
			t.Parallel()
			if err := validateArticleURL(u); err != nil {
				t.Errorf("unexpected error for URL %q: %v", u, err)
			}
		})
	}
}

func TestValidateArticleURL_Empty(t *testing.T) {
	t.Parallel()
	if err := validateArticleURL(""); err != nil {
		t.Errorf("expected no error for empty URL, got: %v", err)
	}
}

func TestGetPostAnalytics_InvalidURNFormat(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getPostAnalyticsAction{conn: conn}

	params, _ := json.Marshal(getPostAnalyticsParams{PostURN: "not-a-valid-urn"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.get_post_analytics",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid URN format")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestDeletePost_InvalidURNFormat(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deletePostAction{conn: conn}

	params, _ := json.Marshal(deletePostParams{PostURN: "not-a-valid-urn"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.delete_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid URN format")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddComment_InvalidURNFormat(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addCommentAction{conn: conn}

	params, _ := json.Marshal(addCommentParams{PostURN: "not-a-valid-urn", Text: "Hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.add_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid URN format")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateCompanyPost_NonNumericOrgID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCompanyPostAction{conn: conn}

	params, _ := json.Marshal(createCompanyPostParams{
		OrganizationID: "abc-not-numeric",
		Text:           "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_company_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for non-numeric organization_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePost_JavascriptSchemeURL(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPostAction{conn: conn}

	params, _ := json.Marshal(createPostParams{
		Text:       "Check this out",
		ArticleURL: "javascript:alert(document.cookie)",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for javascript: scheme URL")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
