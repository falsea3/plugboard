package api

import (
	"github.com/falsea3/plugboard/apps/desktop/internal/db"
	"github.com/falsea3/plugboard/apps/desktop/internal/model"
)

func (a *App) ReadCell(sessionID string, q model.CellQuery) (string, error) {
	r, err := sessionAs[db.CellReader](a, sessionID)
	if err != nil {
		return "", err
	}
	ctx, cancel := a.catalogCtx()
	defer cancel()
	return r.ReadCell(ctx, q)
}
