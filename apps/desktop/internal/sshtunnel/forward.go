package sshtunnel

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

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
