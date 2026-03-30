package x

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//go:embed logo.svg
var logoSVG string

func (c *XConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "x",
		Name:        "X (Twitter)",
		Description: "X/Twitter integration for social media management",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "x.post_tweet",
				Name:        "Post Tweet",
				Description: "Post a tweet, reply, or quote tweet",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["text"],
					"properties": {
						"text": {
							"type": "string",
							"maxLength": 280,
							"description": "Tweet text (max 280 characters)"
						},
						"reply_to_tweet_id": {
							"type": "string",
							"description": "Tweet ID to reply to"
						},
						"quote_tweet_id": {
							"type": "string",
							"description": "Tweet ID to quote"
						},
						"media_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Pre-uploaded media IDs to attach"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.delete_tweet",
				Name:        "Delete Tweet",
				Description: "Delete a tweet (irreversible)",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["tweet_id"],
					"properties": {
						"tweet_id": {
							"type": "string",
							"description": "ID of the tweet to delete"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.send_dm",
				Name:        "Send Direct Message",
				Description: "Send a direct message to a user",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["recipient_id", "text"],
					"properties": {
						"recipient_id": {
							"type": "string",
							"description": "User ID of the recipient"
						},
						"text": {
							"type": "string",
							"maxLength": 10000,
							"description": "Message text (max 10,000 characters)"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.get_user_tweets",
				Name:        "Get User Tweets",
				Description: "Get recent tweets from a specific user",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["user_id"],
					"properties": {
						"user_id": {
							"type": "string",
							"description": "User ID to get tweets from"
						},
						"max_results": {
							"type": "integer",
							"minimum": 1,
							"maximum": 100,
							"default": 10,
							"description": "Maximum number of results (1-100, default 10)"
						},
						"since_id": {
							"type": "string",
							"description": "Only return tweets after this tweet ID"
						},
						"until_id": {
							"type": "string",
							"description": "Only return tweets before this tweet ID"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.search_tweets",
				Name:        "Search Tweets",
				Description: "Search recent tweets (7-day window on Basic tier)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Search query (X search syntax)"
						},
						"max_results": {
							"type": "integer",
							"minimum": 10,
							"maximum": 100,
							"default": 10,
							"description": "Maximum number of results (10-100, default 10)"
						},
						"since_id": {
							"type": "string",
							"description": "Only return tweets after this tweet ID"
						},
						"sort_order": {
							"type": "string",
							"enum": ["recency", "relevancy"],
							"description": "Sort order for results"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.get_me",
				Name:        "Get My Profile",
				Description: "Get the authenticated user's profile info",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
			{
				ActionType:  "x.like_tweet",
				Name:        "Like Tweet",
				Description: "Like a tweet on behalf of the authenticated user",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["tweet_id"],
					"properties": {
						"user_id": {
							"type": "string",
							"description": "Authenticated user's ID. If omitted, resolved automatically via /users/me"
						},
						"tweet_id": {
							"type": "string",
							"description": "ID of the tweet to like"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.unlike_tweet",
				Name:        "Unlike Tweet",
				Description: "Remove a like from a tweet",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["tweet_id"],
					"properties": {
						"user_id": {
							"type": "string",
							"description": "Authenticated user's ID. If omitted, resolved automatically via /users/me"
						},
						"tweet_id": {
							"type": "string",
							"description": "ID of the tweet to unlike"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.retweet",
				Name:        "Retweet",
				Description: "Retweet a tweet on behalf of the authenticated user",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["tweet_id"],
					"properties": {
						"user_id": {
							"type": "string",
							"description": "Authenticated user's ID. If omitted, resolved automatically via /users/me"
						},
						"tweet_id": {
							"type": "string",
							"description": "ID of the tweet to retweet"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.unretweet",
				Name:        "Undo Retweet",
				Description: "Undo a retweet",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["tweet_id"],
					"properties": {
						"user_id": {
							"type": "string",
							"description": "Authenticated user's ID. If omitted, resolved automatically via /users/me"
						},
						"tweet_id": {
							"type": "string",
							"description": "ID of the tweet to unretweet"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.follow_user",
				Name:        "Follow User",
				Description: "Follow a user on behalf of the authenticated user",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["target_user_id"],
					"properties": {
						"user_id": {
							"type": "string",
							"description": "Authenticated user's ID. If omitted, resolved automatically via /users/me"
						},
						"target_user_id": {
							"type": "string",
							"description": "ID of the user to follow"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.unfollow_user",
				Name:        "Unfollow User",
				Description: "Unfollow a user",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["target_user_id"],
					"properties": {
						"user_id": {
							"type": "string",
							"description": "Authenticated user's ID. If omitted, resolved automatically via /users/me"
						},
						"target_user_id": {
							"type": "string",
							"description": "ID of the user to unfollow"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.get_followers",
				Name:        "Get Followers",
				Description: "Get a user's followers list (defaults to the authenticated user)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"user_id": {
							"type": "string",
							"description": "User ID to get followers for. If omitted, defaults to the authenticated user"
						},
						"max_results": {
							"type": "integer",
							"minimum": 1,
							"maximum": 1000,
							"default": 100,
							"description": "Maximum number of results (1-1000, default 100)"
						},
						"pagination_token": {
							"type": "string",
							"description": "Token for fetching the next page of results"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.get_following",
				Name:        "Get Following",
				Description: "Get the list of users that a user follows (defaults to the authenticated user)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"user_id": {
							"type": "string",
							"description": "User ID to get following list for. If omitted, defaults to the authenticated user"
						},
						"max_results": {
							"type": "integer",
							"minimum": 1,
							"maximum": 1000,
							"default": 100,
							"description": "Maximum number of results (1-1000, default 100)"
						},
						"pagination_token": {
							"type": "string",
							"description": "Token for fetching the next page of results"
						}
					}
				}`)),
			},
			{
				ActionType:  "x.upload_media",
				Name:        "Upload Media",
				Description: "Upload media (image/GIF/video) for use in tweets",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["media_data"],
					"properties": {
						"media_data": {
							"type": "string",
							"description": "Base64-encoded media content"
						},
						"media_category": {
							"type": "string",
							"enum": ["tweet_image", "tweet_gif", "tweet_video"],
							"description": "Media category (default: tweet_image)"
						},
						"alt_text": {
							"type": "string",
							"maxLength": 1000,
							"description": "Alt text for accessibility (images only)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "x",
				AuthType:      "oauth2",
				OAuthProvider: "x",
				OAuthScopes:   []string{"tweet.read", "tweet.write", "users.read", "dm.read", "dm.write", "offline.access", "like.read", "like.write", "follows.read", "follows.write"},
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_x_post_tweet",
				ActionType:  "x.post_tweet",
				Name:        "Post tweets on my behalf",
				Description: "Agent can post tweets with any text content.",
				Parameters:  json.RawMessage(`{"text":"*"}`),
			},
			{
				ID:          "tpl_x_post_reply",
				ActionType:  "x.post_tweet",
				Name:        "Reply to tweets",
				Description: "Agent can reply to any tweet with any text content.",
				Parameters:  json.RawMessage(`{"text":"*","reply_to_tweet_id":"*"}`),
			},
			{
				ID:          "tpl_x_delete_tweet",
				ActionType:  "x.delete_tweet",
				Name:        "Delete tweets",
				Description: "Agent can delete any tweet.",
				Parameters:  json.RawMessage(`{"tweet_id":"*"}`),
			},
			{
				ID:          "tpl_x_send_dm",
				ActionType:  "x.send_dm",
				Name:        "Send direct messages",
				Description: "Agent can send DMs to any user.",
				Parameters:  json.RawMessage(`{"recipient_id":"*","text":"*"}`),
			},
			{
				ID:          "tpl_x_get_user_tweets",
				ActionType:  "x.get_user_tweets",
				Name:        "Read user tweets",
				Description: "Agent can read tweets from any user.",
				Parameters:  json.RawMessage(`{"user_id":"*"}`),
			},
			{
				ID:          "tpl_x_search_tweets",
				ActionType:  "x.search_tweets",
				Name:        "Search for mentions of my brand",
				Description: "Agent can search tweets with any query.",
				Parameters:  json.RawMessage(`{"query":"*"}`),
			},
			{
				ID:          "tpl_x_get_me",
				ActionType:  "x.get_me",
				Name:        "Read my profile",
				Description: "Agent can read the authenticated user's profile.",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "tpl_x_like_tweet",
				ActionType:  "x.like_tweet",
				Name:        "Like tweets",
				Description: "Agent can like any tweet on my behalf.",
				Parameters:  json.RawMessage(`{"user_id":"*","tweet_id":"*"}`),
			},
			{
				ID:          "tpl_x_unlike_tweet",
				ActionType:  "x.unlike_tweet",
				Name:        "Unlike tweets",
				Description: "Agent can unlike any tweet on my behalf.",
				Parameters:  json.RawMessage(`{"user_id":"*","tweet_id":"*"}`),
			},
			{
				ID:          "tpl_x_retweet",
				ActionType:  "x.retweet",
				Name:        "Retweet tweets",
				Description: "Agent can retweet any tweet on my behalf.",
				Parameters:  json.RawMessage(`{"user_id":"*","tweet_id":"*"}`),
			},
			{
				ID:          "tpl_x_unretweet",
				ActionType:  "x.unretweet",
				Name:        "Undo retweets",
				Description: "Agent can undo retweets on my behalf.",
				Parameters:  json.RawMessage(`{"user_id":"*","tweet_id":"*"}`),
			},
			{
				ID:          "tpl_x_follow_user",
				ActionType:  "x.follow_user",
				Name:        "Follow users",
				Description: "Agent can follow any user on my behalf.",
				Parameters:  json.RawMessage(`{"user_id":"*","target_user_id":"*"}`),
			},
			{
				ID:          "tpl_x_unfollow_user",
				ActionType:  "x.unfollow_user",
				Name:        "Unfollow users",
				Description: "Agent can unfollow any user on my behalf.",
				Parameters:  json.RawMessage(`{"user_id":"*","target_user_id":"*"}`),
			},
			{
				ID:          "tpl_x_get_followers",
				ActionType:  "x.get_followers",
				Name:        "Read my followers",
				Description: "Agent can read the followers list for any user.",
				Parameters:  json.RawMessage(`{"user_id":"*"}`),
			},
			{
				ID:          "tpl_x_get_following",
				ActionType:  "x.get_following",
				Name:        "Read who I follow",
				Description: "Agent can read the following list for any user.",
				Parameters:  json.RawMessage(`{"user_id":"*"}`),
			},
			{
				ID:          "tpl_x_upload_media",
				ActionType:  "x.upload_media",
				Name:        "Upload media for tweets",
				Description: "Agent can upload images and GIFs for use in tweets.",
				Parameters:  json.RawMessage(`{"media_data":"*"}`),
			},
		},
	}
}
