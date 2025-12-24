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

// minimal stub that only implements what's needed for tests
type minimalDB struct{}

func (minimalDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (minimalDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (minimalDB) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return nil
}

func TestStoreQuerier(t *testing.T) {
	q := sqlc.New(minimalDB{})
	store := &Store{Queries: q}
	if store.Querier() == nil {
		t.Fatal("expected querier to be non-nil")
	}
}

func TestStoreWithTx(t *testing.T) {
	q := sqlc.New(minimalDB{})
	store := &Store{Queries: q}

	minimalTx := &minimalTxStub{}
	result := store.WithTx(minimalTx)
	if result == nil {
		t.Fatal("expected WithTx to return queries")
	}
}

// minimal transaction stub that only implements what's needed for tests
type minimalTxStub struct{}

func (minimalTxStub) Begin(context.Context) (pgx.Tx, error) { return nil, nil }
func (minimalTxStub) Commit(context.Context) error          { return nil }
func (minimalTxStub) Rollback(context.Context) error        { return nil }
func (minimalTxStub) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (minimalTxStub) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults { return nil }
func (minimalTxStub) LargeObjects() pgx.LargeObjects                          { return pgx.LargeObjects{} }
func (minimalTxStub) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (minimalTxStub) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (minimalTxStub) Query(context.Context, string, ...any) (pgx.Rows, error) { return nil, nil }
func (minimalTxStub) QueryRow(context.Context, string, ...any) pgx.Row        { return nil }
func (minimalTxStub) Conn() *pgx.Conn                                         { return nil }
