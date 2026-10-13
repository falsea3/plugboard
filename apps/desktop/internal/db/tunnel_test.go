package db

import (
	"context"
	"io"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

func TestTunnelEndpoints(t *testing.T) {
	c := model.Connection{Host: "203.0.113.7", SSH: model.SSHTunnel{Enabled: true, User: "root"}}
	ssh, target := TunnelEndpoints(c)
	if ssh.Host != "203.0.113.7" || target != "127.0.0.1" {
		t.Fatalf("no SSH host: ssh=%q target=%q", ssh.Host, target)
	}
	c.SSH.Host = "bastion"
	c.Host = "db.internal"
	ssh, target = TunnelEndpoints(c)
	if ssh.Host != "bastion" || target != "db.internal" {
		t.Fatalf("bastion: ssh=%q target=%q", ssh.Host, target)
	}
}

type fakeFiles struct{ io.Closer }

func (fakeFiles) ReadFile(context.Context, string) ([]byte, error) { return nil, nil }
func (fakeFiles) WriteFile(context.Context, string, []byte) error  { return nil }

func TestDatabaseHostFiles(t *testing.T) {
	c := model.Connection{Host: "203.0.113.7", SSH: model.SSHTunnel{Enabled: true, User: "root"}}
	if DatabaseHostFiles(c, fakeFiles{}) == nil {
		t.Fatal("the database runs on the SSH server: its files are there")
	}
	c.SSH.Host = "bastion"
	if DatabaseHostFiles(c, fakeFiles{}) != nil {
		t.Fatal("behind a bastion the files are on another machine")
	}
	c.Host = "localhost"
	if DatabaseHostFiles(c, fakeFiles{}) == nil {
		t.Fatal("a database on the bastion's loopback")
	}
	if DatabaseHostFiles(c, nil) != nil {
		t.Fatal("no tunnel, no files")
	}
}
