package imessage

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	defaultCLIPath   = "imsg"
	maxRPCResponse   = 10 << 20 // 10 MiB
	maxStderrCapture = 8 << 10  // 8 KiB
)

// commandConfig describes how to reach the imsg binary (local or over SSH).
type commandConfig struct {
	CLIPath    string
	RemoteHost string
}

func commandConfigFromCreds(creds connectors.Credentials) (commandConfig, error) {
	path, _ := creds.Get(credKeyCLIPath)
	host, _ := creds.Get(credKeyRemoteHost)
	cfg := commandConfig{CLIPath: path, RemoteHost: host}
	if err := validateCommandConfig(cfg); err != nil {
		return commandConfig{}, err
	}
	return cfg, nil
}

func (c commandConfig) cliPath() string {
	if strings.TrimSpace(c.CLIPath) == "" {
		return defaultCLIPath
	}
	return strings.TrimSpace(c.CLIPath)
}

func validateCommandConfig(cfg commandConfig) error {
	host := strings.TrimSpace(cfg.RemoteHost)
	if host != "" {
		if strings.HasPrefix(host, "-") {
			return &connectors.ValidationError{Message: "remote_host must not start with '-'"}
		}
		for _, r := range host {
			if !isRemoteHostChar(r) {
				return &connectors.ValidationError{Message: "remote_host contains invalid characters"}
			}
		}
	}

	path := strings.TrimSpace(cfg.CLIPath)
	if path == "" {
		return nil
	}
	if strings.HasPrefix(path, "-") {
		return &connectors.ValidationError{Message: "cli_path must not start with '-'"}
	}
	if strings.ContainsAny(path, ";|&$`\"'<>(){}[]*?~!#\\") {
		return &connectors.ValidationError{Message: "cli_path contains invalid characters"}
	}
	return nil
}

func isRemoteHostChar(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '-', r == '_', r == '.', r == '@', r == ':':
		return true
	default:
		return false
	}
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	if e == nil {
		return "rpc error"
	}
	return e.Message
}

// boundedStderr keeps the tail of subprocess stderr for diagnostics.
type boundedStderr struct {
	mu  sync.Mutex
	buf []byte
	max int
}

func newBoundedStderr(max int) *boundedStderr {
	return &boundedStderr{max: max}
}

func (b *boundedStderr) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	if len(b.buf) > b.max {
		b.buf = append([]byte(nil), b.buf[len(b.buf)-b.max:]...)
	}
	return len(p), nil
}

func (b *boundedStderr) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.TrimSpace(string(b.buf))
}

// rpcSession keeps a persistent imsg rpc subprocess alive for reuse.
type rpcSession struct {
	mu     sync.Mutex
	cfg    commandConfig
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr *boundedStderr
	nextID atomic.Int64
	closed atomic.Bool
}

func newRPCSession(cfg commandConfig) (*rpcSession, error) {
	s := &rpcSession{cfg: cfg, stderr: newBoundedStderr(maxStderrCapture)}
	if err := s.start(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *rpcSession) start() error {
	var cmd *exec.Cmd
	path := s.cfg.cliPath()
	if s.cfg.RemoteHost != "" {
		cmd = exec.Command("ssh", "-T", "--", s.cfg.RemoteHost, path, "rpc")
	} else {
		cmd = exec.Command(path, "rpc")
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("imsg rpc stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return fmt.Errorf("imsg rpc stdout pipe: %w", err)
	}
	cmd.Stderr = s.stderr

	if err := cmd.Start(); err != nil {
		stdin.Close()
		return mapStartError(err)
	}

	s.cmd = cmd
	s.stdin = stdin
	s.stdout = bufio.NewReader(stdout)
	s.closed.Store(false)
	return nil
}

func (s *rpcSession) tearDownLocked() {
	s.closed.Store(true)
	if s.stdin != nil {
		_ = s.stdin.Close()
		s.stdin = nil
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
		s.cmd = nil
	}
	s.stdout = nil
}

func mapStartError(err error) error {
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "executable file not found"),
		strings.Contains(lower, "no such file"):
		return &connectors.ExternalError{
			Message: "imsg not found — install with: brew install steipete/tap/imsg",
		}
	case strings.Contains(lower, "connection refused"),
		strings.Contains(lower, "host key verification failed"),
		strings.Contains(lower, "permission denied"):
		return &connectors.ExternalError{
			Message: fmt.Sprintf("failed to start imsg rpc: %s", msg),
		}
	default:
		return &connectors.ExternalError{
			Message: fmt.Sprintf("failed to start imsg rpc: %s", msg),
		}
	}
}

func (s *rpcSession) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Load() {
		return
	}
	s.tearDownLocked()
}

func (s *rpcSession) call(ctx context.Context, method string, params any, result any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed.Load() {
		if err := s.start(); err != nil {
			return err
		}
	}

	id := int(s.nextID.Add(1))
	req := rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal rpc request: %w", err)
	}
	payload = append(payload, '\n')

	if err := ctx.Err(); err != nil {
		return mapContextError(err)
	}

	if _, err := s.stdin.Write(payload); err != nil {
		s.tearDownLocked()
		return &connectors.ExternalError{Message: fmt.Sprintf("imsg rpc write failed: %v", err)}
	}

	resp, err := readRPCResponse(ctx, s.stdout, id)
	if err != nil {
		s.tearDownLocked()
		return err
	}
	if resp.Error != nil {
		return mapRPCError(resp.Error, s.stderr.String())
	}
	if result == nil {
		return nil
	}
	if len(resp.Result) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Result, result); err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("parse imsg rpc result: %v", err)}
	}
	return nil
}

func readLineLimited(r *bufio.Reader, max int) ([]byte, error) {
	var line []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				return line, nil
			}
			return nil, err
		}
		if b == '\n' {
			return line, nil
		}
		if len(line)+1 > max {
			return nil, fmt.Errorf("line exceeds %d bytes", max)
		}
		line = append(line, b)
	}
}

func readRPCResponse(ctx context.Context, r *bufio.Reader, wantID int) (*rpcResponse, error) {
	type result struct {
		resp *rpcResponse
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		for {
			line, err := readLineLimited(r, maxRPCResponse)
			if err != nil {
				ch <- result{err: err}
				return
			}
			trimmed := strings.TrimSpace(string(line))
			if trimmed == "" {
				continue
			}
			var resp rpcResponse
			if err := json.Unmarshal([]byte(trimmed), &resp); err != nil {
				ch <- result{err: &connectors.ExternalError{Message: fmt.Sprintf("parse imsg rpc response: %v", err)}}
				return
			}
			if resp.ID != wantID {
				continue
			}
			ch <- result{resp: &resp}
			return
		}
	}()

	select {
	case <-ctx.Done():
		return nil, mapContextError(ctx.Err())
	case res := <-ch:
		if res.err != nil {
			if connectors.IsTimeout(res.err) {
				return nil, &connectors.TimeoutError{Message: fmt.Sprintf("imsg rpc read failed: %v", res.err)}
			}
			if strings.Contains(res.err.Error(), "line exceeds") {
				return nil, &connectors.ExternalError{Message: "imsg rpc response too large"}
			}
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("imsg rpc read failed: %v", res.err)}
		}
		return res.resp, nil
	}
}

func formatRPCErrorMessage(err *rpcError, stderr string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "imsg rpc error %d: %s", err.Code, err.Message)
	if len(err.Data) > 0 {
		data := strings.TrimSpace(string(err.Data))
		if data != "" {
			fmt.Fprintf(&b, " (data: %s)", data)
		}
	}
	if stderr != "" {
		fmt.Fprintf(&b, " (stderr: %s)", stderr)
	}
	return b.String()
}

func classifyRPCFailureText(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strings.ToLower(p))
	}
	return b.String()
}

func mapRPCError(err *rpcError, stderr string) error {
	if err == nil {
		return &connectors.ExternalError{Message: "imsg rpc error"}
	}
	msg := formatRPCErrorMessage(err, stderr)
	lower := classifyRPCFailureText(err.Message, string(err.Data), stderr)
	switch {
	case strings.Contains(lower, "authorization denied"),
		strings.Contains(lower, "unable to open database"),
		strings.Contains(lower, "full disk access"):
		return &connectors.AuthError{
			Message: "imsg cannot read Messages database — grant Full Disk Access to the process running Permission Slip (and imsg) in System Settings → Privacy & Security",
		}
	case strings.Contains(lower, "automation"),
		strings.Contains(lower, "not authorized to send"):
		return &connectors.AuthError{
			Message: "imsg cannot send messages — grant Automation permission for Messages.app in System Settings → Privacy & Security",
		}
	default:
		return &connectors.ExternalError{Message: msg}
	}
}

func mapContextError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &connectors.TimeoutError{Message: "imsg rpc request timed out"}
	}
	return &connectors.CanceledError{Message: err.Error()}
}

// sessionPool reuses rpc sessions per command configuration.
type sessionPool struct {
	mu       sync.Mutex
	sessions map[string]*rpcSession
}

func newSessionPool() *sessionPool {
	return &sessionPool{sessions: make(map[string]*rpcSession)}
}

func (p *sessionPool) sessionKey(cfg commandConfig) string {
	return cfg.cliPath() + "|" + cfg.RemoteHost
}

func (p *sessionPool) get(cfg commandConfig) (*rpcSession, error) {
	key := p.sessionKey(cfg)
	p.mu.Lock()
	defer p.mu.Unlock()
	if s, ok := p.sessions[key]; ok && !s.closed.Load() {
		return s, nil
	}
	s, err := newRPCSession(cfg)
	if err != nil {
		return nil, err
	}
	p.sessions[key] = s
	return s, nil
}

func (p *sessionPool) closeAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, s := range p.sessions {
		s.close()
	}
	p.sessions = make(map[string]*rpcSession)
}

// imsgClient talks to imsg via persistent RPC and one-shot CLI commands.
type imsgClient struct {
	pool *sessionPool
}

func newIMsgClient() *imsgClient {
	return &imsgClient{pool: newSessionPool()}
}

func (c *imsgClient) rpcCall(ctx context.Context, creds connectors.Credentials, method string, params any, result any) error {
	cfg, err := commandConfigFromCreds(creds)
	if err != nil {
		return err
	}
	s, err := c.pool.get(cfg)
	if err != nil {
		return err
	}
	return s.call(ctx, method, params, result)
}

// runCLI executes a one-shot imsg subcommand and returns newline-delimited JSON objects.
func (c *imsgClient) runCLI(ctx context.Context, creds connectors.Credentials, args ...string) ([]json.RawMessage, error) {
	cfg, err := commandConfigFromCreds(creds)
	if err != nil {
		return nil, err
	}
	path := cfg.cliPath()

	var cmd *exec.Cmd
	if cfg.RemoteHost != "" {
		remoteArgs := append([]string{"-T", "--", cfg.RemoteHost, path}, args...)
		cmd = exec.CommandContext(ctx, "ssh", remoteArgs...)
	} else {
		cmd = exec.CommandContext(ctx, path, args...)
	}

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	stdout, err := cmd.Output()
	if err != nil {
		return nil, mapCLIError(err, stderrBuf.String())
	}
	if len(stdout) > maxRPCResponse {
		return nil, &connectors.ExternalError{Message: "imsg output too large"}
	}

	lines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	out := make([]json.RawMessage, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, json.RawMessage(line))
	}
	return out, nil
}

func mapCLIError(err error, stderr string) error {
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("imsg command timed out: %v", err)}
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if stderr == "" {
			stderr = strings.TrimSpace(string(exitErr.Stderr))
		}
		lower := classifyRPCFailureText(stderr)
		switch {
		case strings.Contains(lower, "authorization denied"),
			strings.Contains(lower, "unable to open database"),
			strings.Contains(lower, "full disk access"):
			return &connectors.AuthError{
				Message: "imsg cannot read Messages database — grant Full Disk Access in System Settings → Privacy & Security",
			}
		case strings.Contains(lower, "automation"):
			return &connectors.AuthError{
				Message: "imsg cannot send messages — grant Automation permission for Messages.app",
			}
		}
		if stderr != "" {
			return &connectors.ExternalError{Message: stderr}
		}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("imsg command failed: %v", err)}
}
