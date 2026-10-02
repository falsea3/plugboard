package db

import (
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
