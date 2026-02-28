package connectors

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"time"
)

// maxStdoutSize is the maximum bytes we'll read from a connector subprocess's
// stdout. This prevents a malicious or buggy binary from OOM-ing the server.
const maxStdoutSize = 10 << 20 // 10 MB

// maxStderrSize is the maximum bytes we'll capture from a connector's stderr.
// Only used in error messages, so a smaller limit suffices.
const maxStderrSize = 1 << 20 // 1 MB

const defaultExecTimeout = 30 * time.Second

// ExternalConnector implements the Connector interface for subprocess-based
// connectors. It reads its metadata from a ConnectorManifest and delegates
// Execute and ValidateCredentials calls to a binary via JSON-over-stdin/stdout.
type ExternalConnector struct {
	manifest *ConnectorManifest
	binPath  string // path to the connector executable
	timeout  time.Duration
}

// NewExternalConnector creates an ExternalConnector from a manifest and binary path.
func NewExternalConnector(manifest *ConnectorManifest, binPath string) *ExternalConnector {
	return &ExternalConnector{
		manifest: manifest,
		binPath:  binPath,
		timeout:  defaultExecTimeout,
	}
}

// ID returns the connector identifier from the manifest.
func (c *ExternalConnector) ID() string { return c.manifest.ID }

// Manifest returns the connector's manifest. Used by the server to auto-seed
// DB rows on startup.
func (c *ExternalConnector) Manifest() *ConnectorManifest { return c.manifest }

// Actions returns action handlers for each action defined in the manifest.
// Each action delegates to the subprocess binary.
func (c *ExternalConnector) Actions() map[string]Action {
	actions := make(map[string]Action, len(c.manifest.Actions))
	for _, a := range c.manifest.Actions {
		actions[a.ActionType] = &externalAction{
			conn:       c,
			actionType: a.ActionType,
		}
	}
	return actions
}

// ValidateCredentials sends a validate_credentials command to the subprocess.
func (c *ExternalConnector) ValidateCredentials(ctx context.Context, creds Credentials) error {
	req := subprocessRequest{
		Command:     "validate_credentials",
		Credentials: creds.ToMap(),
	}
	resp, err := c.invoke(ctx, req)
	if err != nil {
		return err
	}
	if !resp.Success {
		return mapSubprocessError(resp.Error)
	}
	return nil
}

// externalAction implements Action for a single action type on an external connector.
type externalAction struct {
	conn       *ExternalConnector
	actionType string
}

// Execute sends an execute command to the subprocess with the action parameters
// and credentials, then returns the response data.
func (a *externalAction) Execute(ctx context.Context, req ActionRequest) (*ActionResult, error) {
	subReq := subprocessRequest{
		Command:     "execute",
		ActionType:  req.ActionType,
		Parameters:  req.Parameters,
		Credentials: req.Credentials.ToMap(),
	}
	resp, err := a.conn.invoke(ctx, subReq)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, mapSubprocessError(resp.Error)
	}
	return &ActionResult{Data: resp.Data}, nil
}

// subprocessRequest is the JSON payload written to the connector binary's stdin.
type subprocessRequest struct {
	Command     string            `json:"command"`
	ActionType  string            `json:"action_type,omitempty"`
	Parameters  json.RawMessage   `json:"parameters,omitempty"`
	Credentials map[string]string `json:"credentials,omitempty"`
}

// subprocessResponse is the JSON payload read from the connector binary's stdout.
type subprocessResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *subprocessErr  `json:"error,omitempty"`
}

// subprocessErr is the error structure in a subprocess response.
type subprocessErr struct {
	Type    string `json:"type"`    // validation, external, auth, rate_limit, timeout
	Message string `json:"message"`
}

// invoke spawns the connector binary, writes the request to stdin, and reads
// the response from stdout. It enforces a timeout via context.
func (c *ExternalConnector) invoke(ctx context.Context, req subprocessRequest) (*subprocessResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling subprocess request: %w", err)
	}

	cmd := exec.CommandContext(ctx, c.binPath)
	cmd.Dir = filepath.Dir(c.binPath)
	cmd.Stdin = bytes.NewReader(payload)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe for connector %q: %w", c.manifest.ID, err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &limitedWriter{buf: &stderr, remaining: maxStderrSize}

	if err := cmd.Start(); err != nil {
		return nil, &ExternalError{
			Message: fmt.Sprintf("connector %q failed to start: %v", c.manifest.ID, err),
		}
	}

	// Read at most maxStdoutSize+1 bytes so we can detect overflow.
	stdoutBytes, err := io.ReadAll(io.LimitReader(stdoutPipe, maxStdoutSize+1))
	if err != nil {
		return nil, &ExternalError{
			Message: fmt.Sprintf("connector %q: reading stdout: %v", c.manifest.ID, err),
		}
	}
	if len(stdoutBytes) > maxStdoutSize {
		// Kill the process — we won't use its output.
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, &ExternalError{
			Message: fmt.Sprintf("connector %q stdout exceeded %d bytes limit", c.manifest.ID, maxStdoutSize),
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, &TimeoutError{Message: fmt.Sprintf("connector %q timed out after %s", c.manifest.ID, c.timeout)}
		}
		return nil, &ExternalError{
			Message: fmt.Sprintf("connector %q process failed: %v (stderr: %s)", c.manifest.ID, err, stderr.String()),
		}
	}

	var resp subprocessResponse
	if err := json.Unmarshal(stdoutBytes, &resp); err != nil {
		return nil, &ExternalError{
			Message: fmt.Sprintf("connector %q returned invalid JSON: %v (stdout: %s)", c.manifest.ID, err, string(stdoutBytes)),
		}
	}

	return &resp, nil
}

// limitedWriter wraps a bytes.Buffer and silently discards writes once the
// limit is reached. Used for stderr capture so a noisy connector can't OOM
// the server.
type limitedWriter struct {
	buf       *bytes.Buffer
	remaining int
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return len(p), nil // discard silently
	}
	if len(p) > w.remaining {
		p = p[:w.remaining]
	}
	n, err := w.buf.Write(p)
	w.remaining -= n
	return len(p), err // report full write to avoid exec errors
}

// mapSubprocessError converts a subprocess error response to the appropriate
// connector error type.
func mapSubprocessError(e *subprocessErr) error {
	if e == nil {
		return &ExternalError{Message: "connector returned failure with no error details"}
	}
	switch e.Type {
	case "validation":
		return &ValidationError{Message: e.Message}
	case "auth":
		return &AuthError{Message: e.Message}
	case "rate_limit":
		return &RateLimitError{Message: e.Message}
	case "timeout":
		return &TimeoutError{Message: e.Message}
	default:
		return &ExternalError{Message: e.Message}
	}
}
