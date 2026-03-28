package internal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// TmuxClient wraps all tmux CLI interactions.
type TmuxClient struct{}

// tmuxOutput runs a tmux command and returns its trimmed stdout.
func tmuxOutput(args ...string) (string, error) {
	out, err := exec.Command("tmux", args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// NewTmuxClient creates a new TmuxClient.
func NewTmuxClient() *TmuxClient {
	return &TmuxClient{}
}

// IsInsideTmux returns true if the $TMUX env var is set (meaning we're inside
// a tmux session).
func (t *TmuxClient) IsInsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// CurrentPaneID returns the pane ID (%N format) of the current pane.
func (t *TmuxClient) CurrentPaneID() (string, error) {
	out, err := tmuxOutput("display-message", "-p", "#{pane_id}")
	if err != nil {
		return "", fmt.Errorf("current pane id: %w", err)
	}
	return out, nil
}

// CurrentSessionName returns the name of the current tmux session.
func (t *TmuxClient) CurrentSessionName() (string, error) {
	out, err := tmuxOutput("display-message", "-p", "#{session_name}")
	if err != nil {
		return "", fmt.Errorf("current session name: %w", err)
	}
	return out, nil
}

// PaneExists checks whether a pane ID still exists in the tmux server.
func (t *TmuxClient) PaneExists(paneID string) bool {
	_, err := tmuxOutput("display-message", "-t", paneID, "-p", "")
	return err == nil
}

// GetEnv reads a tmux session-level environment variable.
// The output from tmux is in format "NAME=value".
func (t *TmuxClient) GetEnv(name string) (string, error) {
	line, err := tmuxOutput("show-environment", name)
	if err != nil {
		return "", fmt.Errorf("get tmux env %s: %w", name, err)
	}
	_, value, found := strings.Cut(line, "=")
	if !found {
		return "", fmt.Errorf("get tmux env %s: unexpected format %q", name, line)
	}
	return value, nil
}

// SetEnv sets a tmux session-level environment variable.
func (t *TmuxClient) SetEnv(name, value string) error {
	if err := exec.Command("tmux", "set-environment", name, value).Run(); err != nil {
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
	cmd := exec.Command("tmux", "send-keys", "-t", paneID, "-l", "--", text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("send keys (text) to %s: %w", paneID, err)
	}
	// Press Enter.
	cmd = exec.Command("tmux", "send-keys", "-t", paneID, "Enter")
	if err := cmd.Run(); err != nil {
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
	if err := exec.Command("tmux", "new-session", "-d", "-s", name).Run(); err != nil {
		return fmt.Errorf("new session %s: %w", name, err)
	}
	return nil
}

// SplitH creates a horizontal split (new pane to the right) and returns the
// new pane ID.
func (t *TmuxClient) SplitH() (string, error) {
	out, err := tmuxOutput("split-window", "-h", "-P", "-F", "#{pane_id}")
	if err != nil {
		return "", fmt.Errorf("split horizontal: %w", err)
	}
	return out, nil
}

// SplitV creates a vertical split (new pane below) and returns the new pane ID.
func (t *TmuxClient) SplitV() (string, error) {
	out, err := tmuxOutput("split-window", "-v", "-P", "-F", "#{pane_id}")
	if err != nil {
		return "", fmt.Errorf("split vertical: %w", err)
	}
	return out, nil
}

// NewWindow creates a new tmux window and returns the pane ID of its initial
// pane.
func (t *TmuxClient) NewWindow() (string, error) {
	out, err := tmuxOutput("new-window", "-P", "-F", "#{pane_id}")
	if err != nil {
		return "", fmt.Errorf("new window: %w", err)
	}
	return out, nil
}

// CloseOnExit configures a pane to close when its process exits.
func (t *TmuxClient) CloseOnExit(paneID string) error {
	return t.setPaneOption(paneID, "remain-on-exit", "off")
}

// setPaneOption sets a pane-level tmux option.
func (t *TmuxClient) setPaneOption(paneID, option, value string) error {
	if err := exec.Command("tmux", "set-option", "-t", paneID, option, value).Run(); err != nil {
		return fmt.Errorf("set pane option %s on %s: %w", option, paneID, err)
	}
	return nil
}
