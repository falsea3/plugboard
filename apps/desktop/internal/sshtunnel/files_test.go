package sshtunnel

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func TestShellQuote(t *testing.T) {
	if got := shellQuote(`/etc/it's here`); got != `'/etc/it'\''s here'` {
		t.Fatal(got)
	}
}

func TestFilesOnServer(t *testing.T) {
	addr := os.Getenv("PLUGBOARD_TEST_SSH")
	if addr == "" {
		t.Skip("PLUGBOARD_TEST_SSH not set")
	}
	i := strings.LastIndex(addr, ":")
	port, _ := strconv.Atoi(addr[i+1:])
	ctx := context.Background()
	tun, err := Open(ctx, model.SSHTunnel{Host: addr[:i], Port: port, User: "relay", Auth: model.SSHAuthPassword, Password: "relay"},
		"127.0.0.1", 5432, Options{KnownHostsFile: filepath.Join(t.TempDir(), "known_hosts")})
	if err != nil {
		t.Fatal(err)
	}
	defer tun.Close()

	file := "/tmp/plugboard test's.conf"
	if _, err := tun.run(ctx, "printf 'old\\n' > "+shellQuote(file), nil, file); err != nil {
		t.Fatal(err)
	}
	defer tun.run(ctx, "rm -f "+shellQuote(file), nil, file)
	text := "listen_addresses = '*'\n# ünïcode\n"
	if err := tun.WriteFile(ctx, file, []byte(text)); err != nil {
		t.Fatal(err)
	}
	got, err := tun.ReadFile(ctx, file)
	if err != nil || string(got) != text {
		t.Fatalf("read %q, %v", got, err)
	}
	if _, err := tun.ReadFile(ctx, "/tmp/plugboard-missing.conf"); !errors.Is(err, ErrNoFile) {
		t.Fatalf("missing file: %v", err)
	}
	if err := tun.WriteFile(ctx, "/tmp/plugboard-missing.conf", []byte("x")); !errors.Is(err, ErrNoFile) {
		t.Fatalf("writing a missing file: %v", err)
	}
	if err := tun.WriteFile(ctx, "/etc/motd", []byte("x")); err == nil || !strings.HasPrefix(err.Error(), "SSH: ") {
		t.Fatalf("a file the user can't write: %v", err)
	}
}
