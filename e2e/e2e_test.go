//go:build integration

package e2e_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/sarthakagrawal/ax/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if _, err := exec.LookPath("tmux"); err != nil {
		fmt.Fprintln(os.Stderr, "skipping integration tests: tmux not found on PATH")
		os.Exit(0)
	}

	// Deferred cleanup: kill any orphaned tmux server.
	defer func() {
		if testSocket != "" {
			_ = exec.Command("tmux", "-L", testSocket, "kill-server").Run()
		}
	}()

	os.Exit(m.Run())
}

func TestInitialPaneInheritsEnvVars(t *testing.T) {
	h := newTmuxHarness(t)
	paneID := h.initialPaneID()

	// The harness passes AX_TMUX_SOCKET and AX_STATE_DIR via NewSession's -e
	// flags. Verify the initial pane's shell inherited them (not just the tmux
	// session environment).
	err := h.tmux.SendKeys(paneID, "echo AX_SOCKET=$AX_TMUX_SOCKET AX_STATE=$AX_STATE_DIR")
	require.NoError(t, err)

	content := h.waitForPaneContent(paneID, "AX_SOCKET="+h.socket, 2*time.Second)
	assert.Contains(t, content, "AX_STATE="+h.stateDir)
}

func TestFullLifecycle(t *testing.T) {
	h := newTmuxHarness(t)
	paneID := h.initialPaneID()

	var sessionID string
	var agent2PaneID string

	paneIDPattern := regexp.MustCompile(`%\d+`)

	t.Run("01_init", func(t *testing.T) {
		out, rc := h.runInPane(paneID, "init")
		require.Equal(t, 0, rc, "ax init failed: %s", out)
		assert.Contains(t, out, "ax session initialized")

		// Read session ID from tmux env.
		sid, err := h.tmux.GetEnv("AX_SESSION_ID", "e2e")
		require.NoError(t, err, "AX_SESSION_ID not set in tmux env")
		assert.Regexp(t, `^[0-9a-f]{6}$`, sid)
		sessionID = sid

		// Verify session directory exists.
		sessionDir := filepath.Join(h.stateDir, "sessions", sessionID)
		assert.DirExists(t, sessionDir)
		assert.FileExists(t, filepath.Join(sessionDir, "registry.json"))
	})

	t.Run("02_join", func(t *testing.T) {
		require.NotEmpty(t, sessionID, "session ID not set — init must run first")

		out, rc := h.runInPane(paneID, "join", "--role", "implementer", "--label", "agent-1")
		require.Equal(t, 0, rc, "ax join failed: %s", out)

		agents := h.readRegistry(sessionID)
		require.Len(t, agents, 1)
		assert.Equal(t, "agent-1", agents[0].Label)
		assert.Equal(t, "implementer", agents[0].Role)
		assert.Equal(t, internal.StatusAlive, agents[0].Status)
		assert.True(t, paneIDPattern.MatchString(agents[0].TmuxPaneID), "pane ID should match %%N format, got %q", agents[0].TmuxPaneID)
	})

	t.Run("03_spawn", func(t *testing.T) {
		require.NotEmpty(t, sessionID, "session ID not set")

		out, rc := h.runInPane(paneID, "spawn", "--role", "reviewer", "--label", "agent-2", "cat")
		require.Equal(t, 0, rc, "ax spawn failed: %s", out)

		// Output should contain a pane ID (agents need this for layout ops).
		assert.True(t, paneIDPattern.MatchString(out), "spawn output should contain a pane ID (%%N), got: %s", out)

		// Registry should have 2 entries.
		agents := h.readRegistry(sessionID)
		require.Len(t, agents, 2)

		// Find agent-2 and store its pane ID.
		for _, a := range agents {
			if a.Label == "agent-2" {
				agent2PaneID = a.TmuxPaneID
				assert.Equal(t, "reviewer", a.Role)
				assert.Equal(t, internal.StatusAlive, a.Status)
			}
		}
		require.NotEmpty(t, agent2PaneID, "agent-2 not found in registry")

		// Verify the pane actually exists.
		alive, err := h.tmux.PaneExists(agent2PaneID)
		require.NoError(t, err, "PaneExists should not fail")
		assert.True(t, alive, "spawned pane %s should exist", agent2PaneID)
	})

	t.Run("04_send_direct", func(t *testing.T) {
		require.NotEmpty(t, agent2PaneID, "agent-2 pane ID not set")

		out, rc := h.runInPane(paneID, "send", "agent-2", "hello from test")
		require.Equal(t, 0, rc, "ax send failed: %s", out)
		assert.Contains(t, out, "Sent to 1 agent(s)")

		// Verify the message appeared in agent-2's pane.
		expected := internal.FormatMessage("agent-1", "implementer", "hello from test")
		h.waitForPaneContent(agent2PaneID, expected, 2*time.Second)
	})

	t.Run("05_send_role", func(t *testing.T) {
		require.NotEmpty(t, agent2PaneID, "agent-2 pane ID not set")

		out, rc := h.runInPane(paneID, "send", "--role", "reviewer", "broadcast msg")
		require.Equal(t, 0, rc, "ax send --role failed: %s", out)
		assert.Contains(t, out, "Sent to 1 agent(s)")

		// Verify broadcast arrived in agent-2's pane.
		expected := internal.FormatMessage("agent-1", "implementer", "broadcast msg")
		h.waitForPaneContent(agent2PaneID, expected, 2*time.Second)
	})

	t.Run("06_who", func(t *testing.T) {
		require.NotEmpty(t, sessionID, "session ID not set")

		out, rc := h.runInPane(paneID, "who", "--json")
		require.Equal(t, 0, rc, "ax who --json failed: %s", out)

		var entries []struct {
			Label  string `json:"label"`
			Role   string `json:"role"`
			Pane   string `json:"pane"`
			Status string `json:"status"`
		}
		err := json.Unmarshal([]byte(out), &entries)
		require.NoError(t, err, "failed to parse who JSON output: %s", out)

		require.Len(t, entries, 2)
		for _, e := range entries {
			assert.True(t, paneIDPattern.MatchString(e.Pane), "pane field should match %%N, got %q", e.Pane)
			assert.Equal(t, "alive", e.Status)
		}
	})

	t.Run("07_whoami", func(t *testing.T) {
		out, rc := h.runInPane(paneID, "whoami", "--json")
		require.Equal(t, 0, rc, "ax whoami --json failed: %s", out)

		var entry struct {
			Label  string `json:"label"`
			Role   string `json:"role"`
			Pane   string `json:"pane"`
			Status string `json:"status"`
		}
		err := json.Unmarshal([]byte(out), &entry)
		require.NoError(t, err, "failed to parse whoami JSON output: %s", out)

		assert.Equal(t, "agent-1", entry.Label)
		assert.Equal(t, "implementer", entry.Role)
		assert.True(t, paneIDPattern.MatchString(entry.Pane), "pane field should match %%N, got %q", entry.Pane)
	})

	t.Run("08_dead_pane_send", func(t *testing.T) {
		require.NotEmpty(t, agent2PaneID, "agent-2 pane ID not set")

		// Kill agent-2's pane.
		h.killPane(agent2PaneID)

		// Sending to a dead pane should fail.
		out, rc := h.runInPane(paneID, "send", "agent-2", "should fail")
		assert.NotEqual(t, 0, rc, "ax send to dead pane should fail, got rc=0, out: %s", out)
		assert.Contains(t, out, "dead")

		// Registry should reflect the dead status.
		agents := h.readRegistry(sessionID)
		for _, a := range agents {
			if a.Label == "agent-2" {
				assert.Equal(t, internal.StatusDead, a.Status, "agent-2 should be dead in registry")
			}
		}
	})

	t.Run("09_who_after_dead", func(t *testing.T) {
		out, rc := h.runInPane(paneID, "who", "--json")
		require.Equal(t, 0, rc, "ax who --json failed: %s", out)

		var entries []struct {
			Label  string `json:"label"`
			Role   string `json:"role"`
			Pane   string `json:"pane"`
			Status string `json:"status"`
		}
		err := json.Unmarshal([]byte(out), &entries)
		require.NoError(t, err, "failed to parse who JSON output: %s", out)

		require.Len(t, entries, 2)
		for _, e := range entries {
			assert.True(t, paneIDPattern.MatchString(e.Pane), "pane field should match %%N, got %q", e.Pane)
			switch e.Label {
			case "agent-1":
				assert.Equal(t, "alive", e.Status)
			case "agent-2":
				assert.Equal(t, "dead", e.Status)
			}
		}
	})
}
