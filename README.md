# Go Migrations

Database migrations tool and library for Go projects. It lets you organize and run schema changes from Go files and can be used as a standalone tool for sequential workflows. Migrated from https://github.com/rsgcata/go-migrations

Note: For step-by-step usage, commands, and full working demos, see the _examples directory. All code and command examples have been moved there to keep this README focused on features and concepts.

## What it does

- Runs Go-defined migrations sequentially and remembers what ran
- Supports reversible migrations with Up() and Down()
- Works for database schema management and other ordered workflows
- Flexible registration: supports both manual registration and automatic registration via Go `init()` functions
- Transaction support guidance: documentation and examples for atomic migrations in SQL and NoSQL
- Efficient connection management: explicitly designed to share database handles with your application
- Execution state persisted in a storage backend (per supported DB)
- Simple CLI with commands to run, rollback, inspect, generate, and force

## Supported storage backends and build tags

- MySQL/MariaDB: build tag mysql
- MongoDB: build tag mongo
- PostgreSQL: build tag postgres

Backend integration tests require the matching backend tag and the `integration` tag, for example:

- MySQL/MariaDB: `go test -tags "mysql integration" ./execution/repository`
- MongoDB: `go test -tags "mongo integration" ./execution/repository`
- PostgreSQL: `go test -tags "postgres integration" ./execution/repository`

Refer to _examples/README.md for how to build the CLI with the appropriate tags and how to run against each backend.

## How it works (high level)

- A migration is a Go file that implements the Migration interface with Version(), Up(), and Down()
- Migration files are conventionally named version_<unix_timestamp>.go
- Automatic registration: migrations can self-register using `init()` and `migration.Register()`, making them easy to manage
- The registry (e.g., `NewAutoDirMigrationsRegistry`) validates that all migration files are correctly registered
- An execution repository records applied versions in your storage backend
- The handler persists a started execution before running `Up()`, so crashes during migration work leave a detectable unfinished execution
- Supported database repositories acquire a backend-level lock while running mutation commands
- The CLI boots with your registry, repository, migrations directory, and optional process-level locking

## CLI overview

Available commands include: help, up, down, blank, stats, force:up, force:down, force:finish, force:remove.

`stats` reports dirty state when the latest execution is unfinished. `force:finish` and `force:remove` are repair commands for cases where operators have manually verified the database state after a failed or interrupted migration.

For build instructions and concrete usage examples of each command, see the _examples folder.

## Examples and getting started

Complete, runnable examples are provided under _examples for all supported backends (mysql, mongo, postgres). The examples include:

- Building the "migrate" binary with build tags
- Configuration via environment variables
- Running and rolling back migrations

Start with _examples/README.md for instructions.

## Recommendations & hints

- Supported database repositories acquire backend-level locks for mutation commands. In distributed setups, this can be combined with the CLI's process-level exclusive run settings.
- Write migrations to be idempotent when possible. Use transactions to ensure atomicity and prevent partial migration application.
- Database handles can be shared between your application and the migration executions.
