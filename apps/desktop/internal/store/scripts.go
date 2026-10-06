package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
)

const (
	scriptExt     = ".sql"
	tabsFile      = ".tabs.json"
	maxScriptName = 120
)

var (
	connIDPattern   = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	ErrScriptExists = apperr.New("script_exists", "a script with this name already exists")
	ErrNoScript     = apperr.New("script_missing", "this script no longer exists")
)

type Scripts struct {
	mu  sync.Mutex
	dir string
}

func NewScripts(dir string) *Scripts {
	return &Scripts{dir: filepath.Join(dir, "scripts")}
}

func CheckScriptName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return apperr.New("bad_name", "give the script a name")
	case name != strings.TrimSpace(name):
		return apperr.New("bad_name", "a script name can't start or end with a space")
	case len(name) > maxScriptName:
		return apperr.New("bad_name", "a script name is at most 120 characters")
	case strings.HasPrefix(name, "."):
		return apperr.New("bad_name", "a script name can't start with a dot")
	case strings.ContainsAny(name, `/\:`) || strings.ContainsFunc(name, unicode.IsControl):
		return apperr.New("bad_name", `a script name can't contain / \ : or control characters`)
	}
	return nil
}

func (s *Scripts) folder(conn string) (string, error) {
	if !connIDPattern.MatchString(conn) {
		return "", apperr.New("bad_connection", "unknown connection")
	}
	return filepath.Join(s.dir, conn), nil
}

func (s *Scripts) file(conn, name string) (string, error) {
	dir, err := s.folder(conn)
	if err != nil {
		return "", err
	}
	if err := CheckScriptName(name); err != nil {
		return "", err
	}
	return filepath.Join(dir, name+scriptExt), nil
}

func (s *Scripts) List(conn string) ([]string, error) {
	dir, err := s.folder(conn)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), scriptExt)
		if ok && e.Type().IsRegular() && CheckScriptName(name) == nil {
			names = append(names, name)
		}
	}
	slices.SortFunc(names, func(a, b string) int { return strings.Compare(strings.ToLower(a), strings.ToLower(b)) })
	return names, nil
}

func (s *Scripts) Read(conn, name string) (string, error) {
	path, err := s.file(conn, name)
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrNoScript
	}
	return string(data), err
}

func (s *Scripts) Create(conn, name, sql string) error {
	path, err := s.file(conn, name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, fs.ErrExist) {
		return ErrScriptExists
	}
	if err != nil {
		return err
	}
	_, werr := f.WriteString(sql)
	if err := errors.Join(werr, f.Close()); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func (s *Scripts) Write(conn, name, sql string) error {
	path, err := s.file(conn, name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeFileAtomic(path, []byte(sql), 0o600)
}

func (s *Scripts) Rename(conn, from, to string) error {
	src, err := s.file(conn, from)
	if err != nil {
		return err
	}
	dst, err := s.file(conn, to)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := os.Stat(src); errors.Is(err, fs.ErrNotExist) {
		return ErrNoScript
	}
	if !strings.EqualFold(from, to) {
		if _, err := os.Lstat(dst); err == nil {
			return ErrScriptExists
		}
	}
	return os.Rename(src, dst)
}

func (s *Scripts) Delete(conn, name string) error {
	path, err := s.file(conn, name)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
