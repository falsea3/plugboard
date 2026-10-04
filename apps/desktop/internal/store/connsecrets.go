package store

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func (s *Connections) FillSecrets(c *model.Connection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	saved, ok, err := s.find(c.ID)
	if err != nil || !ok {
		return err
	}
	if err := s.readSecrets(&saved); err != nil {
		return err
	}
	for _, f := range secretFields {
		if *f.field(c) == "" && f.target(*c) == f.target(saved) {
			*f.field(c) = *f.field(&saved)
		}
	}
	return nil
}

func (s *Connections) MoveSecrets(from Secrets) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	all, err := s.load()
	if err != nil {
		return err
	}
	var errs []error
	for _, c := range all {
		for _, f := range secretFields {
			key := secretKey(c.ID, f.slot)
			v, err := from.Get(key)
			if errors.Is(err, ErrSecretNotFound) {
				continue
			}
			if err != nil {
				errs = append(errs, fmt.Errorf("read old %s of %s: %w", f.name, c.Name, err))
				continue
			}
			if _, err := s.secrets.Get(key); errors.Is(err, ErrSecretNotFound) {
				if err := s.secrets.Set(key, v); err != nil {
					errs = append(errs, fmt.Errorf("save %s of %s: %w", f.name, c.Name, err))
					continue
				}
			} else if err != nil {
				errs = append(errs, err)
				continue
			}
			if err := from.Delete(key); err != nil && !errors.Is(err, ErrSecretNotFound) {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func (s *Connections) readSecrets(c *model.Connection) error {
	if !c.SavePassword {
		return nil
	}
	for _, f := range secretFields {
		v, err := s.secrets.Get(secretKey(c.ID, f.slot))
		if err != nil && !errors.Is(err, ErrSecretNotFound) {
			return fmt.Errorf("read saved %s: %w", f.name, err)
		}
		*f.field(c) = v
	}
	return nil
}

var secretFields = []struct {
	name   string
	slot   string
	field  func(*model.Connection) *string
	target func(model.Connection) string
}{
	{"password", "password",
		func(c *model.Connection) *string { return &c.Password },
		func(c model.Connection) string { return endpoint(string(c.Driver), c.Host, c.Port, c.User) }},
	{"SSH password", "ssh-password",
		func(c *model.Connection) *string { return &c.SSH.Password },
		func(c model.Connection) string { return endpoint("ssh", c.SSHHost(), c.SSH.Port, c.SSH.User) }},
	{"SSH key passphrase", "ssh-passphrase",
		func(c *model.Connection) *string { return &c.SSH.Passphrase },
		func(c model.Connection) string { return c.SSH.KeyFile }},
}

func endpoint(kind, host string, port int, user string) string {
	return kind + "://" + user + "@" + strings.ToLower(strings.TrimSpace(host)) + ":" + strconv.Itoa(port)
}

func secretKey(id, slot string) string { return "connection:" + id + ":" + slot }
