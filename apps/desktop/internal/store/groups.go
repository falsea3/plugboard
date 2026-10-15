package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

type connFile struct {
	Version     int                     `json:"version"`
	Connections []model.Connection      `json:"connections"`
	Groups      []model.ConnectionGroup `json:"groups"`
}

func (s *Connections) Groups() ([]model.ConnectionGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadFile()
	return f.Groups, err
}

func (s *Connections) Arrange(a model.Arrangement) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadFile()
	if err != nil {
		return err
	}
	byID := map[string]model.Connection{}
	for _, c := range f.Connections {
		byID[c.ID] = c
	}
	arranged := make([]model.Connection, 0, len(f.Connections))
	for _, p := range a.Connections {
		c, ok := byID[p.ID]
		if !ok {
			continue
		}
		delete(byID, p.ID)
		c.Group = strings.TrimSpace(p.Group)
		arranged = append(arranged, c)
	}
	for _, c := range f.Connections {
		if _, left := byID[c.ID]; left {
			arranged = append(arranged, c)
		}
	}
	f.Connections = arranged
	f.Groups = a.Groups
	return s.write(f)
}

func (s *Connections) Touch(id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.loadFile()
	if err != nil {
		return err
	}
	i := slices.IndexFunc(f.Connections, func(c model.Connection) bool { return c.ID == id })
	if i < 0 {
		return nil
	}
	f.Connections[i].LastUsed = at.UTC().Format(time.RFC3339)
	return s.write(f)
}

func (s *Connections) load() ([]model.Connection, error) {
	f, err := s.loadFile()
	return f.Connections, err
}

func (s *Connections) loadFile() (connFile, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return connFile{Connections: []model.Connection{}, Groups: []model.ConnectionGroup{}}, nil
	}
	if err != nil {
		return connFile{}, err
	}
	var f connFile
	if err := json.Unmarshal(data, &f); err != nil {
		return connFile{}, fmt.Errorf("parse %s: %w", s.path, err)
	}
	if f.Connections == nil {
		f.Connections = []model.Connection{}
	}
	f.Groups = tidyGroups(f.Groups, f.Connections)
	return f, nil
}

func (s *Connections) write(f connFile) error {
	f.Version = 2
	f.Groups = tidyGroups(f.Groups, f.Connections)
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(s.path, data, 0o600)
}

func tidyGroups(groups []model.ConnectionGroup, conns []model.Connection) []model.ConnectionGroup {
	out := []model.ConnectionGroup{}
	seen := map[string]bool{}
	add := func(g model.ConnectionGroup) {
		g.Name = strings.TrimSpace(g.Name)
		if g.Name == "" || seen[g.Name] {
			return
		}
		seen[g.Name] = true
		out = append(out, g)
	}
	for _, g := range groups {
		add(g)
	}
	for _, c := range conns {
		add(model.ConnectionGroup{Name: c.Group})
	}
	return slices.Clip(out)
}
