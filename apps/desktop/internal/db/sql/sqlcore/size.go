package sqlcore

import (
	"context"

	"github.com/falsea3/plugboard/apps/desktop/internal/db"
	"github.com/falsea3/plugboard/apps/desktop/internal/db/sql/dialect"
)

func (s *Session) DatabaseSize(ctx context.Context) (int64, error) {
	q, ok := s.Dialect.(dialect.SizeQuery)
	if !ok {
		return 0, db.ErrNotSupported
	}
	return retry(ctx, s, func() (int64, error) {
		var size int64
		err := s.DB.QueryRowContext(ctx, q.SizeQuery()).Scan(&size)
		return size, err
	})
}
