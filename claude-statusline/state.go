package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type (
	state struct {
		status      Status
		stateDir    string
		sessionsDir string
		session     session
	}

	session struct {
		CreatedAt    time.Time
		ChangedFiles struct {
			UpdatedAt time.Time
			Counts    changedFiles
		}
	}
)

func newState(status Status) (*state, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	stateDir := filepath.Join(homeDir, ".local", "state", "5xx.engineer", "claude-statusline")
	if err = mkWritableDir(stateDir); err != nil {
		return nil, err
	}
	sessionsDir := filepath.Join(stateDir, "sessions")
	if err = mkWritableDir(sessionsDir); err != nil {
		return nil, err
	}
	s := &state{
		status:      status,
		stateDir:    stateDir,
		sessionsDir: sessionsDir,
		session: session{
			CreatedAt: time.Now(),
		},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *state) sessionDuration() time.Duration {
	return time.Since(s.session.CreatedAt)
}

func (s *state) load() error {
	path := s.sessionPath()
	if path == "" {
		return nil
	}
	bs, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil
	case err != nil:
		return err
	}
	err = json.NewDecoder(bytes.NewReader(bs)).Decode(&s.session)
	if err != nil {
		return fmt.Errorf("decode session %s: %w", path, err)
	}
	return nil
}

func (s *state) close() error {
	path := s.sessionPath()
	if path == "" {
		return nil
	}
	bs, err := json.Marshal(s.session)
	if err != nil {
		return err
	}
	return os.WriteFile(path, bs, 0644)
}

func (s *state) sessionPath() string {
	if s.status.SessionID == "" {
		return ""
	}
	name := fmt.Sprintf("%s.json", s.status.SessionID)
	return filepath.Join(s.sessionsDir, name)
}

func mkWritableDir(path string) error {
	if err := os.MkdirAll(path, 0744); err != nil {
		return err
	}
	return os.Chmod(path, 0744)
}
