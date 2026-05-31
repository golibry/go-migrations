//go:build postgres

package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/golibry/go-migrations/execution"
	_ "github.com/lib/pq"
)

// PostgresHandler Repository implementation for PostgresSQL integration
type PostgresHandler struct {
	db        *sql.DB
	tableName string
	lockName  string
	ctx       context.Context
}

// NewPostgresHandler Builds a new PostgresHandler. If db is nil, it will try to build a db handle
// from the provided dsn. It is recommended to share the same *sql.DB handle between
// your application and this handler to efficiently manage connection pools.
func NewPostgresHandler(
	dsn string,
	tableName string,
	ctx context.Context,
	db *sql.DB,
) (*PostgresHandler, error) {
	if db == nil {
		var err error
		db, err = newDbHandle(dsn, "postgres")

		if err != nil {
			return nil, err
		}
	}

	quotedTableName, err := quotePostgresTableName(tableName)
	if err != nil {
		return nil, err
	}

	return &PostgresHandler{
		db:        db,
		tableName: quotedTableName,
		lockName:  "go-migrations:" + tableName,
		ctx:       ctx,
	}, nil
}

func quotePostgresTableName(tableName string) (string, error) {
	return quoteSQLTableName(tableName, func(identifier string) string {
		return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
	})
}

func (h *PostgresHandler) Context() context.Context {
	return h.ctx
}

func (h *PostgresHandler) Init() error {
	query := `
		CREATE TABLE IF NOT EXISTS ` + h.tableName + ` (
			version BIGINT NOT NULL,
			executed_at_ms BIGINT NOT NULL,
			finished_at_ms BIGINT NOT NULL,
			PRIMARY KEY (version)
		)
		`

	_, err := h.db.ExecContext(h.ctx, query)
	return err
}

func (h *PostgresHandler) LoadExecutions() (executions []execution.MigrationExecution, err error) {
	query := `SELECT version, executed_at_ms, finished_at_ms FROM ` + h.tableName
	rows, err := h.db.QueryContext(h.ctx, query)

	if err != nil {
		return executions, err
	}

	defer func(rows *sql.Rows) {
		if closeErr := rows.Close(); closeErr != nil && err != nil {
			err = errors.Join(err, closeErr)
		}
	}(rows)

	for rows.Next() {
		var exec execution.MigrationExecution
		if err = rows.Scan(&exec.Version, &exec.ExecutedAtMs, &exec.FinishedAtMs); err != nil {
			return executions, err
		}
		executions = append(executions, exec)
	}

	err = rows.Err()
	return executions, err
}

func (h *PostgresHandler) Save(execution execution.MigrationExecution) error {
	// PostgresSQL uses ON CONFLICT for upsert operations
	query := `
		INSERT INTO ` + h.tableName + ` (version, executed_at_ms, finished_at_ms) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (version) DO UPDATE SET 
		executed_at_ms = $2, 
		finished_at_ms = $3
		`

	_, err := h.db.ExecContext(
		h.ctx,
		query,
		execution.Version, execution.ExecutedAtMs, execution.FinishedAtMs,
	)
	return err
}

func (h *PostgresHandler) Remove(execution execution.MigrationExecution) error {
	query := `DELETE FROM ` + h.tableName + ` WHERE version = $1`
	_, err := h.db.ExecContext(h.ctx, query, execution.Version)
	return err
}

func (h *PostgresHandler) FindOne(version uint64) (*execution.MigrationExecution, error) {
	query := `SELECT version, executed_at_ms, finished_at_ms FROM ` + h.tableName + ` WHERE version = $1`
	row := h.db.QueryRowContext(h.ctx, query, version)

	if row == nil {
		return nil, nil
	}

	var exec execution.MigrationExecution
	err := row.Scan(&exec.Version, &exec.ExecutedAtMs, &exec.FinishedAtMs)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return &exec, row.Err()
}

func (h *PostgresHandler) Lock() (func() error, error) {
	conn, err := h.db.Conn(h.ctx)
	if err != nil {
		return nil, err
	}

	lockID := advisoryLockID(h.lockName)
	if _, err = conn.ExecContext(h.ctx, "SELECT pg_advisory_lock($1)", lockID); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return func() error {
		_, releaseErr := conn.ExecContext(h.ctx, "SELECT pg_advisory_unlock($1)", lockID)
		closeErr := conn.Close()
		return errors.Join(releaseErr, closeErr)
	}, nil
}
