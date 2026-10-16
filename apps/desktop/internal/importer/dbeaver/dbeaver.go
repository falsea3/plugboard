package dbeaver

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type source struct {
	Folders     map[string]json.RawMessage `json:"folders"`
	Connections map[string]dsConnection    `json:"connections"`
	Types       map[string]struct {
		Name string `json:"name"`
	} `json:"connection-types"`
}

type dsConnection struct {
	Provider     string   `json:"provider"`
	Driver       string   `json:"driver"`
	Name         string   `json:"name"`
	Folder       string   `json:"folder"`
	SavePassword bool     `json:"save-password"`
	Config       dsConfig `json:"configuration"`
}

type dsConfig struct {
	Host     string               `json:"host"`
	Port     json.Number          `json:"port"`
	Database string               `json:"database"`
	URL      string               `json:"url"`
	Type     string               `json:"type"`
	ReadOnly bool                 `json:"read-only"`
	User     string               `json:"user"`
	Password string               `json:"password"`
	Handlers map[string]dsHandler `json:"handlers"`
}

type dsHandler struct {
	Type       string            `json:"type"`
	Enabled    bool              `json:"enabled"`
	User       string            `json:"user"`
	Password   string            `json:"password"`
	Properties map[string]string `json:"properties"`
}

func (h *dsHandler) UnmarshalJSON(b []byte) error {
	var raw struct {
		Type       string                     `json:"type"`
		Enabled    bool                       `json:"enabled"`
		User       string                     `json:"user"`
		Password   string                     `json:"password"`
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*h = dsHandler{Type: raw.Type, Enabled: raw.Enabled, User: raw.User, Password: raw.Password, Properties: map[string]string{}}
	for k, v := range raw.Properties {
		var s string
		if json.Unmarshal(v, &s) != nil {
			s = strings.Trim(string(v), `"`)
		}
		h.Properties[k] = s
	}
	return nil
}

func Workspaces() []string {
	home, _ := os.UserHomeDir()
	var root string
	switch runtime.GOOS {
	case "darwin":
		root = filepath.Join(home, "Library", "DBeaverData", "workspace6")
	case "windows":
		dir, _ := os.UserConfigDir()
		root = filepath.Join(dir, "DBeaverData", "workspace6")
	default:
		root = filepath.Join(home, ".local", "share", "DBeaverData", "workspace6")
	}
	if _, err := os.Stat(root); err != nil {
		return nil
	}
	return []string{root}
}

type project struct {
	name string
	dir  string
}

func projects(root string) []project {
	if hasConfig(root) {
		return []project{{name: filepath.Base(filepath.Dir(root)), dir: root}}
	}
	if hasConfig(filepath.Join(root, ".dbeaver")) {
		return []project{{name: filepath.Base(root), dir: filepath.Join(root, ".dbeaver")}}
	}
	entries, _ := os.ReadDir(root)
	var out []project
	for _, e := range entries {
		dir := filepath.Join(root, e.Name(), ".dbeaver")
		if e.IsDir() && hasConfig(dir) {
			out = append(out, project{name: e.Name(), dir: dir})
		}
	}
	return out
}

func hasConfig(dir string) bool {
	files, _ := filepath.Glob(filepath.Join(dir, "data-sources*.json"))
	return len(files) > 0
}

func readProject(p project) ([]source, map[string]credentials, error) {
	files, err := filepath.Glob(filepath.Join(p.dir, "data-sources*.json"))
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(files)
	var out []source
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, nil, err
		}
		var s source
		if err := json.Unmarshal(b, &s); err != nil {
			return nil, nil, errors.New(filepath.Base(f) + " isn't a DBeaver connection list: " + err.Error())
		}
		out = append(out, s)
	}
	creds, err := readCredentials(filepath.Join(p.dir, "credentials-config.json"))
	return out, creds, err
}
