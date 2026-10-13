package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
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
		if err := rows.Scan(&r.Name, &r.Login, &r.Superuser, &r.CreateDB, &r.CreateRole, &r.Inherit, &r.Replication, &r.BypassRLS, &r.ConnLimit,
			&r.ValidUntil, &memberOf, &r.System, &r.Current); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(memberOf), &r.MemberOf); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (d Dialect) RoleSQL(from *model.Role, to model.RoleSpec, preview bool) ([]string, error) {
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
	base := model.Role{Inherit: true, ConnLimit: -1}
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

func (d Dialect) DropRoleSQL(name string) string {
	return "DROP ROLE " + d.QuoteIdent(name)
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
	flag := func(was, is bool, word string) {
		if was != is {
			if !is {
				word = "NO" + word
			}
			out = append(out, word)
		}
	}
	flag(from.Superuser, to.Superuser, "SUPERUSER")
	flag(from.CreateDB, to.CreateDB, "CREATEDB")
	flag(from.CreateRole, to.CreateRole, "CREATEROLE")
	flag(from.Inherit, to.Inherit, "INHERIT")
	flag(from.Login, to.Login, "LOGIN")
	flag(from.Replication, to.Replication, "REPLICATION")
	flag(from.BypassRLS, to.BypassRLS, "BYPASSRLS")
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
