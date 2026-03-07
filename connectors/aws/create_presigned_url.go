package aws

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createPresignedURLAction implements connectors.Action for aws.create_presigned_url.
type createPresignedURLAction struct {
	conn *AWSConnector
}

type createPresignedURLParams struct {
	Region    string `json:"region"`
	Bucket    string `json:"bucket"`
	Key       string `json:"key"`
	Operation string `json:"operation"`
	ExpiresIn int    `json:"expires_in"`
}

func (p *createPresignedURLParams) validate() error {
	if p.Region == "" {
		return &connectors.ValidationError{Message: "missing required parameter: region"}
	}
	if p.Bucket == "" {
		return &connectors.ValidationError{Message: "missing required parameter: bucket"}
	}
	if p.Key == "" {
		return &connectors.ValidationError{Message: "missing required parameter: key"}
	}
	if p.Operation == "" {
		return &connectors.ValidationError{Message: "missing required parameter: operation"}
	}
	if p.Operation != "GET" && p.Operation != "PUT" {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid operation %q: must be GET or PUT", p.Operation)}
	}
	if p.ExpiresIn == 0 {
		p.ExpiresIn = 3600
	}
	if p.ExpiresIn < 1 || p.ExpiresIn > 604800 {
		return &connectors.ValidationError{Message: "expires_in must be between 1 and 604800 seconds"}
	}
	return nil
}

// Execute generates a presigned S3 URL for GET or PUT operations.
// This does not make an API call — it constructs the signed URL locally.
func (a *createPresignedURLAction) Execute(_ context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createPresignedURLParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	accessKey, _ := req.Credentials.Get("access_key_id")
	secretKey, _ := req.Credentials.Get("secret_access_key")
	sessionToken, hasToken := req.Credentials.Get("session_token")

	now := time.Now().UTC()
	datestamp := now.Format("20060102")
	amzdate := now.Format("20060102T150405Z")

	host := fmt.Sprintf("s3.%s.amazonaws.com", params.Region)
	objectPath := "/" + params.Bucket + "/" + params.Key
	credentialScope := datestamp + "/" + params.Region + "/s3/aws4_request"

	// Build canonical query string for presigned URL.
	qp := url.Values{}
	qp.Set("X-Amz-Algorithm", "AWS4-HMAC-SHA256")
	qp.Set("X-Amz-Credential", accessKey+"/"+credentialScope)
	qp.Set("X-Amz-Date", amzdate)
	qp.Set("X-Amz-Expires", strconv.Itoa(params.ExpiresIn))
	qp.Set("X-Amz-SignedHeaders", "host")
	if hasToken && sessionToken != "" {
		qp.Set("X-Amz-Security-Token", sessionToken)
	}
	canonicalQuerystring := qp.Encode()

	// Canonical request.
	canonicalRequest := strings.Join([]string{
		params.Operation,
		objectPath,
		canonicalQuerystring,
		"host:" + host + "\n",
		"host",
		"UNSIGNED-PAYLOAD",
	}, "\n")

	// String to sign.
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzdate,
		credentialScope,
		sha256HexStr([]byte(canonicalRequest)),
	}, "\n")

	// Signing key and signature.
	signingKey := derivePresignKey(secretKey, datestamp, params.Region)
	signature := hex.EncodeToString(hmacSHA256Presign(signingKey, []byte(stringToSign)))

	presignedURL := fmt.Sprintf("https://%s%s?%s&X-Amz-Signature=%s",
		host, objectPath, canonicalQuerystring, signature)

	return connectors.JSONResult(map[string]any{
		"url":        presignedURL,
		"operation":  params.Operation,
		"bucket":     params.Bucket,
		"key":        params.Key,
		"expires_in": params.ExpiresIn,
	})
}

func sha256HexStr(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func hmacSHA256Presign(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func derivePresignKey(secret, datestamp, region string) []byte {
	kDate := hmacSHA256Presign([]byte("AWS4"+secret), []byte(datestamp))
	kRegion := hmacSHA256Presign(kDate, []byte(region))
	kService := hmacSHA256Presign(kRegion, []byte("s3"))
	return hmacSHA256Presign(kService, []byte("aws4_request"))
}
