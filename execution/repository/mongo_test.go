//go:build mongo

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/golibry/go-migrations/execution"
	"github.com/stretchr/testify/suite"
	mongodbtc "github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const MongoCollectionName = "migration_executions"

type MongoTestSuite struct {
	suite.Suite
	dbName    string
	dsn       string
	client    *mongo.Client
	handler   *MongoHandler
	container *mongodbtc.MongoDBContainer
}

func TestMongoTestSuite(t *testing.T) {
	suite.Run(t, new(MongoTestSuite))
}

func (suite *MongoTestSuite) SetupSuite() {
	// Start a MongoDB testcontainer
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Use a stable image; tests do not need auth
	mongoC, err := mongodbtc.Run(
		ctx,
		"mongo:8.2",
	)
	suite.Require().NoError(err)
	suite.container = mongoC

	dsn, err := mongoC.ConnectionString(ctx)
	suite.Require().NoError(err)
	suite.dsn = dsn
	suite.dbName = "migrations"

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(suite.dsn).SetServerAPIOptions(serverAPI)
	opts.SetMaxPoolSize(1)
	opts.SetMaxConnIdleTime(3 * time.Second)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetServerSelectionTimeout(20 * time.Second)
	opts.SetTimeout(20 * time.Second)
	opts.SetSocketTimeout(20 * time.Second)
	client, err := mongo.Connect(context.Background(), opts)
	suite.Require().NoError(err)

	suite.handler = &MongoHandler{client, suite.dbName, MongoCollectionName, context.Background()}
	suite.client = suite.handler.client
	suite.Require().NoError(suite.handler.Init())
}

func (suite *MongoTestSuite) TearDownSuite() {
	_ = suite.client.Disconnect(context.Background())
	if suite.container != nil {
		_ = suite.container.Terminate(context.Background())
	}
}

func (suite *MongoTestSuite) SetupTest() {
	_, _ = suite.client.Database(suite.dbName).Collection(MongoCollectionName).DeleteMany(
		context.Background(), bson.D{},
	)
}

func (suite *MongoTestSuite) TearDownTest() {
	_, _ = suite.client.Database(suite.dbName).Collection(MongoCollectionName).DeleteMany(
		context.Background(), bson.D{},
	)
}

func (suite *MongoTestSuite) TestItCanInitializeTheRepository() {
	_ = suite.client.Database(suite.dbName).Collection(MongoCollectionName).
		Drop(context.Background())
	errInit1 := suite.handler.Init()
	errInit2 := suite.handler.Init()
	suite.Assert().Nil(errInit1)
	suite.Assert().Nil(errInit2)
	names, _ := suite.client.Database(suite.dbName).ListCollectionNames(suite.handler.ctx, bson.D{})
	suite.Assert().Contains(names, MongoCollectionName)
}

func mongoExecutionsProvider() map[uint64]execution.MigrationExecution {
	return map[uint64]execution.MigrationExecution{
		uint64(1): {Version: 1, ExecutedAtMs: 2, FinishedAtMs: 3},
		uint64(4): {Version: 4, ExecutedAtMs: 5, FinishedAtMs: 6},
		uint64(7): {Version: 7, ExecutedAtMs: 8, FinishedAtMs: 9},
	}
}

func (suite *MongoTestSuite) TestItCanLoadAllExecutions() {
	executions := mongoExecutionsProvider()

	for _, exec := range executions {
		_, _ = suite.client.Database(suite.dbName).Collection(MongoCollectionName).InsertOne(
			context.Background(), toBsonExecution(exec),
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

func (suite *MongoTestSuite) TestItCanSaveExecutions() {
	// Insert
	executions := mongoExecutionsProvider()

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

func (suite *MongoTestSuite) TestItCanRemoveExecution() {
	executions := mongoExecutionsProvider()

	for _, exec := range executions {
		_ = suite.handler.Save(exec)
		err := suite.handler.Remove(exec)
		suite.Assert().NoError(err)
	}

	savedExecs, _ := suite.handler.LoadExecutions()

	suite.Assert().Len(savedExecs, 0)
}

func (suite *MongoTestSuite) TestItCanFindOne() {
	executions := mongoExecutionsProvider()

	for _, exec := range executions {
		_ = suite.handler.Save(exec)
	}

	execToFind := executions[uint64(4)]
	foundExec, err := suite.handler.FindOne(uint64(4))
	suite.Assert().Equal(&execToFind, foundExec)
	suite.Assert().Nil(err)
	_ = suite.handler.Remove(*foundExec)
	foundExec, err = suite.handler.FindOne(uint64(4))
	suite.Assert().Nil(foundExec)
	suite.Assert().Nil(err)
}
