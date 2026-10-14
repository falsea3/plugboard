package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/apperr"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

var _ dialect.Roles = Dialect{}

const passwordMask = "'********'"

func (Dialect) ListRoles(ctx context.Context, db *sql.DB) ([]model.Role, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT r.rolname, r.rolcanlogin, r.rolsuper, r.rolcreatedb, r.rolcreaterole, r.rolinherit, r.rolreplication, r.rolbypassrls, r.rolconnlimit,
			CASE WHEN isfinite(r.rolvaliduntil) THEN r.rolvaliduntil::text ELSE '' END,
			COALESCE((SELECT json_agg(b.rolname ORDER BY b.rolname) FROM pg_auth_members m JOIN pg_roles b ON b.oid = m.roleid WHERE m.member = r.oid), '[]')::text,
			r.rolname LIKE 'pg\_%', r.rolname = current_user
		FROM pg_roles r
		ORDER BY r.rolname LIKE 'pg\_%', r.rolname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Role{}
	for rows.Next() {
		var r model.Role
		var memberOf string
		var login, super, createDB, createRole, inherit, replication, bypassRLS bool
		if err := rows.Scan(&r.Name, &login, &super, &createDB, &createRole, &inherit, &replication, &bypassRLS, &r.ConnLimit,
			&r.ValidUntil, &memberOf, &r.System, &r.Current); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(memberOf), &r.MemberOf); err != nil {
			return nil, err
		}
		r.ID, r.Grantable, r.Grants = r.Name, true, []string{}
		r.Flags = map[string]bool{"login": login, "superuser": super, "createDB": createDB, "createRole": createRole, "inherit": inherit, "replication": replication, "bypassRLS": bypassRLS}
		out = append(out, r)
	}
	return out, rows.Err()
}

var roleFlags = []model.RoleFlag{
	{Key: "login", Label: "Can log in", Hint: "A user rather than a group", Badge: "login"},
	{Key: "superuser", Label: "Superuser", Hint: "Bypasses every permission check", Badge: "superuser"},
	{Key: "createDB", Label: "Create databases"},
	{Key: "createRole", Label: "Create roles", Hint: "Create, change and drop other roles"},
	{Key: "inherit", Label: "Inherit privileges", Hint: "Use the privileges of the roles it is a member of"},
	{Key: "replication", Label: "Replication", Hint: "Connect for streaming replication and backups", Badge: "replication"},
	{Key: "bypassRLS", Label: "Bypass row-level security"},
}

var flagWords = map[string]string{
	"superuser": "SUPERUSER", "createDB": "CREATEDB", "createRole": "CREATEROLE", "inherit": "INHERIT",
	"login": "LOGIN", "replication": "REPLICATION", "bypassRLS": "BYPASSRLS",
}

func (Dialect) RoleForm() model.RoleForm {
	return model.RoleForm{
		Noun:       "role",
		Flags:      roleFlags,
		ValidUntil: true,
		ConnLimit:  true,
		Membership: true,
		New:        model.Role{Flags: map[string]bool{"login": true, "inherit": true}, ConnLimit: -1, MemberOf: []string{}, Grants: []string{}},
	}
}

func (d Dialect) RoleSQL(from *model.Role, to model.RoleSpec, _ []model.Role, preview bool) ([]string, error) {
	to.Name = strings.TrimSpace(to.Name)
	to.ValidUntil = strings.TrimSpace(to.ValidUntil)
	if to.Name == "" {
		return nil, apperr.New("bad_name", "give the role a name")
	}
	if to.ConnLimit < -1 {
		return nil, apperr.New("bad_value", "the connection limit is a number of connections, or -1 for no limit")
	}
	password, err := d.passwordClause(to, preview)
	if err != nil {
		return nil, err
	}
	name := d.QuoteIdent(to.Name)
	var out []string
	base := model.Role{Flags: map[string]bool{"inherit": true}, ConnLimit: -1}
	if from == nil {
		out = append(out, strings.TrimSuffix("CREATE ROLE "+name+" "+withClause(attributes(&base, to.Role), password), " "))
	} else {
		base = *from
		if from.Name != to.Name {
			out = append(out, "ALTER ROLE "+d.QuoteIdent(from.Name)+" RENAME TO "+name)
		}
		if with := withClause(attributes(from, to.Role), password); with != "" {
			out = append(out, "ALTER ROLE "+name+" "+with)
		}
	}
	for _, g := range to.MemberOf {
		if !slices.Contains(base.MemberOf, g) {
			out = append(out, "GRANT "+d.QuoteIdent(g)+" TO "+name)
		}
	}
	for _, g := range base.MemberOf {
		if !slices.Contains(to.MemberOf, g) {
			out = append(out, "REVOKE "+d.QuoteIdent(g)+" FROM "+name)
		}
	}
	return out, nil
}

func (Dialect) RoleID(_ *model.Role, to model.RoleSpec) string {
	return strings.TrimSpace(to.Name)
}

func (d Dialect) DropRoleSQL(r model.Role) string {
	return "DROP ROLE " + d.QuoteIdent(r.Name)
}

func (Dialect) passwordClause(to model.RoleSpec, preview bool) (string, error) {
	switch {
	case to.ClearPassword:
		return "PASSWORD NULL", nil
	case to.Password == "":
		return "", nil
	case preview:
		return "PASSWORD " + passwordMask, nil
	}
	verifier, err := scramVerifier(to.Password)
	if err != nil {
		return "", err
	}
	return "PASSWORD '" + verifier + "'", nil
}

func attributes(from *model.Role, to model.Role) []string {
	var out []string
	for _, f := range roleFlags {
		was, is := from.Flags[f.Key], to.Flags[f.Key]
		if was != is {
			word := flagWords[f.Key]
			if !is {
				word = "NO" + word
			}
			out = append(out, word)
		}
	}
	if from.ConnLimit != to.ConnLimit {
		out = append(out, "CONNECTION LIMIT "+strconv.Itoa(to.ConnLimit))
	}
	if from.ValidUntil != to.ValidUntil {
		until := to.ValidUntil
		if until == "" {
			until = "infinity"
		}
		out = append(out, "VALID UNTIL "+literal(until))
	}
	return out
}

func withClause(attrs []string, password string) string {
	if password != "" {
		attrs = append(attrs, password)
	}
	if len(attrs) == 0 {
		return ""
	}
	return "WITH " + strings.Join(attrs, " ")
}

func literal(s string) string {
	quoted := "'" + strings.ReplaceAll(s, "'", "''") + "'"
	if strings.Contains(s, `\`) {
		return "E" + strings.ReplaceAll(quoted, `\`, `\\`)
	}
	return quoted
}
