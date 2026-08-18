package main

import (
	"context"

	"github.com/RamDass1/test-api/internal/service"
	"github.com/RamDass1/test-api/internal/store"
)

type sqlStore struct{ *store.Store }

func newServiceDB(s *store.Store) service.DB { return sqlStore{s} }

func (d sqlStore) InTx(ctx context.Context, fn func(service.DB) error) error {
	return d.Store.InTx(ctx, func(tx *store.Store) error {
		return fn(sqlStore{tx})
	})
}
