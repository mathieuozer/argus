package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTenant acquires a connection and sets the tenant context for RLS.
// The caller must release the connection when done.
func WithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string) (*pgxpool.Conn, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire connection: %w", err)
	}

	_, err = conn.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID)
	if err != nil {
		conn.Release()
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	return conn, nil
}

// WithTenantTx starts a transaction with the tenant context set for RLS.
func WithTenantTx(ctx context.Context, pool *pgxpool.Pool, tenantID string) (pgx.Tx, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}

	_, err = tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, fmt.Errorf("set tenant context: %w", err)
	}

	return tx, nil
}

// ExecWithTenant executes a single query within a tenant context.
func ExecWithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string, sql string, args ...any) error {
	tx, err := WithTenantTx(ctx, pool, tenantID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("exec: %w", err)
	}

	return tx.Commit(ctx)
}

// QueryWithTenant executes a query within a tenant context and returns rows.
func QueryWithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string, sql string, args ...any) (pgx.Rows, func(), error) {
	tx, err := WithTenantTx(ctx, pool, tenantID)
	if err != nil {
		return nil, nil, err
	}

	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		_ = tx.Rollback(ctx)
		return nil, nil, fmt.Errorf("query: %w", err)
	}

	cleanup := func() {
		rows.Close()
		_ = tx.Commit(ctx)
	}

	return rows, cleanup, nil
}
