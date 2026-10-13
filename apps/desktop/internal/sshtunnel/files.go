package sshtunnel

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

var ErrNoFile = errors.New("no such file on the SSH server")

const missingStatus = 3

func (t *Tunnel) ReadFile(ctx context.Context, path string) ([]byte, error) {
	return t.run(ctx, readScript(path), nil, path)
}

func (t *Tunnel) WriteFile(ctx context.Context, path string, data []byte) error {
	_, err := t.run(ctx, writeScript(path), data, path)
	return err
}

func readScript(path string) string {
	return fmt.Sprintf(`f=%s; [ -e "$f" ] || exit %d; if [ -r "$f" ]; then cat -- "$f"; else sudo -n cat -- "$f"; fi`, shellQuote(path), missingStatus)
}

func writeScript(path string) string {
	return fmt.Sprintf(`f=%s; [ -e "$f" ] || exit %d; t=$(mktemp) || exit 1; if ! cat > "$t"; then rm -f "$t"; exit 1; fi; `+
		`if [ -w "$f" ]; then cat "$t" > "$f"; else sudo -n tee -- "$f" < "$t" > /dev/null; fi; rc=$?; rm -f "$t"; exit $rc`, shellQuote(path), missingStatus)
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func (t *Tunnel) run(ctx context.Context, script string, stdin []byte, path string) ([]byte, error) {
	client, err := t.current()
	if err != nil {
		return nil, err
	}
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("SSH: %w", err)
	}
	defer session.Close()
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	if stdin != nil {
		session.Stdin = bytes.NewReader(stdin)
	}
	done := make(chan error, 1)
	go func() { done <- session.Run(script) }()
	select {
	case <-ctx.Done():
		session.Close()
		return nil, ctx.Err()
	case err = <-done:
	}
	var exit *ssh.ExitError
	if errors.As(err, &exit) && exit.ExitStatus() == missingStatus {
		return nil, fmt.Errorf("%s: %w", path, ErrNoFile)
	}
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("SSH: %s", fileError(msg))
		}
		return nil, fmt.Errorf("SSH: %w", err)
	}
	return stdout.Bytes(), nil
}

func fileError(msg string) string {
	if strings.Contains(msg, "a password is required") || strings.Contains(msg, "a terminal is required") {
		return "the SSH user can't access the file and sudo asks for a password — give the user access to the file or passwordless sudo"
	}
	return msg
}
