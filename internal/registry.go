package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"time"
)

type AgentStatus string

const (
	StatusAlive AgentStatus = "alive"
	StatusDead  AgentStatus = "dead"
)

type Agent struct {
	Label      string      `json:"label"`
	Role       string      `json:"role"`
	TmuxPaneID string      `json:"tmux_pane_id"`
	Status     AgentStatus `json:"status"`
	CommsDir   string      `json:"comms_dir"`
	JoinedAt   time.Time   `json:"joined_at"`
}

// Registry provides read/write access to a session's registry.json.
// Load() reads from disk on first call, then caches internally.
// Mutating methods (Upsert, Save, UpdateStatus) invalidate the cache.
type Registry struct {
	path   string
	agents []Agent
	loaded bool
}

func NewRegistry(path string) *Registry {
	return &Registry{path: path}
}

// Load reads and parses registry.json. On first call reads from disk; subsequent calls
// return the cached slice (until a mutation invalidates the cache).
func (r *Registry) Load() ([]Agent, error) {
	if r.loaded {
		return r.agents, nil
	}

	data, err := os.ReadFile(r.path)
	if err != nil {
		return nil, fmt.Errorf("load registry: %w", err)
	}

	var agents []Agent
	if err := json.Unmarshal(data, &agents); err != nil {
		return nil, fmt.Errorf("load registry: %w", err)
	}

	r.agents = agents
	r.loaded = true
	return r.agents, nil
}

// Save writes agents to registry.json atomically (write to temp file, then os.Rename).
// Refreshes the internal cache with the saved data.
func (r *Registry) Save(agents []Agent) error {
	data, err := json.MarshalIndent(agents, "", "  ")
	if err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	tmpPath := r.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	if err := os.Rename(tmpPath, r.path); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	r.agents = agents
	r.loaded = true
	return nil
}

// lockPath returns the path of the lock file used for exclusive access.
func (r *Registry) lockPath() string {
	return r.path + ".lock"
}

// withLock acquires an exclusive file lock, runs fn, then releases the lock.
// A dedicated lock file (registry.json.lock) is used so that atomic renames of
// the data file don't invalidate the lock.
func (r *Registry) withLock(fn func() error) error {
	lf, err := os.OpenFile(r.lockPath(), os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}
	defer lf.Close()

	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquire lock: %w", err)
	}
	defer func() { _ = syscall.Flock(int(lf.Fd()), syscall.LOCK_UN) }()

	return fn()
}

// lockedReadModifyWrite reads registry.json under an exclusive lock, passes
// the agents to mutate, and atomically writes the result via Save.
func (r *Registry) lockedReadModifyWrite(mutate func([]Agent) ([]Agent, error)) error {
	return r.withLock(func() error {
		data, err := os.ReadFile(r.path)
		if err != nil {
			return err
		}

		var agents []Agent
		if len(data) > 0 {
			if err := json.Unmarshal(data, &agents); err != nil {
				return err
			}
		}

		agents, err = mutate(agents)
		if err != nil {
			return err
		}

		return r.Save(agents)
	})
}

// Upsert adds or updates an agent by label. If label exists, replaces it.
// If not, appends. Uses file locking to prevent concurrent corruption.
// Invalidates cache.
func (r *Registry) Upsert(agent Agent) error {
	return r.lockedReadModifyWrite(func(agents []Agent) ([]Agent, error) {
		for i, a := range agents {
			if a.Label == agent.Label {
				agents[i] = agent
				return agents, nil
			}
		}
		return append(agents, agent), nil
	})
}

// UpdateStatus changes the status of an agent by label. Used by `ax send` to mark
// dead panes. Uses file locking. Invalidates cache.
func (r *Registry) UpdateStatus(label string, status AgentStatus) error {
	return r.lockedReadModifyWrite(func(agents []Agent) ([]Agent, error) {
		for i, a := range agents {
			if a.Label == label {
				agents[i].Status = status
				return agents, nil
			}
		}
		return nil, fmt.Errorf("update status: agent %q not found", label)
	})
}

// FindByLabel returns the agent with the given label, or nil if not found.
// Searches cached agents (calls Load() if not yet loaded).
func (r *Registry) FindByLabel(label string) *Agent {
	agents, err := r.Load()
	if err != nil {
		return nil
	}
	for i, a := range agents {
		if a.Label == label {
			return &agents[i]
		}
	}
	return nil
}

// FindByRole returns all agents matching the given role.
// Searches cached agents (calls Load() if not yet loaded).
func (r *Registry) FindByRole(role string) []Agent {
	agents, err := r.Load()
	if err != nil {
		return nil
	}
	var result []Agent
	for _, a := range agents {
		if a.Role == role {
			result = append(result, a)
		}
	}
	return result
}

// FindByPaneID returns the agent running in the given tmux pane, or nil.
// Searches cached agents (calls Load() if not yet loaded).
func (r *Registry) FindByPaneID(paneID string) *Agent {
	agents, err := r.Load()
	if err != nil {
		return nil
	}
	for i, a := range agents {
		if a.TmuxPaneID == paneID {
			return &agents[i]
		}
	}
	return nil
}
