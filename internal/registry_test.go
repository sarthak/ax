package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	path := filepath.Join(t.TempDir(), "registry.json")
	require.NoError(t, os.WriteFile(path, []byte("[]"), 0644))
	return NewRegistry(path)
}

var testTime = time.Now().Truncate(time.Second)

func TestRegistryCRUD(t *testing.T) {
	reg := newTestRegistry(t)

	t.Run("load empty", func(t *testing.T) {
		agents, err := reg.Load()
		require.NoError(t, err)
		assert.Empty(t, agents)
	})

	t.Run("save and load", func(t *testing.T) {
		agents := []Agent{
			{Label: "agent-1", Role: "coder", TmuxPaneID: "%1", Status: StatusAlive, CommsDir: "/tmp/comms/agent-1", JoinedAt: testTime},
			{Label: "agent-2", Role: "reviewer", TmuxPaneID: "%2", Status: StatusAlive, CommsDir: "/tmp/comms/agent-2", JoinedAt: testTime.Add(time.Minute)},
		}
		require.NoError(t, reg.Save(agents))

		loaded, err := NewRegistry(reg.path).Load()
		require.NoError(t, err)
		assert.Equal(t, agents, loaded)
	})

	t.Run("upsert insert", func(t *testing.T) {
		require.NoError(t, reg.Upsert(Agent{Label: "agent-3", Role: "coder", TmuxPaneID: "%5", Status: StatusAlive, JoinedAt: testTime}))

		loaded, err := reg.Load()
		require.NoError(t, err)
		require.Len(t, loaded, 3)
		assert.Equal(t, "agent-3", loaded[2].Label)
	})

	t.Run("upsert update replaces and does not duplicate", func(t *testing.T) {
		require.NoError(t, reg.Upsert(Agent{Label: "agent-3", Role: "reviewer", TmuxPaneID: "%20", Status: StatusAlive, JoinedAt: testTime}))

		loaded, err := reg.Load()
		require.NoError(t, err)
		require.Len(t, loaded, 3)
		assert.Equal(t, "reviewer", loaded[2].Role)
		assert.Equal(t, "%20", loaded[2].TmuxPaneID)
	})

	t.Run("update status", func(t *testing.T) {
		require.NoError(t, reg.UpdateStatus("agent-3", StatusDead))

		loaded, err := reg.Load()
		require.NoError(t, err)
		assert.Equal(t, StatusDead, loaded[2].Status)
	})

	t.Run("update status not found", func(t *testing.T) {
		assert.Error(t, reg.UpdateStatus("nonexistent", StatusDead))
	})
}

func TestRegistryQueries(t *testing.T) {
	reg := newTestRegistry(t)
	require.NoError(t, reg.Save([]Agent{
		{Label: "alpha", Role: "coder", TmuxPaneID: "%1", Status: StatusAlive, JoinedAt: testTime},
		{Label: "beta", Role: "coder", TmuxPaneID: "%2", Status: StatusAlive, JoinedAt: testTime},
		{Label: "gamma", Role: "reviewer", TmuxPaneID: "%3", Status: StatusAlive, JoinedAt: testTime},
	}))

	t.Run("find by label", func(t *testing.T) {
		assert.Equal(t, "alpha", reg.FindByLabel("alpha").Label)
		assert.Nil(t, reg.FindByLabel("missing"))
	})

	t.Run("find by role", func(t *testing.T) {
		assert.Len(t, reg.FindByRole("coder"), 2)
		assert.Len(t, reg.FindByRole("reviewer"), 1)
		assert.Empty(t, reg.FindByRole("manager"))
	})

	t.Run("find by pane ID", func(t *testing.T) {
		assert.Equal(t, "alpha", reg.FindByPaneID("%1").Label)
		assert.Nil(t, reg.FindByPaneID("%99"))
	})
}

func TestRegistryCaching(t *testing.T) {
	reg := newTestRegistry(t)
	require.NoError(t, reg.Save([]Agent{
		{Label: "orig", Role: "coder", TmuxPaneID: "%1", Status: StatusAlive, JoinedAt: testTime},
	}))

	t.Run("load returns cached data", func(t *testing.T) {
		agents1, err := reg.Load()
		require.NoError(t, err)
		require.Len(t, agents1, 1)

		// Modify file directly behind the registry's back
		extra := []Agent{agents1[0], {Label: "sneaky", Role: "hacker", TmuxPaneID: "%99", Status: StatusAlive, JoinedAt: testTime}}
		data, _ := json.MarshalIndent(extra, "", "  ")
		require.NoError(t, os.WriteFile(reg.path, data, 0644))

		agents2, err := reg.Load()
		require.NoError(t, err)
		assert.Len(t, agents2, 1, "should return cached data, not re-read from disk")
	})

	t.Run("upsert invalidates cache", func(t *testing.T) {
		require.NoError(t, reg.Upsert(Agent{Label: "new", Role: "coder", TmuxPaneID: "%2", Status: StatusAlive, JoinedAt: testTime}))

		agents, err := reg.Load()
		require.NoError(t, err)
		// Should see both "sneaky" (from direct file write above) and "new" (from upsert),
		// since upsert re-reads from disk under lock
		assert.Len(t, agents, 3)
	})
}

func TestRegistryConcurrentWrites(t *testing.T) {
	reg := newTestRegistry(t)
	const n = 20

	var wg sync.WaitGroup
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Each goroutine uses its own Registry instance to test file-level locking
			r := NewRegistry(reg.path)
			if err := r.Upsert(Agent{
				Label: fmt.Sprintf("agent-%d", i), Role: "worker",
				TmuxPaneID: fmt.Sprintf("%%%d", i), Status: StatusAlive, JoinedAt: testTime,
			}); err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	data, err := os.ReadFile(reg.path)
	require.NoError(t, err)
	var agents []Agent
	require.NoError(t, json.Unmarshal(data, &agents))
	require.Len(t, agents, n)

	labels := make(map[string]bool)
	for _, a := range agents {
		labels[a.Label] = true
	}
	for i := 0; i < n; i++ {
		assert.True(t, labels[fmt.Sprintf("agent-%d", i)], "missing agent-%d", i)
	}
}
