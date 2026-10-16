package dbeaver

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

const sample = `{
  "folders": {"Team": {}, "Team/Backend": {"parent": "Team"}},
  "connections": {
    "postgres-jdbc-1": {"provider": "postgresql", "driver": "postgres-jdbc", "name": "Shop", "folder": "Team/Backend", "save-password": true,
      "configuration": {"host": "db.internal", "port": "5433", "database": "shop", "type": "prod", "read-only": true, "auth-model": "native",
        "handlers": {
          "ssh_tunnel": {"type": "TUNNEL", "enabled": true, "save-password": true, "properties": {"host": "bastion", "port": 2222, "authType": "PUBLIC_KEY", "keyPath": "${user.home}/.ssh/id_ed25519"}},
          "postgre_ssl": {"type": "CONFIG", "enabled": true, "properties": {"sslMode": "verify-ca"}}}}},
    "mysql8-2": {"provider": "mysql", "driver": "mysql8", "name": "Events", "save-password": false,
      "configuration": {"url": "jdbc:mysql://events.example.com:3307/events", "configurationType": "URL", "type": "test"}},
    "clickhouse-3": {"provider": "clickhouse", "driver": "com_clickhouse", "name": "Warehouse", "save-password": true,
      "configuration": {"host": "olap", "port": "8123", "database": "default", "type": "qa"}},
    "sqlite_jdbc-4": {"provider": "generic", "driver": "sqlite_jdbc", "name": "Music",
      "configuration": {"database": "~/Music/library.db", "type": "dev"}},
    "oracle-5": {"provider": "oracle", "driver": "oracle_thin", "name": "Ledger", "configuration": {"host": "ora", "port": "1521"}}
  },
  "connection-types": {"qa": {"name": "QA staging"}}
}`

const secrets = `{"postgres-jdbc-1": {"#connection": {"user": "app", "password": "s3cret"}, "network/ssh_tunnel": {"user": "deploy", "password": "key pass"}},
  "mysql8-2": {"#connection": {"user": "reporting"}}, "clickhouse-3": {"#connection": {"user": "default", "password": "ch"}}}`

func encrypt(t *testing.T, plain []byte) []byte {
	key, _ := hex.DecodeString(credentialsKey)
	block, _ := aes.NewCipher(key)
	pad := aes.BlockSize - len(plain)%aes.BlockSize
	for range pad {
		plain = append(plain, byte(pad))
	}
	iv := []byte("0123456789abcdef")
	out := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, plain)
	return append(iv, out...)
}

func TestRead(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "General", ".dbeaver")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "data-sources.json"), []byte(sample), 0o644)
	os.WriteFile(filepath.Join(dir, "credentials-config.json"), encrypt(t, []byte(secrets)), 0o644)
	found, err := Read(root)
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]Found{}
	for _, f := range found {
		by[f.Connection.Name] = f
	}
	home, _ := os.UserHomeDir()
	shop := by["Shop"].Connection
	want := model.Connection{Name: "Shop", Driver: model.Postgres, Host: "db.internal", Port: 5433, User: "app", Password: "s3cret", SavePassword: true,
		Database: "shop", SSLMode: "verify-full", Env: "prod", Group: "Team / Backend", ReadOnly: true,
		SSH: model.SSHTunnel{Enabled: true, Host: "bastion", Port: 2222, User: "deploy", Auth: model.SSHAuthKey, KeyFile: filepath.Join(home, ".ssh", "id_ed25519"), Passphrase: "key pass"}}
	if shop != want {
		t.Errorf("shop =\n%+v\nwant\n%+v", shop, want)
	}
	events := by["Events"].Connection
	if events.Driver != model.MySQL || events.Host != "events.example.com" || events.Port != 3307 || events.Database != "events" || events.User != "reporting" || events.SavePassword || events.Env != "staging" {
		t.Errorf("events = %+v", events)
	}
	wh := by["Warehouse"]
	if wh.Connection.Port != 9000 || wh.Note == "" || wh.Connection.Env != "staging" || wh.Connection.Password != "ch" {
		t.Errorf("warehouse = %+v", wh)
	}
	if music := by["Music"].Connection; music.Driver != model.SQLite || music.File != filepath.Join(home, "Music", "library.db") || music.Env != "dev" {
		t.Errorf("music = %+v", music)
	}
	if by["Ledger"].Skip == "" {
		t.Errorf("oracle should be skipped: %+v", by["Ledger"])
	}
}

func TestBrokenCredentials(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "data-sources.json"), []byte(sample), 0o644)
	os.WriteFile(filepath.Join(root, "credentials-config.json"), []byte("not encrypted at all, sorry"), 0o644)
	if _, err := Read(root); err == nil {
		t.Fatal("a broken credentials file should be an error")
	}
}

func TestWorkspaceDirs(t *testing.T) {
	j := filepath.Join
	cases := []struct {
		goos, home, config, xdg string
		want                    []string
	}{
		{"darwin", "/Users/ann", "", "", []string{j("/Users/ann", "Library", "DBeaverData", "workspace6")}},
		{"windows", `C:\Users\ann`, `C:\Users\ann\AppData\Roaming`, "", []string{j(`C:\Users\ann\AppData\Roaming`, "DBeaverData", "workspace6")}},
		{"linux", "/home/ann", "", "", []string{
			j("/home/ann", ".local", "share", "DBeaverData", "workspace6"),
			j("/home/ann", ".var", "app", "io.dbeaver.DBeaverCommunity", "data", "DBeaverData", "workspace6"),
			j("/home/ann", "snap", "dbeaver-ce", "current", ".local", "share", "DBeaverData", "workspace6"),
		}},
		{"linux", "/home/ann", "", "/data/ann", []string{
			j("/data/ann", "DBeaverData", "workspace6"),
			j("/home/ann", ".var", "app", "io.dbeaver.DBeaverCommunity", "data", "DBeaverData", "workspace6"),
			j("/home/ann", "snap", "dbeaver-ce", "current", ".local", "share", "DBeaverData", "workspace6"),
		}},
	}
	for _, c := range cases {
		if got := workspaceDirs(c.goos, c.home, c.config, c.xdg); !slices.Equal(got, c.want) {
			t.Errorf("%s: %q, want %q", c.goos, got, c.want)
		}
	}
}
