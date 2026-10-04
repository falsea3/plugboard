package sshtunnel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"golang.org/x/crypto/ssh"
)

const (
	dialTimeout       = 15 * time.Second
	keepAliveInterval = 30 * time.Second
)

type Tunnel struct {
	ln      net.Listener
	target  string
	dial    func(context.Context) (*ssh.Client, error)
	onState func(State, error)

	mu     sync.Mutex
	client *ssh.Client
	closed bool

	wg   sync.WaitGroup
	stop chan struct{}
}

type State string

const (
	StateLost        State = "lost"
	StateReconnected State = "reconnected"
	StateFailed      State = "failed"
)

type Options struct {
	KnownHostsFile string
	OnState        func(State, error)
}

func Open(ctx context.Context, cfg model.SSHTunnel, targetHost string, targetPort int, opts Options) (*Tunnel, error) {
	if strings.TrimSpace(cfg.Host) == "" {
		return nil, errors.New("SSH: no server to connect to — set the SSH host or the database host")
	}
	if strings.TrimSpace(cfg.User) == "" {
		return nil, errors.New("SSH: set the SSH user")
	}
	port := cfg.Port
	if port == 0 {
		port = 22
	}
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(port))
	hosts := knownHosts{appFile: opts.KnownHostsFile}
	dial := func(ctx context.Context) (*ssh.Client, error) {
		auth, closeAuth, err := authMethods(cfg)
		if err != nil {
			return nil, err
		}
		defer closeAuth()
		d := net.Dialer{Timeout: dialTimeout}
		raw, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("SSH: connect to %s: %w", addr, err)
		}
		if deadline, ok := ctx.Deadline(); ok {
			raw.SetDeadline(deadline)
		}
		config := &ssh.ClientConfig{
			User:              cfg.User,
			Auth:              auth,
			HostKeyCallback:   hosts.check,
			HostKeyAlgorithms: hosts.algorithms(addr, raw.RemoteAddr()),
			Timeout:           dialTimeout,
		}
		c, chans, reqs, err := ssh.NewClientConn(raw, addr, config)
		if err != nil {
			raw.Close()
			return nil, fmt.Errorf("SSH: %w", cleanAuthError(err))
		}
		raw.SetDeadline(time.Time{})
		return ssh.NewClient(c, chans, reqs), nil
	}

	client, err := dial(ctx)
	if err != nil {
		return nil, err
	}
	if targetHost == "" {
		targetHost = "127.0.0.1"
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		client.Close()
		return nil, err
	}
	onState := opts.OnState
	if onState == nil {
		onState = func(State, error) {}
	}
	t := &Tunnel{
		ln:      ln,
		target:  net.JoinHostPort(targetHost, strconv.Itoa(targetPort)),
		dial:    dial,
		onState: onState,
		stop:    make(chan struct{}),
	}
	t.setClient(client)
	t.wg.Add(2)
	go t.accept()
	go t.keepAlive()
	return t, nil
}

func (t *Tunnel) LocalPort() int {
	return t.ln.Addr().(*net.TCPAddr).Port
}

func (t *Tunnel) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	client := t.client
	t.client = nil
	t.mu.Unlock()

	close(t.stop)
	t.ln.Close()
	var err error
	if client != nil {
		err = client.Close()
	}
	t.wg.Wait()
	return err
}
