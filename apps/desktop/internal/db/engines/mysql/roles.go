package mysql

import (
	"context"
	"database/sql"
	"regexp"
	"strconv"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

var _ dialect.Roles = Dialect{}

var privFlags = []struct{ key, word, column string }{
	{"createUser", "CREATE USER", "Create_user_priv"},
	{"process", "PROCESS", "Process_priv"},
	{"reload", "RELOAD", "Reload_priv"},
	{"replication", "REPLICATION SLAVE", "Repl_slave_priv"},
}

func (Dialect) RoleForm() model.RoleForm {
	return model.RoleForm{
		Noun: "user",
		Flags: []model.RoleFlag{
			{Key: "full", Label: "All privileges", Hint: "Everything on every database — ALL PRIVILEGES ON *.*", Badge: "all privileges"},
			{Key: "grant", Label: "Grant option", Hint: "Pass its privileges on to others"},
			{Key: "createUser", Label: "Create users", Hint: "Create, change and drop accounts"},
			{Key: "process", Label: "See all sessions", Hint: "SHOW PROCESSLIST for everyone, and KILL"},
			{Key: "reload", Label: "Reload and flush", Hint: "FLUSH logs, tables and privileges"},
			{Key: "replication", Label: "Replication", Hint: "Read the binary log as a replica"},
			{Key: "locked", Label: "Account locked", Hint: "Can't log in", Badge: "locked"},
		},
		Host:       true,
		ConnLimit:  true,
		Membership: true,
		New:        model.Role{Host: "%", Flags: map[string]bool{}, ConnLimit: -1, MemberOf: []string{}, Grants: []string{}},
	}
}

func (d Dialect) account(name, host string) string {
	if host == "" && d.mariaDB {
		return d.QuoteIdent(name)
	}
	return d.QuoteIdent(name) + "@" + d.QuoteIdent(host)
}

func isRole(r model.Role) bool { return !strings.Contains(r.ID, "@") }

func (d Dialect) ListRoles(ctx context.Context, db *sql.DB) ([]model.Role, error) {
	var current string
	if err := db.QueryRowContext(ctx, `SELECT CURRENT_USER()`).Scan(&current); err != nil {
		return nil, err
	}
	accounts, err := d.accounts(ctx, db)
	if err != nil {
		return nil, err
	}
	locked := d.lockedAccounts(ctx, db, accounts)
	members := d.memberships(ctx, db)
	out := make([]model.Role, 0, len(accounts))
	for _, a := range accounts {
		r := model.Role{Name: a.user, Host: a.host, ConnLimit: a.maxConn, MemberOf: []string{}, Grantable: true}
		if a.role {
			r.ID = d.account(a.user, "")
			r.Host = ""
		} else {
			r.ID = d.account(a.user, a.host)
		}
		if r.ConnLimit == 0 {
			r.ConnLimit = -1
		}
		r.Flags = map[string]bool{"full": a.full, "grant": a.priv["Grant_priv"], "locked": locked[r.ID]}
		for _, f := range privFlags {
			r.Flags[f.key] = a.priv[f.column]
		}
		r.System = strings.HasPrefix(a.user, "mysql.") || a.user == "mariadb.sys"
		r.Current = current == a.user+"@"+a.host
		r.Grantable = !d.mariaDB || a.role
		r.MemberOf = append(r.MemberOf, members[r.ID]...)
		r.Grants = d.grants(ctx, db, r.ID)
		out = append(out, r)
	}
	return out, nil
}

type account struct {
	user, host string
	role, full bool
	maxConn    int
	priv       map[string]bool
}

func (d Dialect) accounts(ctx context.Context, db *sql.DB) ([]account, error) {
	rows, err := db.QueryContext(ctx, `SELECT * FROM mysql.user ORDER BY User, Host`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []account
	for rows.Next() {
		cells := make([]sql.RawBytes, len(names))
		ptrs := make([]any, len(names))
		for i := range cells {
			ptrs[i] = &cells[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		a := account{priv: map[string]bool{}, full: true}
		for i, name := range names {
			v := string(cells[i])
			switch {
			case strings.EqualFold(name, "User"):
				a.user = v
			case strings.EqualFold(name, "Host"):
				a.host = v
			case strings.EqualFold(name, "is_role"):
				a.role = v == "Y"
			case strings.EqualFold(name, "max_user_connections"):
				a.maxConn, _ = strconv.Atoi(v)
			case strings.HasSuffix(name, "_priv"):
				a.priv[name] = v == "Y"
				if name != "Grant_priv" && v != "Y" {
					a.full = false
				}
			}
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (d Dialect) lockedAccounts(ctx context.Context, db *sql.DB, accounts []account) map[string]bool {
	query := `SELECT User, Host, account_locked = 'Y' FROM mysql.user`
	if d.mariaDB {
		query = `SELECT User, Host, COALESCE(JSON_VALUE(Priv, '$.account_locked'), 0) = 1 FROM mysql.global_priv`
	}
	out := map[string]bool{}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var user, host string
		var locked bool
		if rows.Scan(&user, &host, &locked) == nil {
			out[d.account(user, host)] = locked
		}
	}
	return out
}

func (d Dialect) memberships(ctx context.Context, db *sql.DB) map[string][]string {
	query := `SELECT TO_USER, TO_HOST, FROM_USER, FROM_HOST FROM mysql.role_edges ORDER BY FROM_USER, FROM_HOST`
	if d.mariaDB {
		query = `SELECT User, Host, Role, '' FROM mysql.roles_mapping ORDER BY Role`
	}
	out := map[string][]string{}
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var user, host, role, roleHost string
		if rows.Scan(&user, &host, &role, &roleHost) == nil {
			id := d.account(user, host)
			out[id] = append(out[id], d.account(role, roleHost))
		}
	}
	return out
}

var hiddenPassword = regexp.MustCompile(` IDENTIFIED (BY PASSWORD|VIA|WITH) .*$`)

func (d Dialect) grants(ctx context.Context, db *sql.DB, id string) []string {
	out := []string{}
	rows, err := db.QueryContext(ctx, "SHOW GRANTS FOR "+id)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var g string
		if rows.Scan(&g) == nil {
			out = append(out, hiddenPassword.ReplaceAllString(g, ""))
		}
	}
	return out
}
