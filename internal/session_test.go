package internal

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSessionID(t *testing.T) {
	t.Run("returns 6-char hex string", func(t *testing.T) {
		id, err := GenerateSessionID()
		require.NoError(t, err)
		assert.Len(t, id, 6)
		assert.Regexp(t, regexp.MustCompile(`^[0-9a-f]{6}$`), id)
	})

	t.Run("two calls produce different IDs", func(t *testing.T) {
		id1, err := GenerateSessionID()
		require.NoError(t, err)
		id2, err := GenerateSessionID()
		require.NoError(t, err)
		assert.NotEqual(t, id1, id2)
	})
}

func TestResolveStateDir(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name         string
		axStateDir   string
		xdgStateHome string
		want         string
	}{
		{
			name: "default path uses HOME/.local/state/ax",
			want: filepath.Join(home, ".local", "state", "ax"),
		},
		{
			name:         "XDG_STATE_HOME used when AX_STATE_DIR unset",
			xdgStateHome: "/tmp/xdg-state",
			want:         "/tmp/xdg-state/ax",
		},
		{
			name:         "AX_STATE_DIR wins over XDG_STATE_HOME",
			axStateDir:   "/tmp/ax-wins",
			xdgStateHome: "/tmp/xdg-loses",
			want:         "/tmp/ax-wins",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AX_STATE_DIR", tt.axStateDir)
			t.Setenv("XDG_STATE_HOME", tt.xdgStateHome)

			dir, err := ResolveStateDir()
			require.NoError(t, err)
			assert.Equal(t, tt.want, dir)
		})
	}
}

func TestInitSessionDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("AX_STATE_DIR", tmpDir)

	paths, err := InitSessionDir("abc123")
	require.NoError(t, err)

	expectedRoot := filepath.Join(tmpDir, "sessions", "abc123")
	assert.Equal(t, expectedRoot, paths.Root)
	assert.DirExists(t, paths.Root)
	assert.DirExists(t, paths.Comms)

	data, err := os.ReadFile(paths.Registry)
	require.NoError(t, err)
	assert.Equal(t, "[]", string(data))
}

func TestLoadSessionPaths(t *testing.T) {
	t.Run("works for existing session", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("AX_STATE_DIR", tmpDir)

		_, err := InitSessionDir("aaa111")
		require.NoError(t, err)

		paths, err := LoadSessionPaths("aaa111")
		require.NoError(t, err)

		expectedRoot := filepath.Join(tmpDir, "sessions", "aaa111")
		assert.Equal(t, expectedRoot, paths.Root)
		assert.Equal(t, filepath.Join(expectedRoot, "registry.json"), paths.Registry)
		assert.Equal(t, filepath.Join(expectedRoot, "comms"), paths.Comms)
	})

	t.Run("errors for nonexistent session", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("AX_STATE_DIR", tmpDir)

		_, err := LoadSessionPaths("nonexistent")
		assert.Error(t, err)
	})
}
