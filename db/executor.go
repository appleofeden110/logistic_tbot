package db

import (
	"context"
	"database/sql"
)

type DBExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Prepare(query string) (*sql.Stmt, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

func VerifyWhichType(db DBExecutor) string {
	switch db.(type) {
	case *sql.DB:
		return "db"
	case *sql.Tx:
		return "tx"
	default:
		return "un"
	}
}
