package meta

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
func (c *MetaConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "meta",
		Name:        "Meta",
		Description: "Meta integration for Instagram and Facebook Pages",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "meta.create_page_post",
				Name:        "Create Facebook Page Post",
				Description: "Post to a Facebook Page — publicly visible to page followers",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["page_id", "message"],
					"properties": {
						"page_id": {
							"type": "string",
							"description": "Facebook Page ID"
						},
						"message": {
							"type": "string",
							"description": "Post text content"
						},
						"link": {
							"type": "string",
							"description": "URL to share with the post"
						},
						"published": {
							"type": "boolean",
							"default": true,
							"description": "Whether to publish immediately (default true)"
						}
					}
				}`)),
			},
			{
				ActionType:  "meta.delete_page_post",
				Name:        "Delete Facebook Page Post",
				Description: "Delete a Facebook Page post — irreversible",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["post_id"],
					"properties": {
						"post_id": {
							"type": "string",
							"description": "Post ID to delete (format: page_id_post_id)"
						}
					}
				}`)),
			},
			{
				ActionType:  "meta.reply_page_comment",
				Name:        "Reply to Page Comment",
				Description: "Reply to a comment on a Facebook Page post",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["comment_id", "message"],
					"properties": {
						"comment_id": {
							"type": "string",
							"description": "Comment ID to reply to"
						},
						"message": {
							"type": "string",
							"description": "Reply text content"
						}
					}
				}`)),
			},
			{
				ActionType:  "meta.create_instagram_post",
				Name:        "Create Instagram Post",
				Description: "Publish a photo post to Instagram — publicly visible. Image must be hosted at a public URL.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["instagram_account_id", "image_url", "caption"],
					"properties": {
						"instagram_account_id": {
							"type": "string",
							"description": "Instagram Business/Creator account ID"
						},
						"image_url": {
							"type": "string",
							"description": "Public URL of the image to post"
						},
						"caption": {
							"type": "string",
							"maxLength": 2200,
							"description": "Post caption (max 2,200 characters)"
						},
						"hashtags": {
							"type": "string",
							"description": "Hashtags to append to caption (e.g. '#travel #photo')"
						}
					}
				}`)),
			},
			{
				ActionType:  "meta.get_instagram_insights",
				Name:        "Get Instagram Insights",
				Description: "Get account-level insights for an Instagram Business account",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["instagram_account_id"],
					"properties": {
						"instagram_account_id": {
							"type": "string",
							"description": "Instagram Business/Creator account ID"
						},
						"metric": {
							"type": "string",
							"enum": ["impressions", "reach", "profile_views"],
							"default": "impressions",
							"description": "Metric to retrieve (default: impressions)"
						},
						"period": {
							"type": "string",
							"enum": ["day", "week", "days_28"],
							"default": "day",
							"description": "Time period for the metric (default: day)"
						}
					}
				}`)),
			},
			{
				ActionType:  "meta.list_page_posts",
				Name:        "List Page Posts",
				Description: "List recent posts on a Facebook Page with engagement metrics",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["page_id"],
					"properties": {
						"page_id": {
							"type": "string",
							"description": "Facebook Page ID"
						},
						"limit": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 100,
							"description": "Maximum number of posts to return (1-100, default 10)"
						},
						"since": {
							"type": "integer",
							"description": "Unix timestamp — only return posts after this time"
						},
						"until": {
							"type": "integer",
							"description": "Unix timestamp — only return posts before this time"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "meta",
				AuthType:      "oauth2",
				OAuthProvider: "meta",
				OAuthScopes: []string{
					"pages_manage_posts",
					"pages_read_engagement",
					"pages_read_user_content",
					"instagram_basic",
					"instagram_content_publish",
					"instagram_manage_insights",
				},
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_meta_create_page_post",
				ActionType:  "meta.create_page_post",
				Name:        "Post to any Facebook Page",
				Description: "Agent can create posts on any Facebook Page.",
				Parameters:  json.RawMessage(`{"page_id":"*","message":"*","link":"*","published":"*"}`),
			},
			{
				ID:          "tpl_meta_create_page_post_specific",
				ActionType:  "meta.create_page_post",
				Name:        "Post to specific Facebook Page",
				Description: "Locks the page; agent chooses the message content.",
				Parameters:  json.RawMessage(`{"page_id":"PAGE_ID","message":"*","link":"*","published":"*"}`),
			},
			{
				ID:          "tpl_meta_delete_page_post",
				ActionType:  "meta.delete_page_post",
				Name:        "Delete Facebook Page posts",
				Description: "Agent can delete posts on Facebook Pages.",
				Parameters:  json.RawMessage(`{"post_id":"*"}`),
			},
			{
				ID:          "tpl_meta_reply_page_comment",
				ActionType:  "meta.reply_page_comment",
				Name:        "Reply to page comments",
				Description: "Agent can reply to comments on Facebook Page posts.",
				Parameters:  json.RawMessage(`{"comment_id":"*","message":"*"}`),
			},
			{
				ID:          "tpl_meta_create_instagram_post",
				ActionType:  "meta.create_instagram_post",
				Name:        "Post to Instagram",
				Description: "Agent can publish photo posts to Instagram.",
				Parameters:  json.RawMessage(`{"instagram_account_id":"*","image_url":"*","caption":"*","hashtags":"*"}`),
			},
			{
				ID:          "tpl_meta_create_instagram_post_specific",
				ActionType:  "meta.create_instagram_post",
				Name:        "Post to specific Instagram account",
				Description: "Locks the Instagram account; agent chooses image and caption.",
				Parameters:  json.RawMessage(`{"instagram_account_id":"IG_ACCOUNT_ID","image_url":"*","caption":"*","hashtags":"*"}`),
			},
			{
				ID:          "tpl_meta_get_instagram_insights",
				ActionType:  "meta.get_instagram_insights",
				Name:        "View Instagram insights",
				Description: "Agent can view insights for any Instagram Business account.",
				Parameters:  json.RawMessage(`{"instagram_account_id":"*","metric":"*","period":"*"}`),
			},
			{
				ID:          "tpl_meta_list_page_posts",
				ActionType:  "meta.list_page_posts",
				Name:        "List Facebook Page posts",
				Description: "Agent can list posts on any Facebook Page.",
				Parameters:  json.RawMessage(`{"page_id":"*","limit":"*","since":"*","until":"*"}`),
			},
			{
				ID:          "tpl_meta_list_page_posts_specific",
				ActionType:  "meta.list_page_posts",
				Name:        "List posts from specific page",
				Description: "Locks the page; agent can filter by time range.",
				Parameters:  json.RawMessage(`{"page_id":"PAGE_ID","limit":"*","since":"*","until":"*"}`),
			},
		},
	}
}
