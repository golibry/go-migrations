package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/golibry/go-migrations/_examples/mysql/migrations"
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
	db, err := sql.Open("mysql", dbDsn)
	if err != nil {
		panic(fmt.Errorf("failed to connect to migrations db: %w", err))
	}

	cli.Bootstrap(
		ctx,
		db,
		os.Args[1:],
		migration.NewAutoDirMigrationsRegistry(dirPath),
		createMysqlRepository(dbDsn, ctx),
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
		filepath.Join(appBaseDir, "_examples/mysql/migrations"),
	)

	if err != nil {
		panic(fmt.Errorf("invalid migrations path: %w", err))
	}

	return dirPath
}

func createMysqlRepository(
	dbDsn string,
	ctx context.Context,
) *repository.MysqlHandler {
	repo, err := repository.NewMysqlHandler(dbDsn, "migration_executions", ctx, nil)

	if err != nil {
		panic(fmt.Errorf("failed to build executions repository: %w", err))
	}

	return repo
}

// getDbDsn Prepare the Mysql DSN
func getDbDsn() string {
	dbName := os.Getenv("MYSQL_DATABASE")
	dsn := os.Getenv("MYSQL_DSN")

	if dbName == "" {
		dbName = "migrations"
	}

	if dsn == "" {
		// Needed if ran from host machine because we are missing the env variables
		// See pass and port in .env file
		dsn = "root:123456789@tcp(localhost:3306)/" + dbName
	}

	return dsn
}
