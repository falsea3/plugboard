package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
	"github.com/relay-client/relay-db/apps/desktop/internal/sshtunnel"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("RELAYDB_DATA_DIR", t.TempDir())
	t.Setenv("RELAYDB_DISABLE_KEYCHAIN", "1")
	return NewApp()
}

func TestTrustHostKeyNeedsTheKeyShown(t *testing.T) {
	a := newTestApp(t)
	a.changedKeys["db.example.com:22"] = &sshtunnel.HostKeyChangedError{Host: "db.example.com:22", Fingerprint: "SHA256:shown"}
	if err := a.TrustHostKey("db.example.com:22", "SHA256:other"); err == nil {
		t.Fatal("trusted a key the user wasn't shown")
	}
	if err := a.TrustHostKey("db.example.com:22", "SHA256:shown"); err == nil {
		t.Fatal("a refused change can be trusted later")
	}
}

func TestSessionInfoCarriesNoSecrets(t *testing.T) {
	a := newTestApp(t)
	file := filepath.Join(t.TempDir(), "x.db")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := a.SaveConnection(model.Connection{Name: "x", Driver: model.SQLite, File: file, SavePassword: true,
		Password: "pw", SSH: model.SSHTunnel{Password: "sshpw", Passphrase: "keypw"}})
	if err != nil {
		t.Fatal(err)
	}
	res, err := a.Connect(c.ID, model.ConnectSecrets{})
	if err != nil || res.Session == nil {
		t.Fatalf("connect: %+v %v", res, err)
	}
	defer a.Disconnect(res.Session.SessionID)
	if got := res.Session.Connection; got != got.WithoutSecrets() {
		t.Fatalf("session info carries secrets: %+v", got)
	}
}
