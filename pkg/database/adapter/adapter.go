// pkg/database/seeder/adapter.go
package adapter

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PgxAdapter struct {
	pool *pgxpool.Pool
}

func NewPgxAdapter(pool *pgxpool.Pool) Pool {
	return &PgxAdapter{pool: pool}
}

func (a *PgxAdapter) Begin(ctx context.Context) (Tx, error) {
	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &TxAdapter{tx: tx}, nil
}

func (a *PgxAdapter) Query(ctx context.Context, sql string, args ...interface{}) (Rows, error) {
	return a.pool.Query(ctx, sql, args...)
}

type TxAdapter struct {
	tx pgx.Tx
}

func (t *TxAdapter) Exec(ctx context.Context, sql string, args ...interface{}) error {
	_, err := t.tx.Exec(ctx, sql, args...)
	return err
}

func (t *TxAdapter) QueryRow(ctx context.Context, sql string, args ...interface{}) Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

func (t *TxAdapter) Commit(ctx context.Context) error {
	return t.tx.Commit(ctx)
}

func (t *TxAdapter) Rollback(ctx context.Context) error {
	return t.tx.Rollback(ctx)
}
