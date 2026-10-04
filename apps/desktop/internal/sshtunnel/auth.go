package sshtunnel

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func authMethods(cfg model.SSHTunnel) ([]ssh.AuthMethod, func(), error) {
	switch cfg.Auth {
	case model.SSHAuthPassword, "":
		pw := cfg.Password
		return []ssh.AuthMethod{
			ssh.Password(pw),
			ssh.KeyboardInteractive(func(_, _ string, questions []string, _ []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range answers {
					answers[i] = pw
				}
				return answers, nil
			}),
		}, func() {}, nil
	case model.SSHAuthKey:
		if strings.TrimSpace(cfg.KeyFile) == "" {
			return nil, nil, errors.New("SSH: choose a private key file")
		}
		path, err := expandHome(cfg.KeyFile)
		if err != nil {
			return nil, nil, err
		}
		pem, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("SSH: read private key: %w", err)
		}
		var signer ssh.Signer
		if cfg.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(pem, []byte(cfg.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(pem)
		}
		var missing *ssh.PassphraseMissingError
		if errors.As(err, &missing) {
			return nil, nil, errors.New("SSH: the private key is encrypted — enter its passphrase")
		}
		if err != nil {
			return nil, nil, fmt.Errorf("SSH: parse private key: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, func() {}, nil
	case model.SSHAuthAgent:
		sock := os.Getenv("SSH_AUTH_SOCK")
		if sock == "" {
			return nil, nil, errors.New("SSH: no agent is running (SSH_AUTH_SOCK is not set)")
		}
		conn, err := net.Dial("unix", sock)
		if err != nil {
			return nil, nil, fmt.Errorf("SSH: connect to agent: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeysCallback(agent.NewClient(conn).Signers)}, func() { conn.Close() }, nil
	}
	return nil, nil, fmt.Errorf("SSH: unsupported authentication %q", cfg.Auth)
}

func expandHome(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}

func cleanAuthError(err error) error {
	if strings.Contains(err.Error(), "unable to authenticate") {
		return apperr.New("ssh_auth", "sign-in was refused — check the SSH user and the key or password")
	}
	return err
}
