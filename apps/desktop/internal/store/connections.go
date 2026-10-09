package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

type Connections struct {
	mu      sync.Mutex
	path    string
	secrets Secrets
}

func NewConnections(dir string, secrets Secrets) *Connections {
	return &Connections{path: filepath.Join(dir, "connections.json"), secrets: secrets}
}

func (s *Connections) List() ([]model.Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *Connections) Get(id string) (model.Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok, err := s.find(id)
	if err != nil {
		return model.Connection{}, err
	}
	if !ok {
		return model.Connection{}, fmt.Errorf("connection %s not found", id)
	}
	if err := s.readSecrets(&c); err != nil {
		return model.Connection{}, err
	}
	return c, nil
}

func (s *Connections) Save(c model.Connection) (model.Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := Validate(c); err != nil {
		return model.Connection{}, err
	}
	all, err := s.load()
	if err != nil {
		return model.Connection{}, err
	}
	if c.ID == "" {
		c.ID = model.NewID()
	}
	typed := make([]string, len(secretFields))
	for i, f := range secretFields {
		typed[i] = *f.field(&c)
		*f.field(&c) = ""
	}

	var old *model.Connection
	if i := slices.IndexFunc(all, func(p model.Connection) bool { return p.ID == c.ID }); i >= 0 {
		prev := all[i]
		old = &prev
		all[i] = c
	} else {
		all = append(all, c)
	}

	for i, f := range secretFields {
		key := secretKey(c.ID, f.slot)
		var err error
		switch {
		case c.SavePassword && typed[i] != "":
			err = s.secrets.Set(key, typed[i])
		case !c.SavePassword || (old != nil && f.target(*old) != f.target(c)):
			err = s.secrets.Delete(key)
		}
		if err != nil && !errors.Is(err, ErrSecretNotFound) {
			return model.Connection{}, fmt.Errorf("save %s: %w", f.name, err)
		}
	}
	if err := s.write(all); err != nil {
		return model.Connection{}, err
	}
	return c, nil
}

func (s *Connections) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return err
	}
	kept := slices.DeleteFunc(all, func(c model.Connection) bool { return c.ID == id })
	for _, f := range secretFields {
		if err := s.secrets.Delete(secretKey(id, f.slot)); err != nil && !errors.Is(err, ErrSecretNotFound) {
			return err
		}
	}
	return s.write(kept)
}

func (s *Connections) find(id string) (model.Connection, bool, error) {
	all, err := s.load()
	if err != nil {
		return model.Connection{}, false, err
	}
	for _, c := range all {
		if c.ID == id {
			return c, true, nil
		}
	}
	return model.Connection{}, false, nil
}

func (s *Connections) load() ([]model.Connection, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []model.Connection{}, nil
	}
	if err != nil {
		return nil, err
	}
	var file struct {
		Version     int                `json:"version"`
		Connections []model.Connection `json:"connections"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse %s: %w", s.path, err)
	}
	if file.Connections == nil {
		file.Connections = []model.Connection{}
	}
	return file.Connections, nil
}

func (s *Connections) write(all []model.Connection) error {
	slices.SortStableFunc(all, func(a, b model.Connection) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	data, err := json.MarshalIndent(map[string]any{"version": 1, "connections": all}, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(s.path, data, 0o600)
}

func Validate(c model.Connection) error {
	if strings.TrimSpace(c.Name) == "" {
		return errors.New("name is required")
	}
	switch c.Driver {
	case model.Postgres, model.MySQL, model.Redis, model.ClickHouse, model.Cassandra:
		if strings.TrimSpace(c.Host) == "" {
			return errors.New("host is required")
		}
	case model.SQLite:
		if strings.TrimSpace(c.File) == "" {
			return errors.New("database file is required")
		}
	default:
		return fmt.Errorf("unsupported driver %q", c.Driver)
	}
	if c.SSH.Enabled && c.Driver != model.SQLite {
		if strings.TrimSpace(c.SSH.User) == "" {
			return errors.New("SSH user is required")
		}
		switch c.SSH.Auth {
		case model.SSHAuthPassword, model.SSHAuthAgent:
		case model.SSHAuthKey:
			if strings.TrimSpace(c.SSH.KeyFile) == "" {
				return errors.New("choose an SSH private key file")
			}
		default:
			return fmt.Errorf("unsupported SSH authentication %q", c.SSH.Auth)
		}
	}
	return nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
