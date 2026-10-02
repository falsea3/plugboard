package api

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/relay-client/plugboard/apps/desktop/internal/store"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
	"github.com/relay-client/plugboard/apps/desktop/internal/sshtunnel"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	t.Setenv("PLUGBOARD_DATA_DIR", t.TempDir())
	t.Setenv("PLUGBOARD_DISABLE_KEYCHAIN", "1")
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

func TestMovesTheRelayDBFolderOnce(t *testing.T) {
	base := t.TempDir()
	old := filepath.Join(base, "Relay DB")
	dir := filepath.Join(base, "Plugboard")
	if err := os.MkdirAll(old, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "connections.json"), []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !moveOldDataDir(dir) {
		t.Fatal("the Relay DB folder wasn't moved")
	}
	if _, err := os.Stat(filepath.Join(dir, "connections.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(old, 0o700); err != nil {
		t.Fatal(err)
	}
	if moveOldDataDir(dir) {
		t.Fatal("moved over a Plugboard folder that already exists")
	}
}

func TestMovesOldSecretsAndRemembersIt(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PLUGBOARD_DISABLE_KEYCHAIN", "1")
	conns := store.NewConnections(dir, store.NewSecrets(dir))
	c, err := conns.Save(model.Connection{Name: "db", Driver: model.Postgres, Host: "h", User: "u", SavePassword: true})
	if err != nil {
		t.Fatal(err)
	}
	old := store.NewSecrets(filepath.Join(dir, "old"))
	if err := os.MkdirAll(filepath.Join(dir, "old"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := old.Set("connection:"+c.ID+":password", "hunter2"); err != nil {
		t.Fatal(err)
	}
	moveOldSecrets(dir, conns, old)
	if got, _ := conns.Get(c.ID); got.Password != "hunter2" {
		t.Fatalf("password after the move = %q", got.Password)
	}
	if _, err := os.Stat(filepath.Join(dir, ".secrets-moved")); err != nil {
		t.Fatal("the move wasn't recorded, so it would run on every launch")
	}
}
