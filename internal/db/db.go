// Package db defines database interfaces and SQL connection helpers.
package db

import (
	"context"
	"database/sql"
	"time"
)

type RowScanner interface{ Scan(dest ...any) error }
type Queryer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type Transactor interface {
	WithinTx(context.Context, func(context.Context, Queryer) error) error
}

type SQLStore struct{ DB *sql.DB }

func Open(driver, dsn string, maxOpen, maxIdle int, maxLifetime time.Duration) (*SQLStore, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(maxLifetime)
	return &SQLStore{DB: db}, nil
}
func (s *SQLStore) Ping(ctx context.Context) error { return s.DB.PingContext(ctx) }
func (s *SQLStore) Close() error                   { return s.DB.Close() }
func (s *SQLStore) WithinTx(ctx context.Context, fn func(context.Context, Queryer) error) error {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(ctx, tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
