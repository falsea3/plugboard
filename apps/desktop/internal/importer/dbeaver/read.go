package dbeaver

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

type Found struct {
	Key        string
	Project    string
	Connection model.Connection
	Note       string
	Skip       string
}

func Read(root string) ([]Found, error) {
	var out []Found
	ps := projects(root)
	if len(ps) == 0 {
		return nil, fmt.Errorf("no DBeaver connections in %s", root)
	}
	for _, p := range ps {
		sources, creds, err := readProject(p)
		if err != nil {
			return nil, err
		}
		for _, s := range sources {
			ids := make([]string, 0, len(s.Connections))
			for id := range s.Connections {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			for _, id := range ids {
				f := convert(id, s.Connections[id], creds[id], s)
				f.Key, f.Project = p.name+"/"+id, p.name
				if len(ps) > 1 && f.Connection.Group == "" {
					f.Connection.Group = p.name
				}
				out = append(out, f)
			}
		}
	}
	return out, nil
}

func convert(id string, c dsConnection, creds credentials, s source) Found {
	f := Found{Connection: model.Connection{Name: strings.TrimSpace(c.Name), Group: strings.ReplaceAll(strings.Trim(c.Folder, "/"), "/", " / ")}}
	conn := &f.Connection
	if conn.Name == "" {
		conn.Name = id
	}
	conn.Driver = engineOf(c)
	if conn.Driver == "" {
		f.Skip = "Plugboard doesn't connect to " + firstOf(c.Provider, c.Driver) + " databases"
		return f
	}
	cfg := c.Config
	conn.Host, conn.Database = cfg.Host, cfg.Database
	conn.Port, _ = strconv.Atoi(cfg.Port.String())
	if conn.Host == "" && cfg.URL != "" {
		fromURL(conn, cfg.URL)
	}
	if conn.Driver == model.SQLite {
		if conn.File == "" {
			conn.File = cfg.Database
		}
		conn.File, conn.Host, conn.Port, conn.Database = expand(conn.File), "", 0, ""
	} else if conn.Port == 0 {
		conn.Port = defaultPorts[conn.Driver]
	}
	if conn.Driver == model.Redis && conn.Database == "" {
		conn.Database = "0"
	}
	conn.SSLMode = sslOf(cfg.Handlers)
	if conn.Driver == model.ClickHouse {
		switch conn.Port {
		case 8123:
			conn.Port, f.Note = 9000, "port 8123 (HTTP) becomes 9000, ClickHouse's native port"
		case 8443:
			conn.Port, conn.SSLMode, f.Note = 9440, "verify-full", "port 8443 (HTTPS) becomes 9440, ClickHouse's native TLS port"
		}
	}
	conn.Env = envOf(cfg.Type, s.Types)
	conn.ReadOnly = cfg.ReadOnly
	login := creds["#connection"]
	conn.User, conn.Password = firstOf(login.User, cfg.User), firstOf(login.Password, cfg.Password)
	conn.SavePassword = c.SavePassword
	for key, h := range cfg.Handlers {
		if h.Type == "TUNNEL" && h.Enabled {
			tunnel(conn, h, creds["network/"+key])
		}
	}
	return f
}

func tunnel(conn *model.Connection, h dsHandler, login secret) {
	p := h.Properties
	ssh := model.SSHTunnel{Enabled: true, Host: p["host"], User: firstOf(login.User, h.User, p["user"])}
	ssh.Port, _ = strconv.Atoi(p["port"])
	if ssh.Port == 0 {
		ssh.Port = 22
	}
	secret := firstOf(login.Password, h.Password)
	switch strings.ToUpper(p["authType"]) {
	case "AGENT":
		ssh.Auth = model.SSHAuthAgent
	case "PUBLIC_KEY":
		ssh.Auth, ssh.KeyFile, ssh.Passphrase = model.SSHAuthKey, expand(p["keyPath"]), secret
	case "PASSWORD":
		ssh.Auth, ssh.Password = model.SSHAuthPassword, secret
	default:
		if p["keyPath"] != "" {
			ssh.Auth, ssh.KeyFile, ssh.Passphrase = model.SSHAuthKey, expand(p["keyPath"]), secret
		} else {
			ssh.Auth, ssh.Password = model.SSHAuthPassword, secret
		}
	}
	conn.SSH = ssh
	if secret != "" {
		conn.SavePassword = true
	}
}

func firstOf(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
