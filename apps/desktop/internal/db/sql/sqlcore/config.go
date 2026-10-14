package sqlcore

import (
	"context"
	"errors"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/apperr"
	"github.com/falsea3/plugboard/apps/desktop/internal/db"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

var _ db.ServerConfig = (*Session)(nil)

const viaSSH = "ssh"

var errNoPath = apperr.New("bad_value", "enter the path of the file on the server")

func (s *Session) serverConfig() (dialect.ServerConfig, error) {
	if c, ok := s.Dialect.(dialect.ServerConfig); ok {
		return c, nil
	}
	return nil, db.ErrNotSupported
}

func (s *Session) ConfigFiles(ctx context.Context) ([]model.ConfigFile, error) {
	c, err := s.serverConfig()
	if err != nil {
		return nil, err
	}
	return retry(ctx, s, func() ([]model.ConfigFile, error) { return c.ConfigFiles(ctx, s.DB) })
}

func (s *Session) ReadConfig(ctx context.Context, path string) (model.ConfigText, error) {
	c, err := s.serverConfig()
	if err != nil {
		return model.ConfigText{}, err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return model.ConfigText{}, errNoPath
	}
	if files := s.hostFiles(path); files != nil {
		data, err := files.ReadFile(ctx, path)
		if err == nil {
			return model.ConfigText{Path: path, Text: string(data), Via: viaSSH}, nil
		}
		if !errors.Is(err, db.ErrNoFile) {
			return model.ConfigText{}, err
		}
	}
	return retry(ctx, s, func() (model.ConfigText, error) { return c.ReadServerFile(ctx, s.DB, path) })
}

func (s *Session) hostFiles(path string) db.HostFiles {
	if !strings.HasPrefix(path, "/") {
		return nil
	}
	return db.DatabaseHostFiles(s.Conn, s.tunnel)
}

func (s *Session) WriteConfig(ctx context.Context, path, text string) (model.ConfigSaved, error) {
	c, err := s.serverConfig()
	if err != nil {
		return model.ConfigSaved{}, err
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return model.ConfigSaved{}, errNoPath
	}
	if s.Conn.ReadOnly {
		return model.ConfigSaved{}, &db.ReadOnlyError{Reason: "configuration changes"}
	}
	via, err := s.writeFile(ctx, c, path, text)
	if err != nil {
		return model.ConfigSaved{}, err
	}
	problems, restart := c.CheckConfig(ctx, s.DB, path)
	if restart == nil {
		restart = []string{}
	}
	saved := model.ConfigSaved{Via: via, Problems: problems, Restart: restart}
	if len(problems) > 0 {
		return saved, nil
	}
	if err := c.ReloadConfig(ctx, s.DB); err != nil {
		saved.Problems = append(saved.Problems, model.ConfigProblem{Message: err.Error()})
		return saved, nil
	}
	saved.Reloaded = true
	return saved, nil
}

func (s *Session) writeFile(ctx context.Context, c dialect.ServerConfig, path, text string) (string, error) {
	if files := s.hostFiles(path); files != nil {
		err := files.WriteFile(ctx, path, []byte(text))
		if err == nil {
			return viaSSH, nil
		}
		if !errors.Is(err, db.ErrNoFile) {
			return "", err
		}
	}
	return "sql", c.WriteServerFile(ctx, s.DB, path, text)
}
