package sqlc

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// minimalDB implements the minimum required for testing
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

func TestNewQueries(t *testing.T) {
	q := New(minimalDB{})
	if q == nil {
		t.Fatal("expected queries to be non-nil")
	}
}
