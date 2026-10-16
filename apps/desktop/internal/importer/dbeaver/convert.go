package dbeaver

import (
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

var defaultPorts = map[model.Driver]int{model.Postgres: 5432, model.MySQL: 3306, model.ClickHouse: 9000, model.Cassandra: 9042, model.Redis: 6379}

func engineOf(c dsConnection) model.Driver {
	for _, name := range []string{strings.ToLower(c.Provider), strings.ToLower(c.Driver)} {
		switch {
		case strings.Contains(name, "postgres"):
			return model.Postgres
		case strings.Contains(name, "mysql"), strings.Contains(name, "mariadb"):
			return model.MySQL
		case strings.Contains(name, "sqlite"):
			return model.SQLite
		case strings.Contains(name, "clickhouse"):
			return model.ClickHouse
		case strings.Contains(name, "cassandra"):
			return model.Cassandra
		case strings.Contains(name, "redis"):
			return model.Redis
		}
	}
	return ""
}

var envByType = map[string]string{"dev": "dev", "test": "staging", "prod": "prod"}

func envOf(typeID string, types map[string]struct {
	Name string `json:"name"`
}) string {
	if env, ok := envByType[typeID]; ok {
		return env
	}
	name := strings.ToLower(types[typeID].Name + " " + typeID)
	for _, guess := range []struct{ word, env string }{{"prod", "prod"}, {"stag", "staging"}, {"test", "staging"}, {"dev", "dev"}, {"local", "local"}} {
		if strings.Contains(name, guess.word) {
			return guess.env
		}
	}
	return ""
}

func expand(path string) string {
	home, _ := os.UserHomeDir()
	path = strings.ReplaceAll(path, "${user.home}", home)
	if rest, ok := strings.CutPrefix(path, "~"); ok && (rest == "" || rest[0] == '/' || rest[0] == '\\') {
		path = home + rest
	}
	return filepath.Clean(path)
}

func fromURL(c *model.Connection, raw string) {
	rest := strings.TrimPrefix(raw, "jdbc:")
	if file, ok := strings.CutPrefix(rest, "sqlite:"); ok {
		c.File = file
		return
	}
	u, err := url.Parse(rest)
	if err != nil {
		return
	}
	c.Host = u.Hostname()
	c.Port, _ = strconv.Atoi(u.Port())
	c.Database = strings.TrimPrefix(u.Path, "/")
}

var sslModes = map[string]string{"disable": "disable", "allow": "", "prefer": "", "require": "require", "verify-ca": "verify-full", "verify-full": "verify-full"}

func sslOf(handlers map[string]dsHandler) string {
	for key, h := range handlers {
		if !h.Enabled || !strings.Contains(strings.ToLower(key), "ssl") {
			continue
		}
		for _, prop := range []string{"sslMode", "ssl.mode", "sslmode"} {
			if mode, ok := sslModes[strings.ToLower(h.Properties[prop])]; ok && h.Properties[prop] != "" {
				return mode
			}
		}
		return "require"
	}
	return ""
}
