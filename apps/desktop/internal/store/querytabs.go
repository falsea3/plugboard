package store

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (s *Scripts) Tabs(conn string) ([]model.QueryTab, error) {
	dir, err := s.folder(conn)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	tabs := []model.QueryTab{}
	data, err := os.ReadFile(filepath.Join(dir, tabsFile))
	if errors.Is(err, fs.ErrNotExist) {
		return tabs, nil
	}
	if err != nil {
		return nil, err
	}
	if json.Unmarshal(data, &tabs) != nil {
		return []model.QueryTab{}, nil
	}
	for i := range tabs {
		if tabs[i].Script == "" {
			continue
		}
		text, err := os.ReadFile(filepath.Join(dir, tabs[i].Script+scriptExt))
		if CheckScriptName(tabs[i].Script) != nil || err != nil {
			tabs[i].Script, tabs[i].Saved = "", ""
			continue
		}
		tabs[i].Saved = string(text)
	}
	return tabs, nil
}

func (s *Scripts) SaveTabs(conn string, tabs []model.QueryTab) error {
	dir, err := s.folder(conn)
	if err != nil {
		return err
	}
	kept := make([]model.QueryTab, len(tabs))
	for i, t := range tabs {
		if t.Script != "" {
			t.Saved = ""
		}
		kept[i] = t
	}
	data, err := json.MarshalIndent(kept, "", "  ")
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeFileAtomic(filepath.Join(dir, tabsFile), data, 0o600)
}

func (s *Scripts) ForgetTabs(conn string) error {
	dir, err := s.folder(conn)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(filepath.Join(dir, tabsFile)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
