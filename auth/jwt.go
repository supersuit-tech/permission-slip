package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const accessTokenTTL = 15 * time.Minute

// IssueAccessToken returns an HS256 JWT and its absolute expiry time.
func IssueAccessToken(secret []byte, userID, email string, now time.Time) (token string, expiresAt time.Time, err error) {
	if len(secret) < 32 {
		return "", time.Time{}, errors.New("JWT signing secret must be at least 32 bytes")
	}
	if userID == "" {
		return "", time.Time{}, errors.New("empty user id")
	}
	expiresAt = now.Add(accessTokenTTL)
	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"iat":   now.Unix(),
		"exp":   expiresAt.Unix(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token, err = tok.SignedString(secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return token, expiresAt, nil
}

// VerifyAccessToken validates an HS256 access JWT and returns subject and email claims.
func VerifyAccessToken(secret []byte, tokenString string) (userID, email string, err error) {
	if len(secret) < 32 {
		return "", "", errors.New("JWT signing secret must be at least 32 bytes")
	}
	tok, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %q", t.Header["alg"])
		}
		return secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || !tok.Valid {
		return "", "", fmt.Errorf("invalid token: %w", err)
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errors.New("invalid claims type")
	}
	sub, err := claims.GetSubject()
	if err != nil || sub == "" {
		return "", "", errors.New("missing sub claim")
	}
	em, _ := claims["email"].(string)
	return sub, em, nil
}
