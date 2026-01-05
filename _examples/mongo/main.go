package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golibry/go-migrations/cli"
	"github.com/golibry/go-migrations/execution/repository"
	"github.com/golibry/go-migrations/migration"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			switch v := err.(type) {
			case string:
				err = errors.New(v)
			case error:
				err = v
			default:
				err = errors.New(fmt.Sprint(v))
			}
			cmdErr := err.(error)
			fmt.Println("[ERROR] ", cmdErr.Error())
		}
	}()

	ctx := context.Background()
	dirPath := createMigrationsDirPath()
	dbDsn := getDbDsn()
	dbName := getDbName()

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(dbDsn).SetServerAPIOptions(serverAPI)
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		panic(fmt.Errorf("failed to connect to migrations db: %w", err))
	}

	db := client.Database(dbName)

	cli.Bootstrap(
		ctx,
		db,
		os.Args[1:],
		migration.NewAutoDirMigrationsRegistry(dirPath),
		createMongoRepository(client, ctx, dbName),
		dirPath,
		nil,
		os.Stdout,
		os.Exit,
		nil,
	)
}

func createMigrationsDirPath() migration.MigrationsDirPath {
	appBaseDir := os.Getenv("APP_BASE_DIR")

	if appBaseDir == "" {
		appBaseDir = "/go/src/migrations"
	}

	dirPath, err := migration.NewMigrationsDirPath(
		filepath.Join(appBaseDir, "_examples/mongo/migrations"),
	)

	if err != nil {
		panic(fmt.Errorf("invalid migrations path: %w", err))
	}

	return dirPath
}

func createMongoRepository(
	client *mongo.Client,
	ctx context.Context,
	dbName string,
) *repository.MongoHandler {
	repo, err := repository.NewMongoHandler(
		"",
		dbName,
		getCollectionName(),
		ctx,
		client,
	)

	if err != nil {
		panic(fmt.Errorf("failed to build executions repository: %w", err))
	}

	return repo
}

func getDbName() string {
	dbName := os.Getenv("MONGO_DATABASE")

	if dbName == "" {
		dbName = "migrations"
	}

	return dbName
}

func getCollectionName() string {
	collectionName := os.Getenv("MONGO_MIGRATIONS_COLLECTION")

	if collectionName == "" {
		collectionName = "migrations"
	}

	return collectionName
}

// getDbDsn Prepare the Mongo DSN
func getDbDsn() string {
	dsn := os.Getenv("MONGO_DSN")

	if dsn == "" {
		// Needed if ran from the host machine because we are missing the env variables
		// See pass and port in .env file
		dsn = "mongodb://localhost:27017"
	}

	return dsn
}
