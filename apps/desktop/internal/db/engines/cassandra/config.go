package cassandra

import (
	"context"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

var _ db.ServerConfig = (*Session)(nil)

const (
	currentSettings = "system_views.settings"
	defaultYAML     = "/etc/cassandra/cassandra.yaml"
)

func (s *Session) ConfigFiles(context.Context) ([]model.ConfigFile, error) {
	return []model.ConfigFile{{Name: "Current settings", Path: currentSettings}, {Name: "cassandra.yaml", Path: defaultYAML}}, nil
}

func (s *Session) ReadConfig(ctx context.Context, file string) (model.ConfigText, error) {
	if file == currentSettings {
		gs, err := s.main()
		if err != nil {
			return model.ConfigText{}, err
		}
		var b strings.Builder
		b.WriteString("# What this node runs with now. Read-only: change cassandra.yaml and restart the node.\n")
		iter := gs.Query(`SELECT name, value FROM system_views.settings`).WithContext(ctx).Iter()
		var name string
		var value *string
		for iter.Scan(&name, &value) {
			v := "null"
			if value != nil {
				v = *value
			}
			b.WriteString(name + ": " + v + "\n")
		}
		if err := iter.Close(); err != nil {
			return model.ConfigText{}, err
		}
		return model.ConfigText{Path: file, Text: b.String(), Via: "live", ReadOnly: true}, nil
	}
	files, err := s.hostFiles(file)
	if err != nil {
		return model.ConfigText{}, err
	}
	data, err := files.ReadFile(ctx, file)
	if err != nil {
		return model.ConfigText{}, err
	}
	return model.ConfigText{Path: file, Text: string(data), Via: "ssh"}, nil
}

func (s *Session) hostFiles(file string) (db.HostFiles, error) {
	if strings.TrimSpace(file) == "" {
		return nil, apperr.New("bad_value", "enter the path of the file on the server")
	}
	files := db.DatabaseHostFiles(s.conn, s.tunnel)
	if files == nil {
		return nil, apperr.New("not_supported", "Cassandra can't read files itself — connect through SSH to the machine the node runs on to open "+file)
	}
	return files, nil
}

func (s *Session) WriteConfig(ctx context.Context, file, text string) (model.ConfigSaved, error) {
	if file == currentSettings {
		return model.ConfigSaved{}, apperr.New("not_supported", "these settings are read-only — change cassandra.yaml and restart the node")
	}
	if s.conn.ReadOnly {
		return model.ConfigSaved{}, &db.ReadOnlyError{Reason: "configuration changes"}
	}
	files, err := s.hostFiles(file)
	if err != nil {
		return model.ConfigSaved{}, err
	}
	if err := files.WriteFile(ctx, file, []byte(text)); err != nil {
		return model.ConfigSaved{}, err
	}
	return model.ConfigSaved{Via: "ssh", Problems: []model.ConfigProblem{}, Restart: []string{},
		Note: "Saved. Cassandra reads cassandra.yaml only when it starts — restart the node."}, nil
}
