package dialect

import (
	"context"
	"database/sql"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

type ServerConfig interface {
	ConfigFiles(ctx context.Context, db *sql.DB) ([]model.ConfigFile, error)
	ReadServerFile(ctx context.Context, db *sql.DB, path string) (string, error)
	WriteServerFile(ctx context.Context, db *sql.DB, path, text string) error
	CheckConfig(ctx context.Context, db *sql.DB, path string) (problems []model.ConfigProblem, restart []string)
	ReloadConfig(ctx context.Context, db *sql.DB) error
}

type Roles interface {
	RoleForm() model.RoleForm
	ListRoles(ctx context.Context, db *sql.DB) ([]model.Role, error)
	RoleSQL(from *model.Role, to model.RoleSpec, all []model.Role, preview bool) ([]string, error)
	DropRoleSQL(r model.Role) string
	RoleID(from *model.Role, to model.RoleSpec) string
}
