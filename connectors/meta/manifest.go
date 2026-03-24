package meta

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
//go:embed logo.svg
var logoSVG string

func (c *MetaConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "meta",
		Name:        "Meta",
		Description: "Meta integration for Instagram and Facebook Pages",
		LogoSVG:     logoSVG,
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
							"type": "string",
							"format": "date-time",
							"description": "Only return posts after this time. Unix seconds, epoch milliseconds, or RFC 3339 (e.g. 2024-03-01T00:00:00Z).",
							"x-ui": {
								"widget": "datetime",
								"datetime_range_pair": "until",
								"datetime_range_role": "lower"
							}
						},
						"until": {
							"type": "string",
							"format": "date-time",
							"description": "Only return posts before this time. Unix seconds, epoch milliseconds, or RFC 3339.",
							"x-ui": {
								"widget": "datetime",
								"datetime_range_pair": "since",
								"datetime_range_role": "upper"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "meta.create_instagram_story",
				Name:        "Create Instagram Story",
				Description: "Publish a story to Instagram — higher engagement than feed posts. Image must be hosted at a public HTTPS URL.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["instagram_account_id", "image_url"],
					"properties": {
						"instagram_account_id": {
							"type": "string",
							"description": "Instagram Business/Creator account ID"
						},
						"image_url": {
							"type": "string",
							"description": "Public HTTPS URL of the image to publish as a story"
						}
					}
				}`)),
			},
			{
				ActionType:  "meta.get_page_insights",
				Name:        "Get Facebook Page Insights",
				Description: "Get analytics metrics for a Facebook Page (impressions, reach, engagement, fans)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["page_id"],
					"properties": {
						"page_id": {
							"type": "string",
							"description": "Facebook Page ID"
						},
						"metric": {
							"type": "string",
							"enum": ["page_impressions", "page_impressions_unique", "page_engaged_users", "page_post_engagements", "page_fan_adds", "page_fan_removes", "page_views_total", "page_reach"],
							"default": "page_impressions",
							"description": "Metric to retrieve (default: page_impressions)"
						},
						"period": {
							"type": "string",
							"enum": ["day", "week", "days_28", "month"],
							"default": "day",
							"description": "Time period for the metric (default: day)"
						},
						"since": {
							"type": "string",
							"format": "date-time",
							"description": "Only return data after this time. Unix seconds, epoch milliseconds, or RFC 3339.",
							"x-ui": {
								"widget": "datetime",
								"datetime_range_pair": "until",
								"datetime_range_role": "lower"
							}
						},
						"until": {
							"type": "string",
							"format": "date-time",
							"description": "Only return data before this time. Unix seconds, epoch milliseconds, or RFC 3339.",
							"x-ui": {
								"widget": "datetime",
								"datetime_range_pair": "since",
								"datetime_range_role": "upper"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "meta.list_instagram_posts",
				Name:        "List Instagram Posts",
				Description: "List recent posts for an Instagram Business/Creator account with engagement metrics",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["instagram_account_id"],
					"properties": {
						"instagram_account_id": {
							"type": "string",
							"description": "Instagram Business/Creator account ID"
						},
						"limit": {
							"type": "integer",
							"default": 10,
							"minimum": 1,
							"maximum": 100,
							"description": "Maximum number of posts to return (1-100, default 10)"
						}
					}
				}`)),
			},
			{
				ActionType:  "meta.reply_instagram_comment",
				Name:        "Reply to Instagram Comment",
				Description: "Reply to a comment on an Instagram post",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["comment_id", "message"],
					"properties": {
						"comment_id": {
							"type": "string",
							"description": "Instagram comment ID to reply to"
						},
						"message": {
							"type": "string",
							"maxLength": 2200,
							"description": "Reply text (max 2,200 characters)"
						}
					}
				}`)),
			},
			{
				ActionType:  "meta.create_ad_campaign",
				Name:        "Create Ad Campaign",
				Description: "Create a Facebook/Instagram ad campaign — starts paused by default",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ad_account_id", "name", "objective"],
					"properties": {
						"ad_account_id": {
							"type": "string",
							"description": "Ad account ID (without 'act_' prefix)"
						},
						"name": {
							"type": "string",
							"description": "Campaign name"
						},
						"objective": {
							"type": "string",
							"enum": ["OUTCOME_AWARENESS", "OUTCOME_ENGAGEMENT", "OUTCOME_LEADS", "OUTCOME_SALES", "OUTCOME_TRAFFIC", "OUTCOME_APP_PROMOTION"],
							"description": "Campaign objective"
						},
						"status": {
							"type": "string",
							"enum": ["ACTIVE", "PAUSED", "ARCHIVED"],
							"default": "PAUSED",
							"description": "Campaign status (default: PAUSED)"
						},
						"budget_type": {
							"type": "string",
							"enum": ["DAILY", "LIFETIME"],
							"description": "Budget type — use with daily_budget or lifetime_budget respectively"
						},
						"daily_budget": {
							"type": "integer",
							"description": "Daily budget in account currency's smallest unit (e.g. cents) — mutually exclusive with lifetime_budget"
						},
						"lifetime_budget": {
							"type": "integer",
							"description": "Lifetime budget in account currency's smallest unit — mutually exclusive with daily_budget"
						}
					}
				}`)),
			},
			{
				ActionType:  "meta.create_ad",
				Name:        "Create Ad",
				Description: "Create an ad within an existing ad set using an existing creative — starts paused by default",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ad_account_id", "name", "adset_id", "creative_id"],
					"properties": {
						"ad_account_id": {
							"type": "string",
							"description": "Ad account ID (without 'act_' prefix)"
						},
						"name": {
							"type": "string",
							"description": "Ad name"
						},
						"adset_id": {
							"type": "string",
							"description": "ID of the ad set this ad belongs to"
						},
						"creative_id": {
							"type": "string",
							"description": "ID of the ad creative to use"
						},
						"status": {
							"type": "string",
							"enum": ["ACTIVE", "PAUSED", "ARCHIVED"],
							"default": "PAUSED",
							"description": "Ad status (default: PAUSED)"
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
					"ads_management",
					"ads_read",
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
			{
				ID:          "tpl_meta_create_instagram_story",
				ActionType:  "meta.create_instagram_story",
				Name:        "Post Instagram stories",
				Description: "Agent can publish stories to any Instagram account.",
				Parameters:  json.RawMessage(`{"instagram_account_id":"*","image_url":"*"}`),
			},
			{
				ID:          "tpl_meta_get_page_insights",
				ActionType:  "meta.get_page_insights",
				Name:        "View Facebook Page analytics",
				Description: "Agent can read analytics for any Facebook Page.",
				Parameters:  json.RawMessage(`{"page_id":"*","metric":"*","period":"*","since":"*","until":"*"}`),
			},
			{
				ID:          "tpl_meta_list_instagram_posts",
				ActionType:  "meta.list_instagram_posts",
				Name:        "List Instagram posts",
				Description: "Agent can list posts from any Instagram account.",
				Parameters:  json.RawMessage(`{"instagram_account_id":"*","limit":"*"}`),
			},
			{
				ID:          "tpl_meta_reply_instagram_comment",
				ActionType:  "meta.reply_instagram_comment",
				Name:        "Reply to Instagram comments",
				Description: "Agent can reply to comments on Instagram posts.",
				Parameters:  json.RawMessage(`{"comment_id":"*","message":"*"}`),
			},
			{
				ID:          "tpl_meta_create_ad_campaign",
				ActionType:  "meta.create_ad_campaign",
				Name:        "Create ad campaigns",
				Description: "Agent can create Facebook/Instagram ad campaigns.",
				Parameters:  json.RawMessage(`{"ad_account_id":"*","name":"*","objective":"*","status":"*","budget_type":"*","daily_budget":"*","lifetime_budget":"*"}`),
			},
			{
				ID:          "tpl_meta_create_ad",
				ActionType:  "meta.create_ad",
				Name:        "Create ads",
				Description: "Agent can create ads within existing ad sets.",
				Parameters:  json.RawMessage(`{"ad_account_id":"*","name":"*","adset_id":"*","creative_id":"*","status":"*"}`),
			},
		},
	}
}
