package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/golibry/go-migrations/_examples/postgres/migrations"
	"github.com/golibry/go-migrations/cli"
	"github.com/golibry/go-migrations/execution/repository"
	"github.com/golibry/go-migrations/migration"
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
	db, err := sql.Open("postgres", dbDsn)
	if err != nil {
		panic(fmt.Errorf("failed to connect to migrations db: %w", err))
	}

	cli.Bootstrap(
		ctx,
		db,
		os.Args[1:],
		migration.NewAutoDirMigrationsRegistry(dirPath),
		createPostgresRepository(db, ctx),
		dirPath,
		nil,
		os.Stdout,
		os.Exit,
		&cli.BootstrapSettings{
			RunMigrationsExclusively: true,
			RunLockFilesDirPath:      os.TempDir(),
			MigrationsCmdLockName:    "my-app-migrations-lock",
		},
	)
}

func createMigrationsDirPath() migration.MigrationsDirPath {
	appBaseDir := os.Getenv("APP_BASE_DIR")

	if appBaseDir == "" {
		appBaseDir = "/go/src/migrations"
	}

	dirPath, err := migration.NewMigrationsDirPath(
		filepath.Join(appBaseDir, "_examples/postgres/migrations"),
	)

	if err != nil {
		panic(fmt.Errorf("invalid migrations path: %w", err))
	}

	return dirPath
}

func createPostgresRepository(
	db *sql.DB,
	ctx context.Context,
) *repository.PostgresHandler {
	repo, err := repository.NewPostgresHandler("", "migration_executions", ctx, db)

	if err != nil {
		panic(fmt.Errorf("failed to build executions repository: %w", err))
	}

	return repo
}

// getDbDsn Prepare the Postgres DSN
func getDbDsn() string {
	dbName := os.Getenv("POSTGRES_DATABASE")
	dsn := os.Getenv("POSTGRES_DSN")

	if dbName == "" {
		dbName = "migrations"
	}

	if dsn == "" {
		// Default for running from host, adjust user/pass/port as per docker-compose/.env
		dsn = "postgres://postgres:123456789@localhost:5432/" + dbName + "?sslmode=disable"
	}

	return dsn
}
