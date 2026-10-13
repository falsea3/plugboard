package cassandra

import (
	"context"
	"slices"
	"sort"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

var _ db.RoleManager = (*Session)(nil)

func roleForm() model.RoleForm {
	return model.RoleForm{
		Noun: "role",
		Flags: []model.RoleFlag{
			{Key: "login", Label: "Can log in", Hint: "A user rather than a group", Badge: "login"},
			{Key: "superuser", Label: "Superuser", Hint: "Every permission on every keyspace", Badge: "superuser"},
		},
		Membership: true,
		New:        model.Role{Flags: map[string]bool{"login": true}, ConnLimit: -1, MemberOf: []string{}, Grants: []string{}},
	}
}

func roleName(name string) string { return "'" + strings.ReplaceAll(name, "'", "''") + "'" }

func (s *Session) Roles(ctx context.Context) (model.RoleList, error) {
	gs, err := s.main()
	if err != nil {
		return model.RoleList{}, err
	}
	out := model.RoleList{Form: roleForm(), Roles: []model.Role{}}
	iter := gs.Query(`SELECT role, can_login, is_superuser, member_of FROM system_auth.roles`).WithContext(ctx).Iter()
	var name string
	var login, super bool
	var memberOf []string
	for iter.Scan(&name, &login, &super, &memberOf) {
		sort.Strings(memberOf)
		out.Roles = append(out.Roles, model.Role{
			ID: name, Name: name, Flags: map[string]bool{"login": login, "superuser": super}, ConnLimit: -1,
			MemberOf: append([]string{}, memberOf...), Grants: s.permissions(ctx, name), Grantable: true, Current: name == s.conn.User,
		})
		memberOf = nil
	}
	if err := iter.Close(); err != nil {
		return model.RoleList{}, err
	}
	sort.Slice(out.Roles, func(i, j int) bool { return out.Roles[i].Name < out.Roles[j].Name })
	return out, nil
}

func (s *Session) permissions(ctx context.Context, name string) []string {
	out := []string{}
	gs, err := s.main()
	if err != nil {
		return out
	}
	iter := gs.Query(`LIST ALL PERMISSIONS OF ` + roleName(name) + ` NORECURSIVE`).WithContext(ctx).Iter()
	var role, user, resource, permission string
	for iter.Scan(&role, &user, &resource, &permission) {
		out = append(out, "GRANT "+permission+" ON "+cqlResource(resource)+" TO "+roleName(name))
	}
	iter.Close()
	return out
}

func cqlResource(r string) string {
	switch {
	case r == "<all keyspaces>":
		return "ALL KEYSPACES"
	case strings.HasPrefix(r, "<keyspace ") && strings.HasSuffix(r, ">"):
		return "KEYSPACE " + strings.TrimSuffix(strings.TrimPrefix(r, "<keyspace "), ">")
	case strings.HasPrefix(r, "<table ") && strings.HasSuffix(r, ">"):
		return "TABLE " + strings.TrimSuffix(strings.TrimPrefix(r, "<table "), ">")
	case r == "<all roles>":
		return "ALL ROLES"
	case strings.HasPrefix(r, "<role ") && strings.HasSuffix(r, ">"):
		return "ROLE " + strings.TrimSuffix(strings.TrimPrefix(r, "<role "), ">")
	}
	return r
}

func rolePlan(from *model.Role, to model.RoleSpec, preview bool) ([]string, error) {
	name := strings.TrimSpace(to.Name)
	if name == "" {
		return nil, apperr.New("bad_name", "give the role a name")
	}
	if from != nil && from.Name != name {
		return nil, apperr.New("not_supported", "Cassandra can't rename a role — create a new one and drop this one")
	}
	before := model.Role{Flags: map[string]bool{}}
	if from != nil {
		before = *from
	}
	var opts []string
	switch {
	case to.Password != "" && preview:
		opts = append(opts, "PASSWORD = '********'")
	case to.Password != "":
		opts = append(opts, "PASSWORD = "+roleName(to.Password))
	}
	for _, f := range []struct{ key, word string }{{"login", "LOGIN"}, {"superuser", "SUPERUSER"}} {
		if from == nil || before.Flags[f.key] != to.Flags[f.key] {
			opts = append(opts, f.word+" = "+map[bool]string{true: "true", false: "false"}[to.Flags[f.key]])
		}
	}
	var out []string
	switch {
	case from == nil:
		out = append(out, "CREATE ROLE "+roleName(name)+" WITH "+strings.Join(opts, " AND "))
	case len(opts) > 0:
		out = append(out, "ALTER ROLE "+roleName(name)+" WITH "+strings.Join(opts, " AND "))
	}
	for _, g := range to.MemberOf {
		if !slices.Contains(before.MemberOf, g) {
			out = append(out, "GRANT "+roleName(g)+" TO "+roleName(name))
		}
	}
	for _, g := range before.MemberOf {
		if !slices.Contains(to.MemberOf, g) {
			out = append(out, "REVOKE "+roleName(g)+" FROM "+roleName(name))
		}
	}
	return out, nil
}

func (s *Session) find(ctx context.Context, id string) (*model.Role, []model.Role, error) {
	list, err := s.Roles(ctx)
	if err != nil || id == "" {
		return nil, list.Roles, err
	}
	i := slices.IndexFunc(list.Roles, func(r model.Role) bool { return r.ID == id })
	if i < 0 {
		return nil, nil, apperr.New("row_gone", id+" no longer exists — refresh the list")
	}
	return &list.Roles[i], list.Roles, nil
}

func (s *Session) plan(ctx context.Context, from string, to model.RoleSpec, preview bool) ([]string, error) {
	current, all, err := s.find(ctx, from)
	if err != nil {
		return nil, err
	}
	for _, g := range to.MemberOf {
		if !slices.ContainsFunc(all, func(r model.Role) bool { return r.ID == g }) {
			return nil, apperr.New("bad_name", "can't add membership in "+g+": no such role")
		}
	}
	return rolePlan(current, to, preview)
}

func (s *Session) RoleSQL(ctx context.Context, from string, to model.RoleSpec) ([]string, error) {
	out, err := s.plan(ctx, from, to, true)
	if out == nil {
		out = []string{}
	}
	return out, err
}

func (s *Session) SaveRole(ctx context.Context, from string, to model.RoleSpec) (string, error) {
	if s.conn.ReadOnly {
		return "", &db.ReadOnlyError{Reason: "role changes"}
	}
	stmts, err := s.plan(ctx, from, to, false)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(to.Name), s.execAll(ctx, stmts)
}

func (s *Session) DropRole(ctx context.Context, id string) error {
	if s.conn.ReadOnly {
		return &db.ReadOnlyError{Reason: "role changes"}
	}
	if _, _, err := s.find(ctx, id); err != nil || id == "" {
		return err
	}
	return s.execAll(ctx, []string{"DROP ROLE " + roleName(id)})
}

func (s *Session) execAll(ctx context.Context, stmts []string) error {
	gs, err := s.main()
	if err != nil {
		return err
	}
	for _, stmt := range stmts {
		if err := gs.Query(stmt).WithContext(ctx).Exec(); err != nil {
			return err
		}
	}
	return nil
}
