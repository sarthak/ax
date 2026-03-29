//go:build integration

package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sarthakagrawal/ax/internal"
	"github.com/stretchr/testify/require"
)

// testSocket is the tmux socket name used by the current test run, tracked for
// TestMain-level cleanup.
var testSocket string

// tmuxHarness manages an isolated tmux server for e2e testing.
type tmuxHarness struct {
	t          *testing.T
	tmux       *internal.TmuxClient
	axBin      string
	socket     string
	stateDir   string
	cmdCounter atomic.Int64
}

// newTmuxHarness creates an isolated tmux test environment.
// It starts a detached tmux session and registers cleanup to kill the server.
// Assumes the ax binary has been built at ../ax (make test-integration handles this).
func newTmuxHarness(t *testing.T) *tmuxHarness {
	t.Helper()

	// Resolve ax binary path relative to the e2e/ test directory.
	axBin, err := filepath.Abs("../ax")
	require.NoError(t, err, "failed to resolve ax binary path")
	require.FileExists(t, axBin, "ax binary not found — run `make build` first")

	socket := fmt.Sprintf("ax-test-%d", os.Getpid())
	testSocket = socket
	stateDir := t.TempDir()

	t.Setenv("AX_TMUX_SOCKET", socket)
	t.Setenv("AX_STATE_DIR", stateDir)

	tmux := internal.NewTmuxClient()

	t.Cleanup(func() {
		_, _ = tmux.Run("kill-server")
	})

	h := &tmuxHarness{
		t:        t,
		tmux:     tmux,
		axBin:    axBin,
		socket:   socket,
		stateDir: stateDir,
	}

	// Create the initial tmux session.
	err = tmux.NewSession("e2e")
	require.NoError(t, err, "failed to create test tmux session")

	// Set env vars on the tmux session so all panes inherit them.
	require.NoError(t, tmux.SetEnv("AX_TMUX_SOCKET", socket, "e2e"))
	require.NoError(t, tmux.SetEnv("AX_STATE_DIR", stateDir, "e2e"))

	return h
}

// runInPane executes an ax command inside a tmux pane and waits for completion.
// Returns the combined stdout+stderr output and the exit code.
func (h *tmuxHarness) runInPane(paneID string, args ...string) (string, int) {
	h.t.Helper()

	id := h.cmdCounter.Add(1)
	channel := fmt.Sprintf("ax-done-%d", id)
	outFile := filepath.Join(h.stateDir, fmt.Sprintf("out-%d", id))
	rcFile := filepath.Join(h.stateDir, fmt.Sprintf("rc-%d", id))

	// Build the shell command to inject.
	// AX_TMUX_SOCKET and AX_STATE_DIR are already in the pane's env (set on
	// the tmux session), so ax picks them up automatically.
	quotedArgs := make([]string, len(args))
	for i, a := range args {
		quotedArgs[i] = shellescape(a)
	}
	shellCmd := fmt.Sprintf("%s %s > %s 2>&1; echo $? > %s; tmux -L $AX_TMUX_SOCKET wait-for -S %s",
		shellescape(h.axBin), strings.Join(quotedArgs, " "),
		shellescape(outFile), shellescape(rcFile), channel)

	// Inject the command into the pane.
	err := h.tmux.SendKeys(paneID, shellCmd)
	require.NoError(h.t, err, "failed to send keys to pane %s", paneID)

	// Wait for completion with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	waitArgs := []string{"wait-for", channel}
	if sock := os.Getenv("AX_TMUX_SOCKET"); sock != "" {
		waitArgs = append([]string{"-L", sock}, waitArgs...)
	}
	waitCmd := exec.CommandContext(ctx, "tmux", waitArgs...)
	err = waitCmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		h.t.Fatalf("command timed out after 10s: ax %s", strings.Join(args, " "))
	}
	require.NoError(h.t, err, "wait-for failed for channel %s", channel)

	// Read output and exit code.
	outBytes, err := os.ReadFile(outFile)
	require.NoError(h.t, err, "failed to read output file %s", outFile)

	rcBytes, err := os.ReadFile(rcFile)
	require.NoError(h.t, err, "failed to read exit code file %s", rcFile)

	rc, err := strconv.Atoi(strings.TrimSpace(string(rcBytes)))
	require.NoError(h.t, err, "failed to parse exit code: %q", string(rcBytes))

	return string(outBytes), rc
}

// capturePaneContent returns the visible content of a tmux pane.
// Uses -J to join wrapped lines so assertions don't break on narrow panes.
func (h *tmuxHarness) capturePaneContent(paneID string) string {
	h.t.Helper()
	out, err := h.tmux.Run("capture-pane", "-t", paneID, "-p", "-J")
	require.NoError(h.t, err, "failed to capture pane %s", paneID)
	return out
}

// killPane destroys a tmux pane.
func (h *tmuxHarness) killPane(paneID string) {
	h.t.Helper()
	_, err := h.tmux.Run("kill-pane", "-t", paneID)
	require.NoError(h.t, err, "failed to kill pane %s", paneID)
}

// readRegistry parses the registry.json for a given session ID.
func (h *tmuxHarness) readRegistry(sessionID string) []internal.Agent {
	h.t.Helper()
	path := filepath.Join(h.stateDir, "sessions", sessionID, "registry.json")
	data, err := os.ReadFile(path)
	require.NoError(h.t, err, "failed to read registry at %s", path)

	var agents []internal.Agent
	err = json.Unmarshal(data, &agents)
	require.NoError(h.t, err, "failed to parse registry JSON")
	return agents
}

// waitForPaneContent polls capture-pane until the content contains the expected
// substring or the timeout expires. Useful after send-keys which is async.
func (h *tmuxHarness) waitForPaneContent(paneID, substr string, timeout time.Duration) string {
	h.t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		content := h.capturePaneContent(paneID)
		if strings.Contains(content, substr) {
			return content
		}
		time.Sleep(100 * time.Millisecond)
	}
	content := h.capturePaneContent(paneID)
	h.t.Fatalf("timed out waiting for %q in pane %s, got:\n%s", substr, paneID, content)
	return ""
}

// shellescape wraps a string in single quotes for safe shell injection.
func shellescape(s string) string {
	// Replace single quotes with '\'' (end quote, escaped quote, start quote).
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// initialPaneID returns the pane ID of the first pane in the test session.
func (h *tmuxHarness) initialPaneID() string {
	h.t.Helper()
	out, err := h.tmux.Run("list-panes", "-t", "e2e", "-F", "#{pane_id}")
	require.NoError(h.t, err, "failed to list panes")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	require.NotEmpty(h.t, lines, "no panes found in session e2e")
	return lines[0]
}
