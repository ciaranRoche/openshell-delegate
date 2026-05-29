package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ciaranRoche/openshell-delegate/internal/config"
)

const stateFile = "state.json"

// Delegation tracks an active delegation.
type Delegation struct {
	Sandbox  string    `json:"sandbox"`
	Repo     string    `json:"repo"`
	Worktree string    `json:"worktree"`
	Branch   string    `json:"branch"`
	Created  time.Time `json:"created"`
}

// State holds all active delegations.
type State struct {
	Delegations map[string]Delegation `json:"delegations"`
}

func statePath() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, stateFile), nil
}

// Load reads the state file. Returns empty state if file doesn't exist.
func Load() (*State, error) {
	path, err := statePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Delegations: make(map[string]Delegation)}, nil
		}
		return nil, fmt.Errorf("could not read state: %w", err)
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("could not parse state: %w", err)
	}
	if s.Delegations == nil {
		s.Delegations = make(map[string]Delegation)
	}
	return &s, nil
}

// Save writes the state file to disk.
func Save(s *State) error {
	path, err := statePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

// Get returns a delegation by branch name.
func (s *State) Get(branch string) (Delegation, bool) {
	d, ok := s.Delegations[branch]
	return d, ok
}

// Set adds or updates a delegation.
func (s *State) Set(branch string, d Delegation) {
	s.Delegations[branch] = d
}

// Remove deletes a delegation by branch name.
func (s *State) Remove(branch string) {
	delete(s.Delegations, branch)
}
