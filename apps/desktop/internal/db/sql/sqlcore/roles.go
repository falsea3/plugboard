package sqlcore

import (
	"context"
	"strings"

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

func (s *Session) Roles(ctx context.Context) ([]model.Role, error) {
	r, err := s.roles()
	if err != nil {
		return nil, err
	}
	return retry(ctx, s, func() ([]model.Role, error) { return r.ListRoles(ctx, s.DB) })
}

func (s *Session) RoleSQL(ctx context.Context, from string, to model.RoleSpec) ([]string, error) {
	return s.roleSQL(ctx, from, to, true)
}

func (s *Session) roleSQL(ctx context.Context, from string, to model.RoleSpec, preview bool) ([]string, error) {
	r, err := s.roles()
	if err != nil {
		return nil, err
	}
	var current *model.Role
	if from != "" {
		all, err := s.Roles(ctx)
		if err != nil {
			return nil, err
		}
		for i := range all {
			if all[i].Name == from {
				current = &all[i]
			}
		}
		if current == nil {
			return nil, apperr.New("row_gone", "role "+from+" no longer exists — refresh the list")
		}
	}
	return r.RoleSQL(current, to, preview)
}

func (s *Session) SaveRole(ctx context.Context, from string, to model.RoleSpec) error {
	if s.Conn.ReadOnly {
		return &db.ReadOnlyError{Reason: "role changes"}
	}
	stmts, err := s.roleSQL(ctx, from, to, false)
	if err != nil {
		return err
	}
	return s.runAll(ctx, stmts)
}

func (s *Session) DropRole(ctx context.Context, name string) error {
	r, err := s.roles()
	if err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return apperr.New("bad_name", "choose a role to drop")
	}
	if s.Conn.ReadOnly {
		return &db.ReadOnlyError{Reason: "role changes"}
	}
	return s.runAll(ctx, []string{r.DropRoleSQL(name)})
}

func (s *Session) runAll(ctx context.Context, stmts []string) error {
	if len(stmts) == 0 {
		return nil
	}
	tx, err := s.DB.BeginTx(ctx, nil)
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
