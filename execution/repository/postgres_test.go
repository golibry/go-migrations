//go:build postgres

package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/golibry/go-migrations/execution"
	"github.com/golibry/go-migrations/migration"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/suite"
	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const PostgresExecutionsTable = "migration_executions"

type PostgresTestSuite struct {
	suite.Suite
	dbName    string
	dsn       string
	db        *sql.DB
	handler   *PostgresHandler
	container *pgcontainer.PostgresContainer
}

func TestPostgresTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresTestSuite))
}

func (suite *PostgresTestSuite) SetupSuite() {
	// Start a Postgres testcontainer
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pgC, err := pgcontainer.Run(
		ctx,
		"postgres:16",
		pgcontainer.WithDatabase("migrations"),
		pgcontainer.WithUsername("postgres"),
		pgcontainer.WithPassword("postgres"),
	)
	suite.Require().NoError(err)
	suite.container = pgC

	connStr, err := pgC.ConnectionString(ctx, "sslmode=disable")
	suite.Require().NoError(err)
	suite.dsn = connStr
	suite.dbName = "migrations"

    suite.handler, err = NewPostgresHandler(
        suite.dsn,
        PostgresExecutionsTable,
        context.Background(),
        nil,
    )
    suite.Require().NoError(err)
    suite.db = suite.handler.db

    // Wait for the database to become ready (max 20s)
    deadline := time.Now().Add(20 * time.Second)
    var pingErr error
    for {
        // Use a short per-ping timeout
        ctxPing, cancelPing := context.WithTimeout(context.Background(), 1*time.Second)
        pingErr = suite.db.PingContext(ctxPing)
        cancelPing()
        if pingErr == nil {
            break
        }
        if time.Now().After(deadline) {
            break
        }
        time.Sleep(500 * time.Millisecond)
    }
    suite.Require().NoError(pingErr)
}

func (suite *PostgresTestSuite) TearDownSuite() {
	_ = suite.db.Close()
	if suite.container != nil {
		_ = suite.container.Terminate(context.Background())
	}
}

func (suite *PostgresTestSuite) SetupTest() {
	_ = suite.handler.Init()
	_, _ = suite.db.Exec(`DELETE FROM "` + PostgresExecutionsTable + `"`)
}

func (suite *PostgresTestSuite) TearDownTest() {
	_, _ = suite.db.Exec(`DELETE FROM "` + PostgresExecutionsTable + `"`)
}

func (suite *PostgresTestSuite) TestItCanBuildMigrationsExclusiveDbHandle() {
	handle, err := newDbHandle(suite.dsn, "postgres")

	suite.Assert().Nil(err)
	suite.Assert().Equal(1, handle.Stats().MaxOpenConnections)

	var dbName string
	_ = handle.QueryRow("SELECT current_database()").Scan(&dbName)
	suite.Assert().Equal(suite.dbName, dbName)
}

func (suite *PostgresTestSuite) TestItCanBuildHandlerWithProvidedContext() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler, err := NewPostgresHandler(suite.dsn, "migration_execs", ctx, nil)
	suite.Assert().Nil(err)
	suite.Assert().Same(ctx, handler.Context())
}

func (suite *PostgresTestSuite) TestItCanInitializeExecutionsTable() {
	_, _ = suite.db.Exec(`DROP TABLE IF EXISTS "` + PostgresExecutionsTable + `"`)
	tableExists := func() bool {
		var exists bool
		_ = suite.db.QueryRow(
			`
            SELECT EXISTS (
                SELECT FROM pg_tables 
                WHERE schemaname = 'public' 
                AND tablename = $1
            )`, PostgresExecutionsTable,
		).Scan(&exists)
		return exists
	}

	suite.Assert().False(tableExists())
	_ = suite.handler.Init()
	suite.Assert().True(tableExists())
}

func postgresExecutionsProvider() map[uint64]execution.MigrationExecution {
	return map[uint64]execution.MigrationExecution{
		uint64(1): {Version: 1, ExecutedAtMs: 2, FinishedAtMs: 3},
		uint64(4): {Version: 4, ExecutedAtMs: 5, FinishedAtMs: 6},
		uint64(7): {Version: 7, ExecutedAtMs: 8, FinishedAtMs: 9},
	}
}

func (suite *PostgresTestSuite) TestItCanLoadExecutions() {
	executions := postgresExecutionsProvider()

	for _, exec := range executions {
		_, _ = suite.db.Exec(
			`INSERT INTO "`+PostgresExecutionsTable+`" VALUES ($1, $2, $3)`,
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

func (suite *PostgresTestSuite) TestItFailsToExecuteAnyChangesWhenMissingTable() {
	_, _ = suite.db.Exec(`DROP TABLE IF EXISTS "` + PostgresExecutionsTable + `"`)
	migrationExecution := execution.StartExecution(migration.NewDummyMigration(123))
	_, errLoad := suite.handler.LoadExecutions()
	errSave := suite.handler.Save(*migrationExecution)
	errRemove := suite.handler.Remove(*migrationExecution)
	_, errFindOne := suite.handler.FindOne(uint64(123))

	suite.Assert().Error(errLoad)
	suite.Assert().ErrorContains(errLoad, PostgresExecutionsTable)
	suite.Assert().Error(errSave)
	suite.Assert().ErrorContains(errSave, PostgresExecutionsTable)
	suite.Assert().Error(errRemove)
	suite.Assert().ErrorContains(errRemove, PostgresExecutionsTable)
	suite.Assert().Error(errFindOne)
	suite.Assert().ErrorContains(errFindOne, PostgresExecutionsTable)
}

func (suite *PostgresTestSuite) TestItFailsToLoadExecutionsFromInvalidRepoData() {
	_, _ = suite.db.Exec(
		`ALTER TABLE "` + PostgresExecutionsTable + `" 
         ALTER COLUMN finished_at_ms DROP NOT NULL`,
	)
	_, _ = suite.db.Exec(
		`INSERT INTO "` + PostgresExecutionsTable + `" 
         VALUES (1, 2, 1), (3, 4, NULL)`,
	)
	execs, err := suite.handler.LoadExecutions()
	suite.Assert().Len(execs, 1)
	suite.Assert().Error(err)
	suite.Assert().ErrorContains(err, "Scan error")
}

func (suite *PostgresTestSuite) TestItCanSaveExecutions() {
	// Insert
	executions := postgresExecutionsProvider()

	for _, exec := range executions {
		err := suite.handler.Save(exec)
		suite.Assert().NoError(err)
	}

	savedExecs, _ := suite.handler.LoadExecutions()
	for _, exec := range savedExecs {
		suite.Assert().Contains(executions, exec.Version)
		suite.Assert().Equal(executions[exec.Version], exec)
	}

	// Update
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

func (suite *PostgresTestSuite) TestItCanRemoveExecution() {
	executions := postgresExecutionsProvider()

	for _, exec := range executions {
		_ = suite.handler.Save(exec)
		err := suite.handler.Remove(exec)
		suite.Assert().NoError(err)
	}

	savedExecs, _ := suite.handler.LoadExecutions()

	suite.Assert().Len(savedExecs, 0)
}

func (suite *PostgresTestSuite) TestItCanFindOne() {
	executions := postgresExecutionsProvider()

	for _, exec := range executions {
		_, _ = suite.db.Exec(
			`INSERT INTO "`+PostgresExecutionsTable+`" VALUES ($1, $2, $3)`,
			exec.Version, exec.ExecutedAtMs, exec.FinishedAtMs,
		)
	}

	execToFind := executions[uint64(4)]
	foundExec, err := suite.handler.FindOne(uint64(4))
	suite.Assert().Equal(&execToFind, foundExec)
	suite.Assert().Nil(err)
	_, _ = suite.db.Exec(`DELETE FROM "` + PostgresExecutionsTable + `"`)
	foundExec, err = suite.handler.FindOne(uint64(4))
	suite.Assert().Nil(foundExec)
	suite.Assert().Nil(err)
}
