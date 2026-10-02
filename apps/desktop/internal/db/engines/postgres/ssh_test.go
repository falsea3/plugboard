package postgres_test

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/engines/postgres"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/sqlcore"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"github.com/relay-client/plugboard/apps/desktop/internal/sshtunnel"
)

func TestThroughSSH(t *testing.T) {
	addr := os.Getenv("PLUGBOARD_TEST_SSH")
	if addr == "" {
		t.Skip("PLUGBOARD_TEST_SSH not set")
	}
	i := strings.LastIndex(addr, ":")
	host := addr[:i]
	port, _ := strconv.Atoi(addr[i+1:])
	c := model.Connection{
		Driver: model.Postgres, Host: "postgres", Port: 5432, User: "relay", Password: "relay", Database: "shop", SSLMode: "disable",
		SSH: model.SSHTunnel{Enabled: true, Host: host, Port: port, User: "relay", Auth: model.SSHAuthPassword, Password: "relay"},
	}
	ctx := context.Background()
	s, err := sqlcore.Open(ctx, "ssh", c, postgres.Dialect{}, db.OpenOptions{KnownHostsFile: t.TempDir() + "/known_hosts"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	res, err := s.Run(ctx, "select count(*) from customers")
	if err != nil || res[0].Rows[0][0] != int64(2500) {
		t.Fatalf("res = %v, err = %v", res, err)
	}
}

func TestSSHTunnelSurvivesServerRestart(t *testing.T) {
	addr := os.Getenv("PLUGBOARD_TEST_SSH")
	if addr == "" || os.Getenv("PLUGBOARD_TEST_SSH_RESTART") == "" {
		t.Skip("PLUGBOARD_TEST_SSH and PLUGBOARD_TEST_SSH_RESTART not set")
	}
	i := strings.LastIndex(addr, ":")
	host := addr[:i]
	port, _ := strconv.Atoi(addr[i+1:])
	c := model.Connection{
		Driver: model.Postgres, Host: "postgres", Port: 5432, User: "relay", Password: "relay", Database: "shop", SSLMode: "disable",
		SSH: model.SSHTunnel{Enabled: true, Host: host, Port: port, User: "relay", Auth: model.SSHAuthPassword, Password: "relay"},
	}
	var mu sync.Mutex
	var states []sshtunnel.State
	ctx := context.Background()
	s, err := sqlcore.Open(ctx, "ssh", c, postgres.Dialect{}, db.OpenOptions{
		KnownHostsFile: t.TempDir() + "/known_hosts",
		OnTunnelState: func(st sshtunnel.State, _ error) {
			mu.Lock()
			states = append(states, st)
			mu.Unlock()
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.Run(ctx, "select 1"); err != nil {
		t.Fatal(err)
	}

	if out, err := exec.Command("docker", "restart", "dev-bastion-1").CombinedOutput(); err != nil {
		t.Fatalf("restart bastion: %v %s", err, out)
	}
	waitForSSH(t, addr)

	if _, err := s.TablePage(ctx, model.TableQuery{Schema: "public", Table: "customers", Limit: 5}); err != nil {
		t.Fatalf("table page after restart: %v", err)
	}
	if _, err := s.Run(ctx, "select count(*) from customers"); err != nil {
		t.Fatalf("editor read after restart: %v", err)
	}
	mu.Lock()
	got := append([]sshtunnel.State(nil), states...)
	mu.Unlock()
	if len(got) == 0 || got[len(got)-1] != sshtunnel.StateReconnected {
		t.Fatalf("tunnel states = %v, want … reconnected", got)
	}

	if out, err := exec.Command("docker", "restart", "dev-bastion-1").CombinedOutput(); err != nil {
		t.Fatalf("restart bastion: %v %s", err, out)
	}
	waitForSSH(t, addr)
	_, err = s.Run(ctx, "update customers set name = name where id = 1")
	if !errors.Is(err, db.ErrConnLost) {
		t.Fatalf("write after drop: err = %v, want db.ErrConnLost", err)
	}
	if _, err := s.Run(ctx, "update customers set name = name where id = 1"); err != nil {
		t.Fatalf("write on the fresh connection: %v", err)
	}
}

func waitForSSH(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if c, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
			buf := make([]byte, 4)
			c.SetReadDeadline(time.Now().Add(time.Second))
			n, _ := c.Read(buf)
			c.Close()
			if n == 4 && string(buf) == "SSH-" {
				return
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatal("SSH server did not come back")
}
