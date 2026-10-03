package sshtunnel

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type testServer struct {
	addr string
	port int

	mu    *sync.Mutex
	conns *[]net.Conn
}

func (s testServer) dropAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range *s.conns {
		c.Close()
	}
	*s.conns = nil
}

func startServer(t *testing.T, hostKey ssh.Signer, clientKey ssh.PublicKey, moreHostKeys ...ssh.Signer) testServer {
	t.Helper()
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pw []byte) (*ssh.Permissions, error) {
			if c.User() == "relay" && string(pw) == "pw" {
				return nil, nil
			}
			return nil, io.EOF
		},
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if clientKey != nil && string(key.Marshal()) == string(clientKey.Marshal()) {
				return nil, nil
			}
			return nil, io.EOF
		},
	}
	cfg.AddHostKey(hostKey)
	for _, k := range moreHostKeys {
		cfg.AddHostKey(k)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	srv := testServer{addr: "127.0.0.1", port: ln.Addr().(*net.TCPAddr).Port, mu: &sync.Mutex{}, conns: &[]net.Conn{}}
	go func() {
		for {
			raw, err := ln.Accept()
			if err != nil {
				return
			}
			srv.mu.Lock()
			*srv.conns = append(*srv.conns, raw)
			srv.mu.Unlock()
			go serveConn(raw, cfg)
		}
	}()
	return srv
}

func serveConn(raw net.Conn, cfg *ssh.ServerConfig) {
	_, chans, reqs, err := ssh.NewServerConn(raw, cfg)
	if err != nil {
		raw.Close()
		return
	}
	go ssh.DiscardRequests(reqs)
	for nc := range chans {
		if nc.ChannelType() != "direct-tcpip" {
			nc.Reject(ssh.UnknownChannelType, "")
			continue
		}
		var p struct {
			Host     string
			Port     uint32
			OrigHost string
			OrigPort uint32
		}
		ssh.Unmarshal(nc.ExtraData(), &p)
		target, err := net.Dial("tcp", net.JoinHostPort(p.Host, strconv.Itoa(int(p.Port))))
		if err != nil {
			nc.Reject(ssh.ConnectionFailed, err.Error())
			continue
		}
		ch, creqs, _ := nc.Accept()
		go ssh.DiscardRequests(creqs)
		go func() { io.Copy(ch, target); ch.CloseWrite() }()
		go func() { io.Copy(target, ch); target.Close() }()
	}
}

func echoServer(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { io.Copy(c, c); c.Close() }()
		}
	}()
	return ln.Addr().(*net.TCPAddr).Port
}

func newSigner(t *testing.T) (ssh.Signer, ed25519.PrivateKey) {
	t.Helper()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	s, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s, priv
}

func roundTrip(t *testing.T, tun *Tunnel, msg string) {
	t.Helper()
	c, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(tun.LocalPort()))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	io.WriteString(c, msg+"\n")
	got, err := bufio.NewReader(c).ReadString('\n')
	if err != nil || got != msg+"\n" {
		t.Fatalf("echo = %q, %v", got, err)
	}
}

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
