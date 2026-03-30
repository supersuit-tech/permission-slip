package x

import (
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func errMissingParam(name string) error {
	return &connectors.ValidationError{Message: "missing required parameter: " + name}
}

func errInvalidParam(msg string) error {
	return &connectors.ValidationError{Message: msg}
}

func errBadJSON(err error) error {
	return &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
}

// userTweetParams is shared by like_tweet, unlike_tweet, retweet, and unretweet.
// UserID is optional — if empty it is resolved via /users/me before use.
type userTweetParams struct {
	UserID  string `json:"user_id"`
	TweetID string `json:"tweet_id"`
}

func (p *userTweetParams) validate() error {
	if p.TweetID == "" {
		return errMissingParam("tweet_id")
	}
	return nil
}

// userFollowParams is shared by follow_user and unfollow_user.
// UserID is optional — if empty it is resolved via /users/me before use.
type userFollowParams struct {
	UserID       string `json:"user_id"`
	TargetUserID string `json:"target_user_id"`
}

func (p *userFollowParams) validate() error {
	if p.TargetUserID == "" {
		return errMissingParam("target_user_id")
	}
	return nil
}

// userListParams is shared by get_followers and get_following.
// UserID is optional — if empty it is resolved via /users/me before use.
type userListParams struct {
	UserID          string `json:"user_id"`
	MaxResults      int    `json:"max_results"`
	PaginationToken string `json:"pagination_token"`
}

func (p *userListParams) validate() error {
	if p.MaxResults != 0 && (p.MaxResults < 1 || p.MaxResults > 1000) {
		return errInvalidParam("max_results must be between 1 and 1000")
	}
	return nil
}
