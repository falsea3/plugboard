package sshtunnel

import (
	"context"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"golang.org/x/crypto/ssh"
)

func TestPasswordTunnelForwardsTraffic(t *testing.T) {
	host, _ := newSigner(t)
	srv := startServer(t, host, nil)
	target := echoServer(t)
	known := filepath.Join(t.TempDir(), "known_hosts")

	tun, err := Open(context.Background(), model.SSHTunnel{Host: srv.addr, Port: srv.port, User: "relay", Auth: model.SSHAuthPassword, Password: "pw"},
		"127.0.0.1", target, Options{KnownHostsFile: known})
	if err != nil {
		t.Fatal(err)
	}
	defer tun.Close()
	roundTrip(t, tun, "hello")
	roundTrip(t, tun, "second connection")

	data, _ := os.ReadFile(known)
	if !strings.Contains(string(data), host.PublicKey().Type()) {
		t.Fatalf("host key was not recorded: %q", data)
	}
}

func TestKeyAuthWithPassphrase(t *testing.T) {
	host, _ := newSigner(t)
	client, priv := newSigner(t)
	srv := startServer(t, host, client.PublicKey())
	target := echoServer(t)

	block, err := ssh.MarshalPrivateKeyWithPassphrase(priv, "", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(t.TempDir(), "id_ed25519")
	os.WriteFile(keyFile, pem.EncodeToMemory(block), 0o600)
	cfg := model.SSHTunnel{Host: srv.addr, Port: srv.port, User: "relay", Auth: model.SSHAuthKey, KeyFile: keyFile}
	opts := Options{KnownHostsFile: filepath.Join(t.TempDir(), "kh")}

	if _, err := Open(context.Background(), cfg, "127.0.0.1", target, opts); err == nil || !strings.Contains(err.Error(), "passphrase") {
		t.Fatalf("missing passphrase: err = %v", err)
	}
	cfg.Passphrase = "secret"
	tun, err := Open(context.Background(), cfg, "127.0.0.1", target, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer tun.Close()
	roundTrip(t, tun, "via key")
}

func TestWrongPasswordFails(t *testing.T) {
	host, _ := newSigner(t)
	srv := startServer(t, host, nil)
	_, err := Open(context.Background(), model.SSHTunnel{Host: srv.addr, Port: srv.port, User: "relay", Auth: model.SSHAuthPassword, Password: "nope"},
		"127.0.0.1", 1, Options{KnownHostsFile: filepath.Join(t.TempDir(), "kh")})
	if err == nil || !strings.Contains(err.Error(), "sign-in was refused") {
		t.Fatalf("err = %v", err)
	}
}

func TestTunnelReconnectsAfterDrop(t *testing.T) {
	host, _ := newSigner(t)
	srv := startServer(t, host, nil)
	target := echoServer(t)

	var mu sync.Mutex
	var states []State
	tun, err := Open(context.Background(), model.SSHTunnel{Host: srv.addr, Port: srv.port, User: "relay", Auth: model.SSHAuthPassword, Password: "pw"},
		"127.0.0.1", target, Options{
			KnownHostsFile: filepath.Join(t.TempDir(), "kh"),
			OnState: func(s State, _ error) {
				mu.Lock()
				states = append(states, s)
				mu.Unlock()
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	defer tun.Close()
	port := tun.LocalPort()
	roundTrip(t, tun, "before")

	srv.dropAll()
	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		n := len(states)
		mu.Unlock()
		if n > 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	roundTrip(t, tun, "after")
	if tun.LocalPort() != port {
		t.Fatal("local port changed; drivers would lose the tunnel")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(states) < 2 || states[0] != StateLost || states[len(states)-1] != StateReconnected {
		t.Fatalf("states = %v, want lost … reconnected", states)
	}
}

func TestRefusedSignInGetsACode(t *testing.T) {
	err := cleanAuthError(errors.New("ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain"))
	coded, ok := errors.AsType[apperr.Coded](err)
	if !ok || coded.Code() != "ssh_auth" {
		t.Errorf("cleanAuthError = %v", err)
	}
}
