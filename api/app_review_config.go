package api

import (
	"os"
	"strings"
)

// appReviewConfig holds the App Store review account configuration.
type appReviewConfig struct {
	email   string
	otp     string
	enabled bool
}

// loadAppReviewConfig reads the app review credentials from environment variables.
// The feature is only enabled when both APP_REVIEW_EMAIL and APP_REVIEW_OTP are set.
func loadAppReviewConfig() appReviewConfig {
	email := strings.TrimSpace(os.Getenv("APP_REVIEW_EMAIL"))
	otp := strings.TrimSpace(os.Getenv("APP_REVIEW_OTP"))
	return appReviewConfig{
		email:   email,
		otp:     otp,
		enabled: email != "" && otp != "",
	}
}
