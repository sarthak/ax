package internal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// TmuxClient wraps all tmux CLI interactions.
type TmuxClient struct{}

// NewTmuxClient creates a new TmuxClient.
func NewTmuxClient() *TmuxClient {
	return &TmuxClient{}
}

// Run executes a tmux command and returns its trimmed stdout.
// Socket-aware: prepends -L <socket> when AX_TMUX_SOCKET is set.
func (t *TmuxClient) Run(args ...string) (string, error) {
	if sock := os.Getenv("AX_TMUX_SOCKET"); sock != "" {
		args = append([]string{"-L", sock}, args...)
	}
	out, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// IsInsideTmux returns true if the $TMUX env var is set (meaning we're inside
// a tmux session).
func (t *TmuxClient) IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// CurrentPaneID returns the pane ID (%N format) of the current pane.
// Prefers the TMUX_PANE env var (set per-pane by tmux) over display-message,
// because display-message returns the focused pane which may differ from the
// pane where the calling process is running.
func (t *TmuxClient) CurrentPaneID() (string, error) {
	if paneID := os.Getenv("TMUX_PANE"); paneID != "" {
		return paneID, nil
	}
	out, err := t.Run("display-message", "-p", "#{pane_id}")
	if err != nil {
		return "", fmt.Errorf("current pane id: %w", err)
	}
	return out, nil
}

// CurrentSessionName returns the name of the current tmux session.
func (t *TmuxClient) CurrentSessionName() (string, error) {
	out, err := t.Run("display-message", "-p", "#{session_name}")
	if err != nil {
		return "", fmt.Errorf("current session name: %w", err)
	}
	return out, nil
}

// PaneExists checks whether a pane ID still exists and is alive. Uses
// list-panes with a filter to find the exact pane — no fallback behavior
// unlike display-message. Returns false if the pane doesn't exist or its
// process has exited (pane_dead).
func (t *TmuxClient) PaneExists(paneID string) bool {
	out, err := t.Run("list-panes", "-a",
		"-f", fmt.Sprintf("#{==:#{pane_id},%s}", paneID),
		"-F", "#{pane_dead}")
	if err != nil {
		return false
	}
	return out == "0"
}

// GetEnv reads a tmux session-level environment variable. If session is empty,
// targets the current session.
func (t *TmuxClient) GetEnv(name string, session ...string) (string, error) {
	args := []string{"show-environment"}
	if len(session) > 0 && session[0] != "" {
		args = append(args, "-t", session[0])
	}
	args = append(args, name)
	line, err := t.Run(args...)
	if err != nil {
		return "", fmt.Errorf("get tmux env %s: %w", name, err)
	}
	_, value, found := strings.Cut(line, "=")
	if !found {
		return "", fmt.Errorf("get tmux env %s: unexpected format %q", name, line)
	}
	return value, nil
}

// SetEnv sets a tmux session-level environment variable. If session is empty,
// targets the current session.
func (t *TmuxClient) SetEnv(name, value string, session ...string) error {
	args := []string{"set-environment"}
	if len(session) > 0 && session[0] != "" {
		args = append(args, "-t", session[0])
	}
	args = append(args, name, value)
	if _, err := t.Run(args...); err != nil {
		return fmt.Errorf("set tmux env %s: %w", name, err)
	}
	return nil
}

// SendKeys injects text as literal keystrokes into the specified pane, then
// presses Enter. The literal flag (-l) prevents tmux from interpreting text
// like "Enter" or "Escape" as key names. The Enter key press is a separate
// call without -l.
func (t *TmuxClient) SendKeys(paneID string, text string) error {
	// Send the literal text.
	if _, err := t.Run("send-keys", "-t", paneID, "-l", "--", text); err != nil {
		return fmt.Errorf("send keys (text) to %s: %w", paneID, err)
	}
	// Press Enter.
	if _, err := t.Run("send-keys", "-t", paneID, "Enter"); err != nil {
		return fmt.Errorf("send keys (enter) to %s: %w", paneID, err)
	}
	return nil
}

// FormatMessage composes an ax protocol message string.
// Format: [ax sms from <senderLabel> (<senderRole>)]: <message>
func FormatMessage(senderLabel, senderRole, message string) string {
	return fmt.Sprintf("[ax sms from %s (%s)]: %s", senderLabel, senderRole, message)
}

// SendFormattedMessage composes and sends an ax protocol message.
func (t *TmuxClient) SendFormattedMessage(paneID, senderLabel, senderRole, message string) error {
	formatted := FormatMessage(senderLabel, senderRole, message)
	return t.SendKeys(paneID, formatted)
}

// NewSession creates a new detached tmux session with the given name.
// It does NOT attach — the caller handles attachment separately.
func (t *TmuxClient) NewSession(name string) error {
	if _, err := t.Run("new-session", "-d", "-s", name); err != nil {
		return fmt.Errorf("new session %s: %w", name, err)
	}
	return nil
}

// SplitH creates a horizontal split (new pane to the right) and returns the
// new pane ID.
func (t *TmuxClient) SplitH() (string, error) {
	out, err := t.Run("split-window", "-h", "-P", "-F", "#{pane_id}")
	if err != nil {
		return "", fmt.Errorf("split horizontal: %w", err)
	}
	return out, nil
}

// SplitV creates a vertical split (new pane below) and returns the new pane ID.
func (t *TmuxClient) SplitV() (string, error) {
	out, err := t.Run("split-window", "-v", "-P", "-F", "#{pane_id}")
	if err != nil {
		return "", fmt.Errorf("split vertical: %w", err)
	}
	return out, nil
}

// NewWindow creates a new tmux window and returns the pane ID of its initial
// pane.
func (t *TmuxClient) NewWindow() (string, error) {
	out, err := t.Run("new-window", "-P", "-F", "#{pane_id}")
	if err != nil {
		return "", fmt.Errorf("new window: %w", err)
	}
	return out, nil
}

// CloseOnExit configures a pane to close when its process exits.
func (t *TmuxClient) CloseOnExit(paneID string) error {
	return t.setPaneOption(paneID, "remain-on-exit", "off")
}

// setPaneOption sets a pane-level tmux option. The -p flag is required to
// target the pane itself; without it tmux interprets the target as a session.
func (t *TmuxClient) setPaneOption(paneID, option, value string) error {
	if _, err := t.Run("set-option", "-p", "-t", paneID, option, value); err != nil {
		return fmt.Errorf("set pane option %s on %s: %w", option, paneID, err)
	}
	return nil
}

// AttachSession replaces the current process with tmux attach-session.
// Socket-aware via AX_TMUX_SOCKET. Does not return on success.
func (t *TmuxClient) AttachSession(name string) error {
	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("find tmux binary: %w", err)
	}
	args := []string{"tmux"}
	if sock := os.Getenv("AX_TMUX_SOCKET"); sock != "" {
		args = append(args, "-L", sock)
	}
	args = append(args, "attach-session", "-t", name)
	return syscall.Exec(tmuxPath, args, os.Environ())
}
