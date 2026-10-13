package redis

import (
	"context"
	"slices"
	"strconv"
	"strings"

	goredis "github.com/redis/go-redis/v9"
	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

var _ db.RoleManager = (*Session)(nil)

func roleForm() model.RoleForm {
	return model.RoleForm{
		Noun: "user",
		Flags: []model.RoleFlag{
			{Key: "disabled", Label: "Disabled", Hint: "Can't log in", Badge: "off"},
			{Key: "nopass", Label: "No password", Hint: "Any password logs in", Badge: "nopass"},
		},
		Rules: "What it may touch, in ACL rules: ~cache:* +@read -@dangerous — empty allows nothing",
		New:   model.Role{Flags: map[string]bool{}, ConnLimit: -1, MemberOf: []string{}, Grants: []string{}},
	}
}

func (s *Session) admin() (*goredis.Client, error) { return s.client(0, true) }

func (s *Session) Roles(ctx context.Context) (model.RoleList, error) {
	c, err := s.admin()
	if err != nil {
		return model.RoleList{}, err
	}
	lines, err := c.Do(ctx, "ACL", "LIST").StringSlice()
	if err != nil {
		return model.RoleList{}, classify(err)
	}
	me, _ := c.Do(ctx, "ACL", "WHOAMI").Text()
	out := model.RoleList{Form: roleForm(), Roles: []model.Role{}}
	for _, line := range lines {
		if r, ok := parseACL(line); ok {
			r.Current = r.Name == me
			out.Roles = append(out.Roles, r)
		}
	}
	return out, nil
}

func parseACL(line string) (model.Role, bool) {
	tokens := ruleTokens(line)
	if len(tokens) < 2 || tokens[0] != "user" {
		return model.Role{}, false
	}
	r := model.Role{ID: tokens[1], Name: tokens[1], Flags: map[string]bool{"disabled": true}, ConnLimit: -1, MemberOf: []string{}, Grants: []string{}}
	var rules []string
	for _, t := range tokens[2:] {
		switch {
		case t == "on":
			r.Flags["disabled"] = false
		case t == "off":
			r.Flags["disabled"] = true
		case t == "nopass":
			r.Flags["nopass"] = true
		case t == "sanitize-payload", strings.HasPrefix(t, "#"):
		default:
			rules = append(rules, t)
		}
	}
	r.Rules = strings.Join(rules, " ")
	return r, true
}

func ruleTokens(s string) []string {
	var out []string
	depth, start := 0, -1
	for i, ch := range s {
		switch {
		case ch == '(':
			if depth == 0 && start < 0 {
				start = i
			}
			depth++
		case ch == ')' && depth > 0:
			depth--
		case (ch == ' ' || ch == '\t' || ch == '\n') && depth == 0:
			if start >= 0 {
				out = append(out, s[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		out = append(out, s[start:])
	}
	return out
}

func aclPlan(from *model.Role, to model.RoleSpec, preview bool) ([]string, error) {
	name := strings.TrimSpace(to.Name)
	if name == "" || strings.ContainsAny(name, " \t\n") {
		return nil, apperr.New("bad_name", "give the user a name without spaces")
	}
	if from != nil && from.Name != name {
		return nil, apperr.New("not_supported", "Redis can't rename a user — create a new one and delete this one")
	}
	before := model.Role{Flags: map[string]bool{"disabled": true}}
	if from != nil {
		before = *from
	}
	var args []string
	if from == nil || before.Flags["disabled"] != to.Flags["disabled"] {
		args = append(args, map[bool]string{true: "off", false: "on"}[to.Flags["disabled"]])
	}
	password := to.Password != "" || to.ClearPassword
	if password || (before.Flags["nopass"] && !to.Flags["nopass"]) {
		args = append(args, "resetpass")
	}
	if to.Password != "" && !to.Flags["nopass"] {
		args = append(args, ">"+map[bool]string{true: "********", false: to.Password}[preview])
	}
	if to.Flags["nopass"] && (!before.Flags["nopass"] || password) {
		args = append(args, "nopass")
	}
	rules := ruleTokens(to.Rules)
	if strings.Join(rules, " ") != strings.Join(ruleTokens(before.Rules), " ") {
		if from != nil {
			args = append(args, "clearselectors", "resetkeys", "resetchannels", "nocommands")
		}
		args = append(args, rules...)
	}
	if len(args) == 0 {
		return nil, nil
	}
	return append([]string{"ACL", "SETUSER", name}, args...), nil
}

func (s *Session) find(ctx context.Context, id string) (*model.Role, error) {
	if id == "" {
		return nil, nil
	}
	list, err := s.Roles(ctx)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(list.Roles, func(r model.Role) bool { return r.ID == id })
	if i < 0 {
		return nil, apperr.New("row_gone", id+" no longer exists — refresh the list")
	}
	return &list.Roles[i], nil
}

func (s *Session) RoleSQL(ctx context.Context, from string, to model.RoleSpec) ([]string, error) {
	current, err := s.find(ctx, from)
	if err != nil {
		return nil, err
	}
	cmd, err := aclPlan(current, to, true)
	if err != nil || cmd == nil {
		return []string{}, err
	}
	return []string{shown(cmd)}, nil
}

func shown(cmd []string) string {
	parts := make([]string, len(cmd))
	for i, a := range cmd {
		parts[i] = a
		if a == "" || (strings.ContainsAny(a, " \"'") && !strings.HasPrefix(a, "(")) {
			parts[i] = strconv.Quote(a)
		}
	}
	return strings.Join(parts, " ")
}

func (s *Session) SaveRole(ctx context.Context, from string, to model.RoleSpec) (string, error) {
	if s.conn.ReadOnly {
		return "", &db.ReadOnlyError{Reason: "user changes"}
	}
	current, err := s.find(ctx, from)
	if err != nil {
		return "", err
	}
	cmd, err := aclPlan(current, to, false)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(to.Name)
	if cmd == nil {
		return name, nil
	}
	return name, s.aclRun(ctx, cmd)
}

func (s *Session) DropRole(ctx context.Context, id string) error {
	if s.conn.ReadOnly {
		return &db.ReadOnlyError{Reason: "user changes"}
	}
	if _, err := s.find(ctx, id); err != nil || id == "" {
		return err
	}
	return s.aclRun(ctx, []string{"ACL", "DELUSER", id})
}

func (s *Session) aclRun(ctx context.Context, cmd []string) error {
	c, err := s.admin()
	if err != nil {
		return err
	}
	args := make([]any, len(cmd))
	for i, a := range cmd {
		args[i] = a
	}
	if err := c.Do(ctx, args...).Err(); err != nil {
		return classify(err)
	}
	if c.Do(ctx, "ACL", "SAVE").Err() != nil {
		c.Do(ctx, "CONFIG", "REWRITE")
	}
	return nil
}
