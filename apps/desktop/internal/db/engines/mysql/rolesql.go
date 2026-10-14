package mysql

import (
	"slices"
	"strconv"
	"strings"

	"github.com/falsea3/plugboard/apps/desktop/internal/apperr"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

const passwordMask = "'********'"

func (d Dialect) RoleSQL(from *model.Role, to model.RoleSpec, _ []model.Role, preview bool) ([]string, error) {
	to.Name = strings.TrimSpace(to.Name)
	to.Host = strings.TrimSpace(to.Host)
	if to.Name == "" {
		return nil, apperr.New("bad_name", "give the user a name")
	}
	if to.ConnLimit < -1 {
		return nil, apperr.New("bad_value", "the connection limit is a number of connections, or -1 for no limit")
	}
	role := from != nil && isRole(*from)
	id := d.RoleID(from, to)
	var out []string
	before := model.Role{Flags: map[string]bool{}, ConnLimit: -1}
	if from == nil {
		out = append(out, strings.TrimRight("CREATE USER "+id+" "+d.accountOptions(&before, to, preview), " "))
	} else {
		before = *from
		if from.Name != to.Name || (!role && from.Host != to.Host) {
			out = append(out, "RENAME USER "+from.ID+" TO "+id)
		}
		if opts := d.accountOptions(from, to, preview); opts != "" && !role {
			out = append(out, "ALTER USER "+id+" "+opts)
		}
	}
	out = append(out, privileges(before, to.Role, id)...)
	return append(out, d.membership(before.MemberOf, to.MemberOf, id)...), nil
}

func (d Dialect) accountOptions(from *model.Role, to model.RoleSpec, preview bool) string {
	var parts []string
	switch {
	case to.ClearPassword:
		parts = append(parts, "IDENTIFIED BY ''")
	case to.Password != "" && preview:
		parts = append(parts, "IDENTIFIED BY "+passwordMask)
	case to.Password != "":
		parts = append(parts, "IDENTIFIED BY "+d.QuoteString(to.Password))
	}
	if from.ConnLimit != to.ConnLimit {
		parts = append(parts, "WITH MAX_USER_CONNECTIONS "+strconv.Itoa(max(to.ConnLimit, 0)))
	}
	if from.Flags["locked"] != to.Flags["locked"] {
		parts = append(parts, map[bool]string{true: "ACCOUNT LOCK", false: "ACCOUNT UNLOCK"}[to.Flags["locked"]])
	}
	return strings.Join(parts, " ")
}

func privileges(from, to model.Role, id string) []string {
	var out []string
	wasFull, isFull := from.Flags["full"], to.Flags["full"]
	if wasFull && isFull && slices.ContainsFunc(privFlags, func(f struct{ key, word, column string }) bool { return from.Flags[f.key] && !to.Flags[f.key] }) {
		isFull = false
	}
	var grant, revoke []string
	switch {
	case !wasFull && isFull:
		out = append(out, "GRANT ALL PRIVILEGES ON *.* TO "+id)
	case wasFull && !isFull:
		out = append(out, "REVOKE ALL PRIVILEGES ON *.* FROM "+id)
		for _, f := range privFlags {
			if to.Flags[f.key] {
				grant = append(grant, f.word)
			}
		}
	case !isFull:
		for _, f := range privFlags {
			if !from.Flags[f.key] && to.Flags[f.key] {
				grant = append(grant, f.word)
			} else if from.Flags[f.key] && !to.Flags[f.key] {
				revoke = append(revoke, f.word)
			}
		}
	}
	if len(grant) > 0 {
		out = append(out, "GRANT "+strings.Join(grant, ", ")+" ON *.* TO "+id)
	}
	if len(revoke) > 0 {
		out = append(out, "REVOKE "+strings.Join(revoke, ", ")+" ON *.* FROM "+id)
	}
	if !from.Flags["grant"] && to.Flags["grant"] {
		out = append(out, "GRANT USAGE ON *.* TO "+id+" WITH GRANT OPTION")
	} else if from.Flags["grant"] && !to.Flags["grant"] {
		out = append(out, "REVOKE GRANT OPTION ON *.* FROM "+id)
	}
	return out
}

func (d Dialect) membership(from, to []string, id string) []string {
	var out []string
	for _, r := range to {
		if !slices.Contains(from, r) {
			out = append(out, "GRANT "+r+" TO "+id)
		}
	}
	for _, r := range from {
		if !slices.Contains(to, r) {
			out = append(out, "REVOKE "+r+" FROM "+id)
		}
	}
	if len(out) == 0 {
		return nil
	}
	switch {
	case !d.mariaDB && len(to) > 0:
		out = append(out, "SET DEFAULT ROLE ALL TO "+id)
	case d.mariaDB && len(to) > 0:
		out = append(out, "SET DEFAULT ROLE "+to[0]+" FOR "+id)
	case d.mariaDB:
		out = append(out, "SET DEFAULT ROLE NONE FOR "+id)
	}
	return out
}

func (d Dialect) RoleID(from *model.Role, to model.RoleSpec) string {
	name, host := strings.TrimSpace(to.Name), strings.TrimSpace(to.Host)
	if from != nil && isRole(*from) {
		return d.account(name, "")
	}
	if host == "" {
		host = "%"
	}
	return d.account(name, host)
}

func (d Dialect) DropRoleSQL(r model.Role) string {
	if isRole(r) {
		return "DROP ROLE " + r.ID
	}
	return "DROP USER " + r.ID
}
