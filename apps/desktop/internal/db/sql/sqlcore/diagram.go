package sqlcore

import (
	"context"
	"errors"
	"sync"

	"github.com/relay-client/plugboard/apps/desktop/internal/model"
)

const diagramReads = 4

func (s *Session) Relations(ctx context.Context, schema string) ([]model.Relation, error) {
	return retry(ctx, s, func() ([]model.Relation, error) { return s.Dialect.ListRelations(ctx, s.DB, schema) })
}

func (s *Session) Indexes(ctx context.Context, schema, table string) ([]model.Index, error) {
	return retry(ctx, s, func() ([]model.Index, error) { return s.Dialect.ListIndexes(ctx, s.DB, schema, table) })
}

func (s *Session) Diagram(ctx context.Context, schema string) (model.Diagram, error) {
	tables, err := s.Tables(ctx, schema)
	if err != nil {
		return model.Diagram{}, err
	}
	relations, err := s.Relations(ctx, schema)
	if err != nil {
		return model.Diagram{}, err
	}
	out := model.Diagram{Schema: schema, Tables: make([]model.DiagramTable, len(tables)), Relations: relations}
	errs := make([]error, len(tables))
	slots := make(chan struct{}, diagramReads)
	var wg sync.WaitGroup
	for i, t := range tables {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			cols, err := s.Columns(ctx, schema, t.Name)
			out.Tables[i] = model.DiagramTable{Name: t.Name, Kind: t.Kind, Columns: cols}
			errs[i] = err
		}()
	}
	wg.Wait()
	return out, errors.Join(errs...)
}
