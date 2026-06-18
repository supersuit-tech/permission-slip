package s3sigv4

import (
	"net/http"
	"testing"
	"time"
)

func TestParseServiceHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		host        string
		wantService string
		wantRegion  string
	}{
		{"ec2.us-east-1.amazonaws.com", "ec2", "us-east-1"},
		{"s3.us-west-2.amazonaws.com", "s3", "us-west-2"},
		{"monitoring.eu-west-1.amazonaws.com", "monitoring", "eu-west-1"},
		{"rds.ap-southeast-1.amazonaws.com", "rds", "ap-southeast-1"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			t.Parallel()
			service, region := ParseServiceHost(tt.host)
			if service != tt.wantService {
				t.Errorf("service = %q, want %q", service, tt.wantService)
			}
			if region != tt.wantRegion {
				t.Errorf("region = %q, want %q", region, tt.wantRegion)
			}
		})
	}
}

func TestSHA256Hex(t *testing.T) {
	t.Parallel()
	got := SHA256Hex([]byte(""))
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("SHA256Hex(\"\") = %q, want %q", got, want)
	}
}

func TestDeriveSigningKeyDeterministic(t *testing.T) {
	t.Parallel()
	k1 := DeriveSigningKey("secret", "20240101", "us-east-1", "s3")
	k2 := DeriveSigningKey("secret", "20240101", "us-east-1", "s3")
	if len(k1) != 32 {
		t.Fatalf("signing key length = %d, want 32", len(k1))
	}
	for i := range k1 {
		if k1[i] != k2[i] {
			t.Fatal("DeriveSigningKey is not deterministic")
		}
	}
}

func TestBuildCanonicalHeaders(t *testing.T) {
	t.Parallel()
	req, err := http.NewRequest(http.MethodGet, "https://s3.us-east-1.amazonaws.com/bucket", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Host", "s3.us-east-1.amazonaws.com")
	req.Header.Set("X-Amz-Date", "20240101T000000Z")
	req.Header.Set("X-Amz-Content-Sha256", "abc")

	canonical, signed := BuildCanonicalHeaders(req)
	if signed != "host;x-amz-content-sha256;x-amz-date" {
		t.Errorf("signed headers = %q", signed)
	}
	if !stringsContains(canonical, "host:s3.us-east-1.amazonaws.com") {
		t.Errorf("canonical headers missing host: %q", canonical)
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestSignRequestAWSStyle(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest(http.MethodGet, "https://s3.us-east-1.amazonaws.com/my-bucket?list-type=2", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "s3.us-east-1.amazonaws.com"

	creds := Credentials{
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	cfg := SigningConfig{Region: "us-east-1", Service: "s3"}
	fixed := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	if err := signRequestAt(req, creds, nil, cfg, fixed); err != nil {
		t.Fatalf("signRequestAt() error: %v", err)
	}

	auth := req.Header.Get("Authorization")
	if auth == "" {
		t.Fatal("Authorization header is empty")
	}
	if !stringsContains(auth, "Credential=AKIAIOSFODNN7EXAMPLE/20240101/us-east-1/s3/aws4_request") {
		t.Errorf("unexpected Authorization: %s", auth)
	}
	if req.Header.Get("X-Amz-Date") != "20240101T000000Z" {
		t.Errorf("X-Amz-Date = %q", req.Header.Get("X-Amz-Date"))
	}
}

func TestSignRequestCustomHTTPEndpoint(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:3000/my-tresor?list-type=2", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "127.0.0.1:3000"

	creds := Credentials{
		AccessKeyID:     "client-id",
		SecretAccessKey: "client-secret",
	}
	cfg := SigningConfig{Region: "us-east-1", Service: "s3"}
	fixed := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	if err := signRequestAt(req, creds, nil, cfg, fixed); err != nil {
		t.Fatalf("signRequestAt() error: %v", err)
	}

	auth := req.Header.Get("Authorization")
	if auth == "" {
		t.Fatal("Authorization header is empty")
	}
	if !stringsContains(auth, "Credential=client-id/20240601/us-east-1/s3/aws4_request") {
		t.Errorf("unexpected Authorization: %s", auth)
	}
	if req.URL.Scheme != "http" {
		t.Errorf("scheme = %q, want http", req.URL.Scheme)
	}
}

func TestURIEncodePath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in, want string
	}{
		{"bucket/key", "bucket/key"},
		{"bucket/my file.txt", "bucket/my%20file.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := URIEncodePath(tt.in); got != tt.want {
				t.Errorf("URIEncodePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
