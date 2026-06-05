//go:build sqlite

package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/golibry/go-migrations/execution"
	"github.com/golibry/go-migrations/migration"
	"github.com/stretchr/testify/suite"
	_ "modernc.org/sqlite"
)

const SqliteExecutionsTable = "migration_executions"

type SqliteTestSuite struct {
	suite.Suite
	dsn     string
	db      *sql.DB
	handler *SqliteHandler
}

func TestSqliteTestSuite(t *testing.T) {
	suite.Run(t, new(SqliteTestSuite))
}

func (suite *SqliteTestSuite) SetupTest() {
	dbPath := filepath.Join(suite.T().TempDir(), "migrations.sqlite")
	suite.dsn = dbPath

	handler, err := NewSqliteHandler(suite.dsn, SqliteExecutionsTable, context.Background(), nil)
	suite.Require().NoError(err)
	suite.handler = handler
	suite.db = handler.db
	suite.Require().NoError(suite.handler.Init())
}

func (suite *SqliteTestSuite) TearDownTest() {
	_ = suite.db.Close()
}

func (suite *SqliteTestSuite) TestItCanBuildMigrationsExclusiveDbHandle() {
	handle, err := newDbHandle(suite.dsn, "sqlite")
	suite.Require().NoError(err)
	defer func() {
		_ = handle.Close()
	}()

	suite.Assert().Equal(1, handle.Stats().MaxOpenConnections)
	suite.Assert().NoError(handle.Ping())
}

func (suite *SqliteTestSuite) TestItCanBuildHandlerWithProvidedContext() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler, err := NewSqliteHandler(suite.dsn, "migration_execs", ctx, suite.db)
	suite.Assert().Nil(err)
	suite.Assert().Same(ctx, handler.Context())
}

func (suite *SqliteTestSuite) TestItCanInitializeExecutionsTable() {
	_, _ = suite.db.Exec(`DROP TABLE IF EXISTS "` + SqliteExecutionsTable + `"`)
	tableExists := func() bool {
		var table string
		_ = suite.db.QueryRow(
			`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`,
			SqliteExecutionsTable,
		).Scan(&table)
		return table == SqliteExecutionsTable
	}

	suite.Assert().False(tableExists())
	_ = suite.handler.Init()
	suite.Assert().True(tableExists())
}

func sqliteExecutionsProvider() map[uint64]execution.MigrationExecution {
	return map[uint64]execution.MigrationExecution{
		uint64(1): {Version: 1, ExecutedAtMs: 2, FinishedAtMs: 3},
		uint64(4): {Version: 4, ExecutedAtMs: 5, FinishedAtMs: 6},
		uint64(7): {Version: 7, ExecutedAtMs: 8, FinishedAtMs: 9},
	}
}

func (suite *SqliteTestSuite) TestItCanLoadExecutions() {
	executions := sqliteExecutionsProvider()

	for _, exec := range executions {
		_, _ = suite.db.Exec(
			`INSERT INTO "`+SqliteExecutionsTable+`" VALUES (?, ?, ?)`,
			exec.Version, exec.ExecutedAtMs, exec.FinishedAtMs,
		)
	}

	loadedExecs, err := suite.handler.LoadExecutions()

	suite.Assert().NoError(err)
	for _, exec := range loadedExecs {
		suite.Assert().Contains(executions, exec.Version)
		suite.Assert().Equal(executions[exec.Version], exec)
		delete(executions, exec.Version)
	}
	suite.Assert().Len(executions, 0)
}

func (suite *SqliteTestSuite) TestItFailsToExecuteAnyChangesWhenMissingTable() {
	_, _ = suite.db.Exec(`DROP TABLE IF EXISTS "` + SqliteExecutionsTable + `"`)
	migrationExecution := execution.StartExecution(migration.NewDummyMigration(123))
	_, errLoad := suite.handler.LoadExecutions()
	errSave := suite.handler.Save(*migrationExecution)
	errRemove := suite.handler.Remove(*migrationExecution)
	_, errFindOne := suite.handler.FindOne(uint64(123))

	suite.Assert().Error(errLoad)
	suite.Assert().ErrorContains(errLoad, SqliteExecutionsTable)
	suite.Assert().Error(errSave)
	suite.Assert().ErrorContains(errSave, SqliteExecutionsTable)
	suite.Assert().Error(errRemove)
	suite.Assert().ErrorContains(errRemove, SqliteExecutionsTable)
	suite.Assert().Error(errFindOne)
	suite.Assert().ErrorContains(errFindOne, SqliteExecutionsTable)
}

func (suite *SqliteTestSuite) TestItFailsToLoadExecutionsFromInvalidRepoData() {
	_, _ = suite.db.Exec(`DROP TABLE IF EXISTS "` + SqliteExecutionsTable + `"`)
	_, _ = suite.db.Exec(
		`CREATE TABLE "` + SqliteExecutionsTable + `" (
			version INTEGER NOT NULL,
			executed_at_ms INTEGER NOT NULL,
			finished_at_ms INTEGER,
			PRIMARY KEY (version)
		)`,
	)
	_, _ = suite.db.Exec(
		`INSERT INTO "` + SqliteExecutionsTable + `" VALUES (1, 2, 1), (3, 4, NULL)`,
	)
	execs, err := suite.handler.LoadExecutions()
	suite.Assert().Len(execs, 1)
	suite.Assert().Error(err)
	suite.Assert().ErrorContains(err, "Scan error")
}

func (suite *SqliteTestSuite) TestItCanSaveExecutions() {
	executions := sqliteExecutionsProvider()

	for _, exec := range executions {
		err := suite.handler.Save(exec)
		suite.Assert().NoError(err)
	}

	savedExecs, _ := suite.handler.LoadExecutions()
	for _, exec := range savedExecs {
		suite.Assert().Contains(executions, exec.Version)
		suite.Assert().Equal(executions[exec.Version], exec)
	}

	for i, exec := range executions {
		exec.FinishedAtMs++
		exec.ExecutedAtMs++
		executions[i] = exec
		err := suite.handler.Save(executions[i])
		suite.Assert().NoError(err)
	}

	savedExecs, _ = suite.handler.LoadExecutions()
	for _, exec := range savedExecs {
		suite.Assert().Contains(executions, exec.Version)
		suite.Assert().Equal(executions[exec.Version], exec)
	}
}

func (suite *SqliteTestSuite) TestItCanRemoveExecution() {
	executions := sqliteExecutionsProvider()

	for _, exec := range executions {
		_ = suite.handler.Save(exec)
		err := suite.handler.Remove(exec)
		suite.Assert().NoError(err)
	}

	savedExecs, _ := suite.handler.LoadExecutions()

	suite.Assert().Len(savedExecs, 0)
}

func (suite *SqliteTestSuite) TestItCanFindOne() {
	executions := sqliteExecutionsProvider()

	for _, exec := range executions {
		_, _ = suite.db.Exec(
			`INSERT INTO "`+SqliteExecutionsTable+`" VALUES (?, ?, ?)`,
			exec.Version, exec.ExecutedAtMs, exec.FinishedAtMs,
		)
	}

	execToFind := executions[uint64(4)]
	foundExec, err := suite.handler.FindOne(uint64(4))
	suite.Assert().Equal(&execToFind, foundExec)
	suite.Assert().Nil(err)
	_, _ = suite.db.Exec(`DELETE FROM "` + SqliteExecutionsTable + `"`)
	foundExec, err = suite.handler.FindOne(uint64(4))
	suite.Assert().Nil(foundExec)
	suite.Assert().Nil(err)
}
