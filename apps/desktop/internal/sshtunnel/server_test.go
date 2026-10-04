package sshtunnel

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"

	"golang.org/x/crypto/ssh"
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
