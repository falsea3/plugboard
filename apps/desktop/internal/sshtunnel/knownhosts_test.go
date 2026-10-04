package sshtunnel

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestChangedHostKeyIsRefused(t *testing.T) {
	known := filepath.Join(t.TempDir(), "known_hosts")
	target := echoServer(t)

	first, _ := newSigner(t)
	srv := startServer(t, first, nil)
	cfg := model.SSHTunnel{Host: srv.addr, Port: srv.port, User: "relay", Auth: model.SSHAuthPassword, Password: "pw"}
	tun, err := Open(context.Background(), cfg, "127.0.0.1", target, Options{KnownHostsFile: known})
	if err != nil {
		t.Fatal(err)
	}
	tun.Close()

	second, _ := newSigner(t)
	srv2 := startServer(t, second, nil)
	data, _ := os.ReadFile(known)
	oldHost := "[127.0.0.1]:" + strconv.Itoa(srv.port)
	newHost := "[127.0.0.1]:" + strconv.Itoa(srv2.port)
	os.WriteFile(known, []byte(strings.Replace(string(data), oldHost, newHost, 1)), 0o600)

	cfg.Port = srv2.port
	_, err = Open(context.Background(), cfg, "127.0.0.1", target, Options{KnownHostsFile: known})
	var changed *HostKeyChangedError
	if !errors.As(err, &changed) {
		t.Fatalf("err = %v, want host key change refusal", err)
	}

	if err := changed.Trust(known); err != nil {
		t.Fatal(err)
	}
	tun, err = Open(context.Background(), cfg, "127.0.0.1", target, Options{KnownHostsFile: known})
	if err != nil {
		t.Fatalf("after trusting the new key: %v", err)
	}
	tun.Close()

	third, _ := newSigner(t)
	srv3 := startServer(t, third, nil)
	data, _ = os.ReadFile(known)
	os.WriteFile(known, []byte(strings.Replace(string(data), newHost, "[127.0.0.1]:"+strconv.Itoa(srv3.port), 1)), 0o600)
	cfg.Port = srv3.port
	if _, err := Open(context.Background(), cfg, "127.0.0.1", target, Options{KnownHostsFile: known}); !errors.As(err, &changed) {
		t.Fatalf("err = %v, want the third key refused", err)
	}
}

func TestKnownKeyOfAnotherTypeIsUsed(t *testing.T) {
	known := filepath.Join(t.TempDir(), "known_hosts")
	target := echoServer(t)
	ed, _ := newSigner(t)
	ecKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ec, err := ssh.NewSignerFromKey(ecKey)
	if err != nil {
		t.Fatal(err)
	}
	srv := startServer(t, ed, nil, ec)
	host := knownhosts.Normalize(net.JoinHostPort(srv.addr, strconv.Itoa(srv.port)))
	for _, onFile := range []ssh.Signer{ec, ed} {
		os.WriteFile(known, []byte(knownhosts.Line([]string{host}, onFile.PublicKey())+"\n"), 0o600)
		cfg := model.SSHTunnel{Host: srv.addr, Port: srv.port, User: "relay", Auth: model.SSHAuthPassword, Password: "pw"}
		tun, err := Open(context.Background(), cfg, "127.0.0.1", target, Options{KnownHostsFile: known})
		if err != nil {
			t.Fatalf("known %s key: %v", onFile.PublicKey().Type(), err)
		}
		tun.Close()
	}
}
