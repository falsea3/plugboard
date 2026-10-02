package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/zalando/go-keyring"
)

var ErrSecretNotFound = errors.New("secret not found")

type Secrets interface {
	Get(key string) (string, error)
	Set(key, value string) error
	Delete(key string) error
}

const (
	keyringService    = "Plugboard"
	oldKeyringService = "Relay DB"
)

func NewSecrets(dir string) Secrets {
	if keychainDisabled() {
		return &fileSecrets{path: filepath.Join(dir, "secrets.dev.json")}
	}
	return keychainSecrets{service: keyringService}
}

func OldSecrets() Secrets {
	if keychainDisabled() {
		return nil
	}
	return keychainSecrets{service: oldKeyringService}
}

func keychainDisabled() bool { return os.Getenv("PLUGBOARD_DISABLE_KEYCHAIN") == "1" }

type keychainSecrets struct{ service string }

func (k keychainSecrets) Get(key string) (string, error) {
	v, err := keyring.Get(k.service, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrSecretNotFound
	}
	return v, err
}

func (k keychainSecrets) Set(key, value string) error {
	return keyring.Set(k.service, key, value)
}

func (k keychainSecrets) Delete(key string) error {
	err := keyring.Delete(k.service, key)
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrSecretNotFound
	}
	return err
}

type fileSecrets struct {
	mu   sync.Mutex
	path string
}

func (f *fileSecrets) read() (map[string]string, error) {
	m := map[string]string{}
	data, err := os.ReadFile(f.path)
	if errors.Is(err, os.ErrNotExist) {
		return m, nil
	}
	if err != nil {
		return nil, err
	}
	return m, json.Unmarshal(data, &m)
}

func (f *fileSecrets) write(m map[string]string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(f.path, data, 0o600)
}

func (f *fileSecrets) Get(key string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.read()
	if err != nil {
		return "", err
	}
	v, ok := m[key]
	if !ok {
		return "", ErrSecretNotFound
	}
	return v, nil
}

func (f *fileSecrets) Set(key, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.read()
	if err != nil {
		return err
	}
	m[key] = value
	return f.write(m)
}

func (f *fileSecrets) Delete(key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.read()
	if err != nil {
		return err
	}
	if _, ok := m[key]; !ok {
		return ErrSecretNotFound
	}
	delete(m, key)
	return f.write(m)
}
