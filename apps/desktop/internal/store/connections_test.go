package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/relay-client/relay-db/apps/desktop/internal/model"
)

func newTestStore(t *testing.T) (*Connections, string) {
	t.Helper()
	dir := t.TempDir()
	return NewConnections(dir, &fileSecrets{path: filepath.Join(dir, "secrets.json")}), dir
}

func TestSaveKeepsPasswordOutOfConnectionsFile(t *testing.T) {
	s, dir := newTestStore(t)
	c, err := s.Save(model.Connection{
		Name: "local", Driver: model.Postgres, Host: "localhost",
		User: "app", Password: "hunter2", SavePassword: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.ID == "" || c.Password != "" {
		t.Fatalf("saved connection = %+v, want an id and no password", c)
	}
	data, err := os.ReadFile(filepath.Join(dir, "connections.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "hunter2") {
		t.Fatal("password leaked into connections.json")
	}
	got, err := s.Get(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Password != "hunter2" {
		t.Fatalf("Get password = %q, want hunter2", got.Password)
	}
}

func TestSaveWithEmptyPasswordKeepsStoredOne(t *testing.T) {
	s, _ := newTestStore(t)
	c, _ := s.Save(model.Connection{Name: "a", Driver: model.MySQL, Host: "h", Password: "pw", SavePassword: true})
	c.Name = "renamed"
	if _, err := s.Save(c); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(c.ID)
	if got.Name != "renamed" || got.Password != "pw" {
		t.Fatalf("got %+v", got)
	}
}

func TestTurningOffSavePasswordForgetsIt(t *testing.T) {
	s, _ := newTestStore(t)
	c, _ := s.Save(model.Connection{Name: "a", Driver: model.MySQL, Host: "h", Password: "pw", SavePassword: true})
	c.SavePassword = false
	if _, err := s.Save(c); err != nil {
		t.Fatal(err)
	}
	c.SavePassword = true
	if _, err := s.Save(c); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(c.ID)
	if got.Password != "" {
		t.Fatalf("password survived opt-out: %q", got.Password)
	}
}

func TestDeleteRemovesSecret(t *testing.T) {
	s, _ := newTestStore(t)
	c, _ := s.Save(model.Connection{Name: "a", Driver: model.SQLite, File: "/tmp/x.db", Password: "pw", SavePassword: true})
	if err := s.Delete(c.ID); err != nil {
		t.Fatal(err)
	}
	list, _ := s.List()
	if len(list) != 0 {
		t.Fatalf("list = %+v, want empty", list)
	}
	if _, err := s.secrets.Get(secretKey(c.ID, "password")); err != ErrSecretNotFound {
		t.Fatalf("secret still present: %v", err)
	}
}

func TestValidate(t *testing.T) {
	s, _ := newTestStore(t)
	bad := []model.Connection{
		{Driver: model.Postgres, Host: "h"},
		{Name: "x", Driver: model.Postgres},
		{Name: "x", Driver: model.SQLite},
		{Name: "x", Driver: "oracle", Host: "h"},
	}
	for _, c := range bad {
		if _, err := s.Save(c); err == nil {
			t.Errorf("Save(%+v) succeeded, want error", c)
		}
	}
}

func TestSSHSecretsStayOutOfFileAndRoundTrip(t *testing.T) {
	s, dir := newTestStore(t)
	c, err := s.Save(model.Connection{
		Name: "via bastion", Driver: model.Postgres, Host: "db.internal", Password: "dbpw", SavePassword: true,
		SSH: model.SSHTunnel{Enabled: true, Host: "bastion", User: "deploy", Auth: model.SSHAuthKey, KeyFile: "~/.ssh/id_ed25519", Passphrase: "keypw"},
	})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "connections.json"))
	for _, secret := range []string{"dbpw", "keypw"} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("%s leaked into connections.json", secret)
		}
	}
	got, _ := s.Get(c.ID)
	if got.Password != "dbpw" || got.SSH.Passphrase != "keypw" || got.SSH.KeyFile != "~/.ssh/id_ed25519" {
		t.Fatalf("got %+v", got)
	}
	if err := s.Delete(c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.secrets.Get(secretKey(c.ID, "ssh-passphrase")); err != ErrSecretNotFound {
		t.Fatalf("passphrase survived delete: %v", err)
	}
}

func TestValidateSSH(t *testing.T) {
	s, _ := newTestStore(t)
	base := model.Connection{Name: "x", Driver: model.MySQL, Host: "h"}
	for _, ssh := range []model.SSHTunnel{
		{Enabled: true, Host: "b", Auth: model.SSHAuthPassword},
		{Enabled: true, Host: "b", User: "u", Auth: model.SSHAuthKey},
		{Enabled: true, Host: "b", User: "u", Auth: "telepathy"},
	} {
		c := base
		c.SSH = ssh
		if _, err := s.Save(c); err == nil {
			t.Errorf("Save with %+v succeeded", ssh)
		}
	}
}

func TestEmptySSHHostIsAllowed(t *testing.T) {
	s, _ := newTestStore(t)
	_, err := s.Save(model.Connection{Name: "vps", Driver: model.Postgres, Host: "203.0.113.7",
		SSH: model.SSHTunnel{Enabled: true, User: "root", Auth: model.SSHAuthAgent}})
	if err != nil {
		t.Fatal(err)
	}
}

// A profile edited to point at another server must not take the old
// password there: not on Test, and not on the next Connect after Save.
func TestSecretsStayWithTheirServer(t *testing.T) {
	s, _ := newTestStore(t)
	c, _ := s.Save(model.Connection{
		Name: "a", Driver: model.Postgres, Host: "db.example.com", Port: 5432, User: "app", Password: "pw", SavePassword: true,
		SSH: model.SSHTunnel{Enabled: true, Host: "bastion", Port: 22, User: "deploy", Auth: model.SSHAuthPassword, Password: "sshpw"},
	})

	same := c
	if err := s.FillSecrets(&same); err != nil {
		t.Fatal(err)
	}
	if same.Password != "pw" || same.SSH.Password != "sshpw" {
		t.Fatalf("unchanged profile: %+v", same)
	}

	moved := c
	moved.Host = "evil.example.net"
	if err := s.FillSecrets(&moved); err != nil {
		t.Fatal(err)
	}
	if moved.Password != "" || moved.SSH.Password != "sshpw" {
		t.Fatalf("moved database host: password %q, SSH password %q", moved.Password, moved.SSH.Password)
	}

	if _, err := s.Save(moved); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(c.ID)
	if got.Password != "" || got.SSH.Password != "sshpw" {
		t.Fatalf("after saving the move: %+v", got)
	}
}

func TestWithoutSecretsClearsEverySecret(t *testing.T) {
	var c model.Connection
	for _, f := range secretFields {
		*f.field(&c) = "secret"
	}
	c = c.WithoutSecrets()
	for _, f := range secretFields {
		if *f.field(&c) != "" {
			t.Errorf("%s survived WithoutSecrets", f.name)
		}
	}
}
