package store

import (
	"context"
	"testing"

	"timesync/backend/internal/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestStoreCloseNilSafe(t *testing.T) {
	var s *Store
	s.Close()

	s = &Store{}
	s.Close()
}

func TestOpenInvalidURL(t *testing.T) {
	if _, err := Open(context.Background(), "not-a-url"); err == nil {
		t.Fatal("expected error for invalid database url")
	}
}

type stubDB struct{}

func (stubDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (stubDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (stubDB) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return nil
}

type stubTx struct{}

func (stubTx) Begin(context.Context) (pgx.Tx, error) { return stubTx{}, nil }
func (stubTx) Commit(context.Context) error          { return nil }
func (stubTx) Rollback(context.Context) error        { return nil }
func (stubTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (stubTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (stubTx) LargeObjects() pgx.LargeObjects                         { return pgx.LargeObjects{} }
func (stubTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (stubTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (stubTx) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (stubTx) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (stubTx) Conn() *pgx.Conn                                         { return nil }

func TestStoreQuerier(t *testing.T) {
	q := sqlc.New(stubDB{})
	store := &Store{Queries: q}
	if store.Querier() == nil {
		t.Fatal("expected querier to be non-nil")
	}
}

func TestStoreWithTx(t *testing.T) {
	q := sqlc.New(stubDB{})
	store := &Store{Queries: q}
	if store.WithTx(stubTx{}) == nil {
		t.Fatal("expected WithTx to return queries")
	}
}
