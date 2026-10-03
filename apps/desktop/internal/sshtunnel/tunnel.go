package sshtunnel

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
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

func (t *Tunnel) setClient(c *ssh.Client) {
	t.mu.Lock()
	t.client = c
	t.mu.Unlock()
	go func() {
		c.Wait()
		t.mu.Lock()
		lost := t.client == c && !t.closed
		if lost {
			t.client = nil
		}
		t.mu.Unlock()
		if lost {
			t.onState(StateLost, nil)
			t.reconnectSoon()
		}
	}()
}

func (t *Tunnel) reconnectSoon() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.wg.Add(1)
	t.mu.Unlock()
	go func() {
		defer t.wg.Done()
		for _, delay := range []time.Duration{time.Second, 2 * time.Second, 5 * time.Second, 10 * time.Second, 30 * time.Second} {
			select {
			case <-t.stop:
				return
			case <-time.After(delay):
			}
			t.mu.Lock()
			done := t.closed || t.client != nil
			t.mu.Unlock()
			if done {
				return
			}
			if _, err := t.current(); err == nil {
				return
			}
		}
	}()
}

func (t *Tunnel) current() (*ssh.Client, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, net.ErrClosed
	}
	if c := t.client; c != nil {
		t.mu.Unlock()
		return c, nil
	}
	t.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	c, err := t.dial(ctx)
	if err != nil {
		t.onState(StateFailed, err)
		return nil, err
	}
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		c.Close()
		return nil, net.ErrClosed
	}
	if t.client != nil {
		existing := t.client
		t.mu.Unlock()
		c.Close()
		return existing, nil
	}
	t.mu.Unlock()
	t.setClient(c)
	t.onState(StateReconnected, nil)
	return c, nil
}

func (t *Tunnel) forget(c *ssh.Client) {
	t.mu.Lock()
	wasCurrent := t.client == c && !t.closed
	if wasCurrent {
		t.client = nil
	}
	t.mu.Unlock()
	c.Close()
	if wasCurrent {
		t.onState(StateLost, nil)
		t.reconnectSoon()
	}
}

func (t *Tunnel) accept() {
	defer t.wg.Done()
	for {
		local, err := t.ln.Accept()
		if err != nil {
			return
		}
		go t.forward(local)
	}
}

func (t *Tunnel) forward(local net.Conn) {
	var remote net.Conn
	for attempt := 0; attempt < 2; attempt++ {
		c, err := t.current()
		if err != nil {
			local.Close()
			return
		}
		remote, err = c.Dial("tcp", t.target)
		if err == nil {
			break
		}
		var open *ssh.OpenChannelError
		if errors.As(err, &open) {
			local.Close()
			return
		}
		t.forget(c)
	}
	if remote == nil {
		local.Close()
		return
	}
	done := make(chan struct{}, 2)
	pipe := func(dst, src net.Conn) {
		io.Copy(dst, src)
		done <- struct{}{}
	}
	go pipe(remote, local)
	go pipe(local, remote)
	<-done
	local.Close()
	remote.Close()
	<-done
}

func (t *Tunnel) keepAlive() {
	defer t.wg.Done()
	tick := time.NewTicker(keepAliveInterval)
	defer tick.Stop()
	for {
		select {
		case <-t.stop:
			return
		case <-tick.C:
			t.mu.Lock()
			c := t.client
			t.mu.Unlock()
			if c == nil {
				continue
			}
			if _, _, err := c.SendRequest("keepalive@openssh.com", true, nil); err != nil {
				t.forget(c)
			}
		}
	}
}

func authMethods(cfg model.SSHTunnel) ([]ssh.AuthMethod, func(), error) {
	switch cfg.Auth {
	case model.SSHAuthPassword, "":
		pw := cfg.Password
		return []ssh.AuthMethod{
			ssh.Password(pw),
			ssh.KeyboardInteractive(func(_, _ string, questions []string, _ []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range answers {
					answers[i] = pw
				}
				return answers, nil
			}),
		}, func() {}, nil
	case model.SSHAuthKey:
		if strings.TrimSpace(cfg.KeyFile) == "" {
			return nil, nil, errors.New("SSH: choose a private key file")
		}
		path, err := expandHome(cfg.KeyFile)
		if err != nil {
			return nil, nil, err
		}
		pem, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("SSH: read private key: %w", err)
		}
		var signer ssh.Signer
		if cfg.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(pem, []byte(cfg.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(pem)
		}
		var missing *ssh.PassphraseMissingError
		if errors.As(err, &missing) {
			return nil, nil, errors.New("SSH: the private key is encrypted — enter its passphrase")
		}
		if err != nil {
			return nil, nil, fmt.Errorf("SSH: parse private key: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, func() {}, nil
	case model.SSHAuthAgent:
		sock := os.Getenv("SSH_AUTH_SOCK")
		if sock == "" {
			return nil, nil, errors.New("SSH: no agent is running (SSH_AUTH_SOCK is not set)")
		}
		conn, err := net.Dial("unix", sock)
		if err != nil {
			return nil, nil, fmt.Errorf("SSH: connect to agent: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeysCallback(agent.NewClient(conn).Signers)}, func() { conn.Close() }, nil
	}
	return nil, nil, fmt.Errorf("SSH: unsupported authentication %q", cfg.Auth)
}

type knownHosts struct {
	appFile string
}

func (k knownHosts) files() []string {
	files := appendIfExists(nil, k.appFile)
	if home, err := os.UserHomeDir(); err == nil {
		files = appendIfExists(files, filepath.Join(home, ".ssh", "known_hosts"))
	}
	return files
}

func (k knownHosts) check(hostname string, remote net.Addr, key ssh.PublicKey) error {
	if files := k.files(); len(files) > 0 {
		check, err := knownhosts.New(files...)
		if err != nil {
			return fmt.Errorf("read known_hosts: %w", err)
		}
		err = check(hostname, remote, key)
		if err == nil {
			return nil
		}
		var keyErr *knownhosts.KeyError
		if !errors.As(err, &keyErr) {
			return err
		}
		if len(keyErr.Want) > 0 {
			return &HostKeyChangedError{Host: hostname, Fingerprint: ssh.FingerprintSHA256(key), Key: key}
		}
	}
	if k.appFile == "" {
		return nil
	}
	return writeKnownHost(k.appFile, hostname, key, false)
}

func (k knownHosts) algorithms(hostname string, remote net.Addr) []string {
	for _, file := range k.files() {
		check, err := knownhosts.New(file)
		if err != nil {
			continue
		}
		var keyErr *knownhosts.KeyError
		if !errors.As(check(hostname, remote, probeKey), &keyErr) || len(keyErr.Want) == 0 {
			continue
		}
		var algos []string
		for _, known := range keyErr.Want {
			if t := known.Key.Type(); t == ssh.KeyAlgoRSA {
				algos = append(algos, ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA)
			} else {
				algos = append(algos, t)
			}
		}
		return algos
	}
	return nil
}

var probeKey = func() ssh.PublicKey {
	k, err := ssh.NewPublicKey(ed25519.PublicKey(make([]byte, ed25519.PublicKeySize)))
	if err != nil {
		panic(err)
	}
	return k
}()

type HostKeyChangedError struct {
	Host        string
	Fingerprint string
	Key         ssh.PublicKey
}

func (e *HostKeyChangedError) Error() string {
	return fmt.Sprintf("host key for %s has changed (now %s). If the server was rebuilt, trust the new key; otherwise someone may be intercepting the connection", e.Host, e.Fingerprint)
}

func (e *HostKeyChangedError) Trust(appKnownHosts string) error {
	return writeKnownHost(appKnownHosts, e.Host, e.Key, true)
}

func writeKnownHost(file, host string, key ssh.PublicKey, replace bool) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	data, err := os.ReadFile(file)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	want := knownhosts.Normalize(host)
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) == 0 || (replace && containsHost(f[0], want)) {
			continue
		}
		lines = append(lines, line)
	}
	lines = append(lines, knownhosts.Line([]string{want}, key))
	return os.WriteFile(file, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}

func containsHost(field, host string) bool {
	for _, h := range strings.Split(field, ",") {
		if h == host {
			return true
		}
	}
	return false
}

func appendIfExists(files []string, path string) []string {
	if path == "" {
		return files
	}
	if _, err := os.Stat(path); err == nil {
		return append(files, path)
	}
	return files
}

func expandHome(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, path[1:]), nil
	}
	return path, nil
}

func cleanAuthError(err error) error {
	if strings.Contains(err.Error(), "unable to authenticate") {
		return apperr.New("ssh_auth", "sign-in was refused — check the SSH user and the key or password")
	}
	return err
}
