package internal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// SessionPaths holds the filesystem paths for a session.
type SessionPaths struct {
	Root     string // ~/.local/state/ax/sessions/<id>
	Registry string // Root + "/registry.json"
	Comms    string // Root + "/comms"
}

// GenerateSessionID returns a 6-character random hex string (3 bytes from
// crypto/rand, hex-encoded).
func GenerateSessionID() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate session id: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// ResolveStateDir returns the base state directory, following the XDG Base
// Directory Specification for application state ($XDG_STATE_HOME, defaulting
// to ~/.local/state). Go's stdlib provides os.UserConfigDir and os.UserCacheDir
// but has no equivalent for the state directory, so we resolve it manually.
//
// Precedence: $AX_STATE_DIR (ax-specific override) → $XDG_STATE_HOME/ax
// (XDG standard) → ~/.local/state/ax (XDG default).
func ResolveStateDir() (string, error) {
	if dir := os.Getenv("AX_STATE_DIR"); dir != "" {
		return dir, nil
	}
	if dir := os.Getenv("XDG_STATE_HOME"); dir != "" {
		return filepath.Join(dir, "ax"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve state dir: %w", err)
	}
	return filepath.Join(home, ".local", "state", "ax"), nil
}

// InitSessionDir creates the full directory tree for a new session and returns
// paths. It creates <stateDir>/sessions/<id>/, writes an empty JSON array []
// to registry.json, and creates a comms/ subdirectory.
func InitSessionDir(sessionID string) (*SessionPaths, error) {
	stateDir, err := ResolveStateDir()
	if err != nil {
		return nil, err
	}

	// Unix permission bits: 0755 = rwxr-xr-x (owner: full, group/others: read+execute).
	// 0644 = rw-r--r-- (owner: read+write, group/others: read-only).
	// Directories need the execute bit for traversal.
	root := filepath.Join(stateDir, "sessions", sessionID)
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}

	registryPath := filepath.Join(root, "registry.json")
	if err := os.WriteFile(registryPath, []byte("[]"), 0644); err != nil {
		return nil, fmt.Errorf("write registry.json: %w", err)
	}

	commsPath := filepath.Join(root, "comms")
	if err := os.MkdirAll(commsPath, 0755); err != nil {
		return nil, fmt.Errorf("create comms dir: %w", err)
	}

	return &SessionPaths{
		Root:     root,
		Registry: registryPath,
		Comms:    commsPath,
	}, nil
}

// LoadSessionPaths resolves paths for an existing session without creating
// anything. It returns an error if the session directory does not exist.
func LoadSessionPaths(sessionID string) (*SessionPaths, error) {
	stateDir, err := ResolveStateDir()
	if err != nil {
		return nil, err
	}

	root := filepath.Join(stateDir, "sessions", sessionID)
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("load session paths: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("load session paths: %s is not a directory", root)
	}

	return &SessionPaths{
		Root:     root,
		Registry: filepath.Join(root, "registry.json"),
		Comms:    filepath.Join(root, "comms"),
	}, nil
}
