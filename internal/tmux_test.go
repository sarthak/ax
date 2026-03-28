// Integration tests requiring tmux are in tmux_integration_test.go (build tag: integration)

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTmuxClient(t *testing.T) {
	client := NewTmuxClient()
	require.NotNil(t, client)
}

func TestIsInsideTmux_Set(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	client := NewTmuxClient()
	assert.True(t, client.IsInsideTmux())
}

func TestIsInsideTmux_Unset(t *testing.T) {
	t.Setenv("TMUX", "")
	client := NewTmuxClient()
	assert.False(t, client.IsInsideTmux())
}

func TestFormatMessage_Basic(t *testing.T) {
	got := FormatMessage("alice", "engineer", "hello world")
	assert.Equal(t, "[ax sms from alice (engineer)]: hello world", got)
}

func TestFormatMessage_SpecialChars(t *testing.T) {
	tests := []struct {
		name        string
		senderLabel string
		senderRole  string
		message     string
		want        string
	}{
		{
			name:        "dollar sign",
			senderLabel: "bot",
			senderRole:  "assistant",
			message:     "$HOME is /Users/me",
			want:        "[ax sms from bot (assistant)]: $HOME is /Users/me",
		},
		{
			name:        "backticks",
			senderLabel: "bot",
			senderRole:  "assistant",
			message:     "run `ls -la`",
			want:        "[ax sms from bot (assistant)]: run `ls -la`",
		},
		{
			name:        "single quotes",
			senderLabel: "bot",
			senderRole:  "assistant",
			message:     "it's a test",
			want:        "[ax sms from bot (assistant)]: it's a test",
		},
		{
			name:        "brackets",
			senderLabel: "bot",
			senderRole:  "assistant",
			message:     "array[0] = {value}",
			want:        "[ax sms from bot (assistant)]: array[0] = {value}",
		},
		{
			name:        "empty message",
			senderLabel: "bot",
			senderRole:  "assistant",
			message:     "",
			want:        "[ax sms from bot (assistant)]: ",
		},
		{
			name:        "label with spaces",
			senderLabel: "Agent One",
			senderRole:  "lead engineer",
			message:     "status update",
			want:        "[ax sms from Agent One (lead engineer)]: status update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatMessage(tt.senderLabel, tt.senderRole, tt.message)
			assert.Equal(t, tt.want, got)
		})
	}
}
