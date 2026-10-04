package sshtunnel

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

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
