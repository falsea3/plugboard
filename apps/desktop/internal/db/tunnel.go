package db

import (
	"context"
	"io"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
	"github.com/falsea3/plugboard/apps/desktop/internal/sshtunnel"
)

func OpenTunnel(ctx context.Context, c model.Connection, defaultPort int, opts OpenOptions) (model.Connection, io.Closer, error) {
	if !c.SSH.Enabled || defaultPort == 0 {
		return c, nil, nil
	}
	port := c.Port
	if port == 0 {
		port = defaultPort
	}
	ssh, target := TunnelEndpoints(c)
	tunnel, err := sshtunnel.Open(ctx, ssh, target, port, sshtunnel.Options{KnownHostsFile: opts.KnownHostsFile, OnState: opts.OnTunnelState})
	if err != nil {
		return c, nil, err
	}
	dial := c
	dial.Host = "127.0.0.1"
	dial.Port = tunnel.LocalPort()
	return dial, tunnel, nil
}

func TunnelEndpoints(c model.Connection) (ssh model.SSHTunnel, targetHost string) {
	ssh = c.SSH
	ssh.Host = c.SSHHost()
	if strings.TrimSpace(c.SSH.Host) == "" {
		return ssh, "127.0.0.1"
	}
	return ssh, c.Host
}

type HostFiles interface {
	ReadFile(ctx context.Context, path string) ([]byte, error)
	WriteFile(ctx context.Context, path string, data []byte) error
}

var ErrNoFile = sshtunnel.ErrNoFile

func DatabaseHostFiles(c model.Connection, tunnel io.Closer) HostFiles {
	files, ok := tunnel.(HostFiles)
	if !ok || !c.SSH.Enabled {
		return nil
	}
	_, target := TunnelEndpoints(c)
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "127.0.0.1", "localhost", "::1":
		return files
	}
	return nil
}
