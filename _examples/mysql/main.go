package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/golibry/go-migrations/_examples/mysql/migrations"
	"github.com/golibry/go-migrations/cli"
	"github.com/golibry/go-migrations/execution/repository"
	"github.com/golibry/go-migrations/migration"
	"os"
	"path/filepath"
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
	cli.Bootstrap(
		os.Args[1:],
		buildRegistry(dirPath, ctx, dbDsn),
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

// buildRegistry This will create a new registry and register all migrations
func buildRegistry(
	dirPath migration.MigrationsDirPath,
	ctx context.Context,
	dbDsn string,
) *migration.DirMigrationsRegistry {
	// New db needed to not conflict with executions repository connection session
	db, err := sql.Open("mysql", dbDsn)

	if err != nil {
		panic(fmt.Errorf("failed to connect to migrations db: %w", err))
	}

	// OPTION 1: Manual registration (original approach)
	// It's not necessary to add them in order, the tool will handle ordering based on
	// their version number
	allMigrations := []migration.Migration{
		&migrations.Migration1712953077{Db: db},
		&migrations.Migration1712953080{Db: db},
		&migrations.Migration1712953083{Db: db, Ctx: ctx},
	}

	return migration.NewDirMigrationsRegistry(dirPath, allMigrations)

	// OPTION 2: Auto-discovery (new approach) - uncomment to use
	// Using auto-discovery - no need to manually register each migration!
	// Just provide a dependency provider function and one example from the migrations package
	/*
	return migration.NewAutoDiscoveryDirMigrationsRegistry(
		dirPath,
		func(migrationType reflect.Type) []reflect.Value {
			// Provide dependencies based on what each migration type needs
			dependencies := []reflect.Value{reflect.ValueOf(db)}
			
			// Some migrations might need additional dependencies like context
			// Check if the migration has a Ctx field
			for i := 0; i < migrationType.NumField(); i++ {
				field := migrationType.Field(i)
				if field.Name == "Ctx" && field.Type.String() == "context.Context" {
					dependencies = append(dependencies, reflect.ValueOf(ctx))
					break
				}
			}
			
			return dependencies
		},
		&migrations.Migration1712953077{}, // Example to tell the system which package to scan
	)
	*/
}
