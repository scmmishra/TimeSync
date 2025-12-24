package sqlc

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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

func TestNewQueries(t *testing.T) {
	q := New(stubDB{})
	if q == nil {
		t.Fatal("expected queries to be non-nil")
	}
}
