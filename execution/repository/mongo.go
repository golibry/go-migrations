//go:build mongo

// Package repository includes migration execution persistence related
// logic via execution.Repository implementations.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golibry/go-migrations/execution"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type bsonExecution struct {
	Version      uint64 `bson:"_id"`
	ExecutedAtMs uint64 `bson:"executedAtMs"`
	FinishedAtMs uint64 `bson:"finishedAtMs"`
}

func toBsonExecution(exec execution.MigrationExecution) bsonExecution {
	return bsonExecution{
		Version:      exec.Version,
		ExecutedAtMs: exec.ExecutedAtMs,
		FinishedAtMs: exec.FinishedAtMs,
	}
}

func toMigrationExecution(exec bsonExecution) execution.MigrationExecution {
	return execution.MigrationExecution{
		Version:      exec.Version,
		ExecutedAtMs: exec.ExecutedAtMs,
		FinishedAtMs: exec.FinishedAtMs,
	}
}

func newMongoClient(dsn string, ctx context.Context) (*mongo.Client, error) {
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(dsn).SetServerAPIOptions(serverAPI)
	opts.SetMaxPoolSize(1)
	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, err
	}
	if err = client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return client, nil
}

// MongoHandler Repository implementation for MongoDb integration
type MongoHandler struct {
	client         *mongo.Client
	databaseName   string
	collectionName string
	lockName       string
	ctx            context.Context
}

// NewMongoHandler Builds a new MongoHandler. If client is nil, it will try to build a client
// from the provided dsn. It is recommended to share the same *mongo.Client handle between
// your application and this handler to efficiently manage connection pools.
func NewMongoHandler(
	dsn string,
	databaseName string,
	collectionName string,
	ctx context.Context,
	client *mongo.Client,
) (*MongoHandler, error) {
	if client == nil {
		var err error
		client, err = newMongoClient(dsn, ctx)

		if err != nil {
			return nil, err
		}
	}

	return &MongoHandler{
		client:         client,
		databaseName:   databaseName,
		collectionName: collectionName,
		lockName:       "go-migrations:" + databaseName + "." + collectionName,
		ctx:            ctx,
	}, nil
}

func (h *MongoHandler) Context() context.Context {
	return h.ctx
}

func (h *MongoHandler) Init() error {
	names, err := h.client.Database(h.databaseName).ListCollectionNames(h.ctx, bson.D{})

	if err != nil {
		return err
	}

	for _, name := range names {
		if name == h.collectionName {
			return nil
		}
	}

	collectionOpts := options.CreateCollection()
	collectionOpts.SetValidator(
		bson.D{
			{
				Key: "$jsonSchema", Value: bson.D{
					{Key: "bsonType", Value: "object"},
					{Key: "title", Value: "migration execution object validation"},
					{
						Key: "properties", Value: bson.D{
							{
								Key: "_id", Value: bson.D{
									{Key: "bsonType", Value: "long"},
									{Key: "minimum", Value: 0},
									{
										Key: "description",
										Value: "_id (executed version) must be greater or equal" +
											" to 0",
									},
								},
							},
							{
								Key: "executedAtMs", Value: bson.D{
									{Key: "bsonType", Value: "long"},
									{Key: "minimum", Value: 0},
									{
										Key:   "description",
										Value: "executed at must be greater or equal to 0",
									},
								},
							},
							{
								Key: "finishedAtMs", Value: bson.D{
									{Key: "bsonType", Value: "long"},
									{Key: "minimum", Value: 0},
									{
										Key:   "description",
										Value: "finished at must be greater or equal to 0",
									},
								},
							},
						},
					},
				},
			},
		},
	)

	return h.client.Database(h.databaseName).CreateCollection(
		h.ctx, h.collectionName, collectionOpts,
	)
}

func (h *MongoHandler) Lock() (func() error, error) {
	ctx, cancel := context.WithTimeout(h.ctx, 30*time.Second)
	defer cancel()

	collection := h.client.Database(h.databaseName).Collection(h.collectionName + "_locks")
	lockDoc := bson.D{
		{Key: "_id", Value: h.lockName},
		{Key: "createdAtMs", Value: time.Now().UnixMilli()},
	}

	for {
		_, err := collection.InsertOne(ctx, lockDoc)
		if err == nil {
			return func() error {
				_, deleteErr := collection.DeleteOne(h.ctx, bson.D{{Key: "_id", Value: h.lockName}})
				return deleteErr
			}, nil
		}

		var writeException mongo.WriteException
		if !errors.As(err, &writeException) || !writeException.HasErrorCode(11000) {
			return nil, err
		}

		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("failed to acquire MongoDB migration lock %q: %w", h.lockName, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (h *MongoHandler) LoadExecutions() (executions []execution.MigrationExecution, err error) {
	collection := h.client.Database(h.databaseName).Collection(h.collectionName)
	cursor, err := collection.Find(h.ctx, bson.D{})

	if err != nil {
		return nil, err
	}

	var bsonExecutions []bsonExecution
	if err = cursor.All(h.ctx, &bsonExecutions); err != nil {
		return nil, err
	}

	var migrationExecutions []execution.MigrationExecution
	for _, b := range bsonExecutions {
		migrationExecutions = append(migrationExecutions, toMigrationExecution(b))
	}

	return migrationExecutions, nil
}

func (h *MongoHandler) Save(exec execution.MigrationExecution) error {
	collection := h.client.Database(h.databaseName).Collection(h.collectionName)
	filter := bson.D{{Key: "_id", Value: exec.Version}}
	updateOpts := options.UpdateOne()
	updateOpts.SetUpsert(true)
	_, err := collection.UpdateOne(
		h.ctx, filter, bson.D{{Key: "$set", Value: toBsonExecution(exec)}}, updateOpts,
	)
	return err
}

func (h *MongoHandler) Remove(exec execution.MigrationExecution) error {
	collection := h.client.Database(h.databaseName).Collection(h.collectionName)
	filter := bson.D{{Key: "_id", Value: exec.Version}}
	_, err := collection.DeleteOne(h.ctx, filter)
	return err
}

func (h *MongoHandler) FindOne(version uint64) (*execution.MigrationExecution, error) {
	collection := h.client.Database(h.databaseName).Collection(h.collectionName)
	filter := bson.D{{Key: "_id", Value: version}}

	var result bsonExecution
	err := collection.FindOne(h.ctx, filter).Decode(&result)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	exec := toMigrationExecution(result)
	return &exec, err
}
