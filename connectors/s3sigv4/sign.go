// Package s3sigv4 implements AWS Signature Version 4 request signing.
// Shared by the aws and tresorit connectors for S3-compatible APIs.
package s3sigv4

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Credentials holds the access key material for SigV4 signing.
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string // optional
}

// SigningConfig identifies the AWS region and service for the signature.
type SigningConfig struct {
	Region  string
	Service string
}

// SignRequest signs req with AWS Signature Version 4 using the given
// credentials, payload bytes, and signing configuration.
func SignRequest(req *http.Request, creds Credentials, payload []byte, cfg SigningConfig) error {
	return signRequestAt(req, creds, payload, cfg, time.Now().UTC())
}

func signRequestAt(req *http.Request, creds Credentials, payload []byte, cfg SigningConfig, now time.Time) error {
	datestamp := now.Format("20060102")
	amzdate := now.Format("20060102T150405Z")

	req.Header.Set("X-Amz-Date", amzdate)
	req.Header.Set("Host", req.Host)

	if creds.SessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", creds.SessionToken)
	}

	payloadHash := SHA256Hex(payload)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)

	canonicalHeaders, signedHeaders := BuildCanonicalHeaders(req)
	canonicalQuerystring := ""
	if req.URL.RawQuery != "" {
		canonicalQuerystring = req.URL.RawQuery
	}
	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.Path,
		canonicalQuerystring,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := datestamp + "/" + cfg.Region + "/" + cfg.Service + "/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzdate,
		credentialScope,
		SHA256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := DeriveSigningKey(creds.SecretAccessKey, datestamp, cfg.Region, cfg.Service)
	signature := hex.EncodeToString(HMACSHA256(signingKey, []byte(stringToSign)))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		creds.AccessKeyID, credentialScope, signedHeaders, signature)
	req.Header.Set("Authorization", authHeader)

	return nil
}

// ParseServiceHost extracts service and region from an AWS hostname.
// For standard hosts like "ec2.us-east-1.amazonaws.com", returns ("ec2", "us-east-1").
func ParseServiceHost(host string) (service, region string) {
	host = strings.TrimSuffix(host, ".amazonaws.com")
	parts := strings.SplitN(host, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return host, "us-east-1"
}

// SHA256Hex returns the lowercase hex-encoded SHA-256 digest of data.
func SHA256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// HMACSHA256 returns the HMAC-SHA256 of data using key.
func HMACSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// DeriveSigningKey derives the SigV4 signing key from the secret key.
func DeriveSigningKey(secret, datestamp, region, service string) []byte {
	kDate := HMACSHA256([]byte("AWS4"+secret), []byte(datestamp))
	kRegion := HMACSHA256(kDate, []byte(region))
	kService := HMACSHA256(kRegion, []byte(service))
	return HMACSHA256(kService, []byte("aws4_request"))
}

// BuildCanonicalHeaders builds the canonical headers and signed headers
// strings for SigV4.
func BuildCanonicalHeaders(req *http.Request) (canonicalHeaders, signedHeaders string) {
	headers := make(map[string]string)
	var names []string
	for name := range req.Header {
		lower := strings.ToLower(name)
		if lower == "host" || lower == "content-type" || strings.HasPrefix(lower, "x-amz-") {
			headers[lower] = strings.TrimSpace(req.Header.Get(name))
			names = append(names, lower)
		}
	}
	sort.Strings(names)

	var canonical, signed []string
	for _, name := range names {
		canonical = append(canonical, name+":"+headers[name])
		signed = append(signed, name)
	}

	return strings.Join(canonical, "\n") + "\n", strings.Join(signed, ";")
}

// URIEncodePath encodes a path string per AWS SigV4 rules: each segment is
// percent-encoded but "/" separators are preserved.
func URIEncodePath(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}
