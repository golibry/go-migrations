//go:build sqlite

package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/golibry/go-migrations/execution"
	_ "modernc.org/sqlite"
)

// SqliteHandler Repository implementation for SQLite integration.
type SqliteHandler struct {
	db        *sql.DB
	tableName string
	ctx       context.Context
}

// NewSqliteHandler builds a new SqliteHandler. If db is nil, it will try to build a db handle
// from the provided dsn. It is recommended to share the same *sql.DB handle between
// your application and this handler to efficiently manage connection pools.
func NewSqliteHandler(
	dsn string,
	tableName string,
	ctx context.Context,
	db *sql.DB,
) (*SqliteHandler, error) {
	if db == nil {
		var err error
		db, err = newDbHandle(dsn, "sqlite")

		if err != nil {
			return nil, err
		}
	}

	quotedTableName, err := quoteSqliteTableName(tableName)
	if err != nil {
		return nil, err
	}

	return &SqliteHandler{
		db:        db,
		tableName: quotedTableName,
		ctx:       ctx,
	}, nil
}

func quoteSqliteTableName(tableName string) (string, error) {
	return quoteSQLTableName(tableName, func(identifier string) string {
		return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
	})
}

func (h *SqliteHandler) Context() context.Context {
	return h.ctx
}

func (h *SqliteHandler) Init() error {
	_, err := h.db.ExecContext(
		h.ctx,
		"CREATE TABLE IF NOT EXISTS "+h.tableName+" ("+
			"version INTEGER NOT NULL,"+
			"executed_at_ms INTEGER NOT NULL,"+
			"finished_at_ms INTEGER NOT NULL,"+
			"PRIMARY KEY (version)"+
			")",
	)
	return err
}

func (h *SqliteHandler) LoadExecutions() (executions []execution.MigrationExecution, err error) {
	rows, err := h.db.QueryContext(
		h.ctx,
		"SELECT version, executed_at_ms, finished_at_ms FROM "+h.tableName,
	)

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

func (h *SqliteHandler) Save(execution execution.MigrationExecution) error {
	_, err := h.db.ExecContext(
		h.ctx,
		"INSERT INTO "+h.tableName+" (version, executed_at_ms, finished_at_ms) "+
			"VALUES (?, ?, ?) "+
			"ON CONFLICT(version) DO UPDATE SET "+
			"executed_at_ms = excluded.executed_at_ms, "+
			"finished_at_ms = excluded.finished_at_ms",
		execution.Version, execution.ExecutedAtMs, execution.FinishedAtMs,
	)
	return err
}

func (h *SqliteHandler) Remove(execution execution.MigrationExecution) error {
	_, err := h.db.ExecContext(
		h.ctx,
		"DELETE FROM "+h.tableName+" WHERE version = ?",
		execution.Version,
	)
	return err
}

func (h *SqliteHandler) FindOne(version uint64) (*execution.MigrationExecution, error) {
	row := h.db.QueryRowContext(
		h.ctx,
		"SELECT version, executed_at_ms, finished_at_ms FROM "+h.tableName+" WHERE version = ?",
		version,
	)

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
