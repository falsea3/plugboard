package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

type Settings struct {
	mu   sync.Mutex
	path string
}

func NewSettings(dir string) *Settings {
	return &Settings{path: filepath.Join(dir, "settings.json")}
}

func (s *Settings) Load() model.Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := model.DefaultSettings()
	data, err := os.ReadFile(s.path)
	if err != nil {
		return out
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return model.DefaultSettings()
	}
	return Sanitize(out)
}

func (s *Settings) Save(v model.Settings) (model.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v = Sanitize(v)
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return v, err
	}
	if err := writeFileAtomic(s.path, data, 0o600); err != nil {
		return v, fmt.Errorf("save settings: %w", err)
	}
	return v, nil
}

func Sanitize(v model.Settings) model.Settings {
	d := model.DefaultSettings()
	switch v.Theme {
	case model.ThemeSystem, model.ThemeLight, model.ThemeDark:
	default:
		v.Theme = d.Theme
	}
	if v.PageSize < 50 || v.PageSize > 5000 {
		v.PageSize = d.PageSize
	}
	if v.EditorFontSize < 10 || v.EditorFontSize > 24 {
		v.EditorFontSize = d.EditorFontSize
	}
	return v
}
