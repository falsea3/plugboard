package redis

import (
	"context"
	"path"
	"sort"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/apperr"
	"github.com/falsea3/plugboard/apps/desktop/internal/db"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

var _ db.ServerConfig = (*Session)(nil)

const liveSettings = "CONFIG GET *"

const liveHeader = "# Live settings. Change a value and save: each change runs CONFIG SET, then CONFIG REWRITE keeps it in the config file.\n"

func (s *Session) ConfigFiles(ctx context.Context) ([]model.ConfigFile, error) {
	c, err := s.admin()
	if err != nil {
		return nil, err
	}
	files := []model.ConfigFile{{Name: "Live settings", Path: liveSettings}}
	info, err := c.Info(ctx, "server").Result()
	if err != nil {
		return nil, classify(err)
	}
	for line := range strings.Lines(info) {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "config_file:"); ok && v != "" {
			files = append(files, model.ConfigFile{Name: path.Base(v), Path: v})
		}
	}
	if acl, err := c.ConfigGet(ctx, "aclfile").Result(); err == nil && acl["aclfile"] != "" {
		files = append(files, model.ConfigFile{Name: "ACL file", Path: acl["aclfile"]})
	}
	return files, nil
}

func (s *Session) ReadConfig(ctx context.Context, file string) (model.ConfigText, error) {
	if file == liveSettings {
		c, err := s.admin()
		if err != nil {
			return model.ConfigText{}, err
		}
		all, err := c.ConfigGet(ctx, "*").Result()
		if err != nil {
			return model.ConfigText{}, classify(err)
		}
		return model.ConfigText{Path: file, Text: liveHeader + liveText(all), Via: "live"}, nil
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
		return nil, apperr.New("not_supported", "Redis can't read files itself — connect through SSH to the machine Redis runs on to open "+file+", or change Live settings")
	}
	return files, nil
}

func liveText(all map[string]string) string {
	names := make([]string, 0, len(all))
	for k := range all {
		names = append(names, k)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, k := range names {
		v := all[k]
		if v == "" {
			v = `""`
		}
		b.WriteString(k + " " + v + "\n")
	}
	return b.String()
}

type setting struct {
	line        int
	name, value string
}

func liveSettingsOf(text string) []setting {
	var out []setting
	n := 0
	for line := range strings.Lines(text) {
		n++
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, _ := strings.Cut(line, " ")
		value = strings.TrimSpace(value)
		if value == `""` {
			value = ""
		}
		out = append(out, setting{line: n, name: strings.ToLower(name), value: value})
	}
	return out
}

func (s *Session) WriteConfig(ctx context.Context, file, text string) (model.ConfigSaved, error) {
	if s.conn.ReadOnly {
		return model.ConfigSaved{}, &db.ReadOnlyError{Reason: "configuration changes"}
	}
	c, err := s.admin()
	if err != nil {
		return model.ConfigSaved{}, err
	}
	saved := model.ConfigSaved{Problems: []model.ConfigProblem{}, Restart: []string{}}
	if file == liveSettings {
		saved.Via = "live"
		current, err := c.ConfigGet(ctx, "*").Result()
		if err != nil {
			return saved, classify(err)
		}
		changed := 0
		for _, st := range liveSettingsOf(text) {
			if was, ok := current[st.name]; ok && was == st.value {
				continue
			}
			if err := c.ConfigSet(ctx, st.name, st.value).Err(); err != nil {
				saved.Problems = append(saved.Problems, model.ConfigProblem{Line: st.line, Message: err.Error()})
				continue
			}
			changed++
		}
		saved.Reloaded = len(saved.Problems) == 0
		if changed > 0 {
			if err := c.ConfigRewrite(ctx).Err(); err != nil {
				saved.Note = "Applied, but not written to a config file (" + err.Error() + ") — the change lasts until Redis restarts."
			}
		}
		return saved, nil
	}
	files, err := s.hostFiles(file)
	if err != nil {
		return saved, err
	}
	saved.Via = "ssh"
	if err := files.WriteFile(ctx, file, []byte(text)); err != nil {
		return saved, err
	}
	acl, _ := c.ConfigGet(ctx, "aclfile").Result()
	if acl["aclfile"] != file {
		saved.Note = "Saved. Redis reads its config file only when it starts — restart it, or change Live settings for now."
		return saved, nil
	}
	if err := c.Do(ctx, "ACL", "LOAD").Err(); err != nil {
		saved.Problems = append(saved.Problems, model.ConfigProblem{Line: 0, Message: err.Error()})
		return saved, nil
	}
	saved.Reloaded = true
	return saved, nil
}
