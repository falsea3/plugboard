package sqlcore

import (
	"context"
	"slices"

	"github.com/relay-client/plugboard/apps/desktop/internal/apperr"
	"github.com/relay-client/plugboard/apps/desktop/internal/db"
	"github.com/relay-client/plugboard/apps/desktop/internal/db/sql/dialect"
	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

var _ db.RoleManager = (*Session)(nil)

func (s *Session) roles() (dialect.Roles, error) {
	if r, ok := s.Dialect.(dialect.Roles); ok {
		return r, nil
	}
	return nil, db.ErrNotSupported
}

func (s *Session) Roles(ctx context.Context) (model.RoleList, error) {
	r, err := s.roles()
	if err != nil {
		return model.RoleList{}, err
	}
	list, err := retry(ctx, s, func() ([]model.Role, error) { return r.ListRoles(ctx, s.DB) })
	return model.RoleList{Form: r.RoleForm(), Roles: list}, err
}

func (s *Session) RoleSQL(ctx context.Context, from string, to model.RoleSpec) ([]string, error) {
	return s.roleSQL(ctx, from, to, true)
}

func (s *Session) find(ctx context.Context, r dialect.Roles, id string) (*model.Role, []model.Role, error) {
	all, err := retry(ctx, s, func() ([]model.Role, error) { return r.ListRoles(ctx, s.DB) })
	if err != nil || id == "" {
		return nil, all, err
	}
	i := slices.IndexFunc(all, func(r model.Role) bool { return r.ID == id })
	if i < 0 {
		return nil, all, apperr.New("row_gone", id+" no longer exists — refresh the list")
	}
	return &all[i], all, nil
}

func (s *Session) roleSQL(ctx context.Context, from string, to model.RoleSpec, preview bool) ([]string, error) {
	r, err := s.roles()
	if err != nil {
		return nil, err
	}
	current, all, err := s.find(ctx, r, from)
	if err != nil {
		return nil, err
	}
	for _, g := range to.MemberOf {
		if !slices.ContainsFunc(all, func(r model.Role) bool { return r.ID == g && r.Grantable }) {
			return nil, apperr.New("bad_name", "can't add membership in "+g+": no such role")
		}
	}
	return r.RoleSQL(current, to, all, preview)
}

func (s *Session) SaveRole(ctx context.Context, from string, to model.RoleSpec) (string, error) {
	if s.Conn.ReadOnly {
		return "", &db.ReadOnlyError{Reason: "role changes"}
	}
	r, err := s.roles()
	if err != nil {
		return "", err
	}
	current, _, err := s.find(ctx, r, from)
	if err != nil {
		return "", err
	}
	stmts, err := s.roleSQL(ctx, from, to, false)
	if err != nil {
		return "", err
	}
	return r.RoleID(current, to), s.runAll(ctx, stmts)
}

func (s *Session) DropRole(ctx context.Context, id string) error {
	r, err := s.roles()
	if err != nil {
		return err
	}
	if s.Conn.ReadOnly {
		return &db.ReadOnlyError{Reason: "role changes"}
	}
	if id == "" {
		return apperr.New("bad_name", "choose a role to drop")
	}
	role, _, err := s.find(ctx, r, id)
	if err != nil {
		return err
	}
	return s.runAll(ctx, []string{r.DropRoleSQL(*role)})
}

func (s *Session) runAll(ctx context.Context, stmts []string) error {
	if len(stmts) == 0 {
		return nil
	}
	conn, err := s.DB.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if !s.Dialect.TransactionalDDL() {
		for _, stmt := range stmts {
			if _, err := conn.ExecContext(ctx, stmt); err != nil {
				return err
			}
		}
		return nil
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return tx.Commit()
}
