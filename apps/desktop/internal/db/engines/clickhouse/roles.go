package clickhouse

import (
	"context"
	"database/sql"
	"slices"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/apperr"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

var _ dialect.Roles = Dialect{}

const rolePrefix = "role:"

func (Dialect) RoleForm() model.RoleForm {
	return model.RoleForm{
		Noun:       "user",
		Flags:      []model.RoleFlag{{Key: "full", Label: "All privileges", Hint: "GRANT ALL ON *.*", Badge: "all privileges"}},
		Membership: true,
		New:        model.Role{Flags: map[string]bool{}, ConnLimit: -1, MemberOf: []string{}, Grants: []string{}},
	}
}

func isRole(r model.Role) bool { return r.Kind == "role" }

func (d Dialect) ListRoles(ctx context.Context, db *sql.DB) ([]model.Role, error) {
	var current string
	if err := db.QueryRowContext(ctx, `SELECT currentUser()`).Scan(&current); err != nil {
		return nil, err
	}
	full, err := stringSet(ctx, db, `SELECT DISTINCT coalesce(user_name, role_name) FROM system.grants WHERE access_type = 'ALL' AND database IS NULL AND is_partial_revoke = 0`)
	if err != nil {
		return nil, err
	}
	members := map[string][]string{}
	rows, err := db.QueryContext(ctx, `SELECT coalesce(user_name, ''), coalesce(role_name, ''), granted_role_name FROM system.role_grants ORDER BY granted_role_name`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var user, role, granted string
		if err := rows.Scan(&user, &role, &granted); err != nil {
			rows.Close()
			return nil, err
		}
		id := user
		if user == "" {
			id = rolePrefix + role
		}
		members[id] = append(members[id], rolePrefix+granted)
	}
	rows.Close()
	var out []model.Role
	for _, kind := range []string{"", "role"} {
		table := map[string]string{"": "system.users", "role": "system.roles"}[kind]
		rows, err := db.QueryContext(ctx, `SELECT name, storage FROM `+table+` ORDER BY name`)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var name, storage string
			if err := rows.Scan(&name, &storage); err != nil {
				rows.Close()
				return nil, err
			}
			r := model.Role{ID: name, Name: name, Kind: kind, Flags: map[string]bool{"full": full[name]}, ConnLimit: -1,
				MemberOf: []string{}, System: storage == "users_xml", Current: kind == "" && name == current, Grantable: kind == "role"}
			if kind == "role" {
				r.ID = rolePrefix + name
			}
			r.MemberOf = append(r.MemberOf, members[r.ID]...)
			out = append(out, r)
		}
		rows.Close()
	}
	for i := range out {
		out[i].Grants, _ = stringList(ctx, db, "SHOW GRANTS FOR "+d.QuoteIdent(out[i].Name))
	}
	return out, nil
}

func stringList(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	out := []string{}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return out, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func stringSet(ctx context.Context, db *sql.DB, query string) (map[string]bool, error) {
	list, err := stringList(ctx, db, query)
	out := map[string]bool{}
	for _, s := range list {
		out[s] = true
	}
	return out, err
}

func (d Dialect) RoleID(from *model.Role, to model.RoleSpec) string {
	name := strings.TrimSpace(to.Name)
	if from != nil && isRole(*from) {
		return rolePrefix + name
	}
	return name
}

func (d Dialect) RoleSQL(from *model.Role, to model.RoleSpec, _ []model.Role, preview bool) ([]string, error) {
	name := strings.TrimSpace(to.Name)
	if name == "" {
		return nil, apperr.New("bad_name", "give the user a name")
	}
	role := from != nil && isRole(*from)
	if role && (to.Password != "" || to.ClearPassword) {
		return nil, apperr.New("bad_value", "a role has no password — give it to the users that are members")
	}
	if from == nil && to.Password == "" {
		return nil, apperr.New("bad_value", "give the new user a password")
	}
	entity := map[bool]string{true: "ROLE", false: "USER"}[role]
	id := d.QuoteIdent(name)
	password := ""
	switch {
	case to.ClearPassword:
		password = "NOT IDENTIFIED"
	case to.Password != "" && preview:
		password = "IDENTIFIED WITH sha256_password BY '********'"
	case to.Password != "":
		password = "IDENTIFIED WITH sha256_password BY " + d.QuoteString(to.Password)
	}
	var out []string
	before := model.Role{Flags: map[string]bool{}}
	if from == nil {
		out = append(out, "CREATE USER "+id+" "+password)
	} else {
		before = *from
		if from.Name != name {
			out = append(out, "ALTER "+entity+" "+d.QuoteIdent(from.Name)+" RENAME TO "+id)
		}
		if password != "" {
			out = append(out, "ALTER USER "+id+" "+password)
		}
	}
	if before.Flags["full"] != to.Flags["full"] {
		out = append(out, map[bool]string{true: "GRANT ALL ON *.* TO " + id, false: "REVOKE ALL ON *.* FROM " + id}[to.Flags["full"]])
	}
	for _, g := range to.MemberOf {
		if !slices.Contains(before.MemberOf, g) {
			out = append(out, "GRANT "+d.QuoteIdent(strings.TrimPrefix(g, rolePrefix))+" TO "+id)
		}
	}
	for _, g := range before.MemberOf {
		if !slices.Contains(to.MemberOf, g) {
			out = append(out, "REVOKE "+d.QuoteIdent(strings.TrimPrefix(g, rolePrefix))+" FROM "+id)
		}
	}
	return out, nil
}

func (d Dialect) DropRoleSQL(r model.Role) string {
	if isRole(r) {
		return "DROP ROLE " + d.QuoteIdent(r.Name)
	}
	return "DROP USER " + d.QuoteIdent(r.Name)
}
