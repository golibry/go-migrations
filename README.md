# Go Migrations

Database migrations tool and library for Go projects. It lets you organize and run schema changes from Go files and can be used as a standalone tool for sequential workflows. Migrated from https://github.com/rsgcata/go-migrations

Note: For step-by-step usage, commands, and full working demos, see the _examples directory. All code and command examples have been moved there to keep this README focused on features and concepts.

## What it does

- Runs Go-defined migrations sequentially and remembers what ran
- Supports reversible migrations with Up() and Down()
- Works for database schema management and other ordered workflows
- Manual registration for explicit dependency wiring and type safety
- Execution state persisted in a storage backend (per supported DB)
- Simple CLI with commands to run, rollback, inspect, generate, and force

## Supported storage backends and build tags

- MySQL/MariaDB: build tag mysql
- MongoDB: build tag mongo
- PostgreSQL: build tag postgres

Refer to _examples/README.md for how to build the CLI with the appropriate tags and how to run against each backend.

## How it works (high level)

- A migration is a Go file that implements the Migration interface with Version(), Up(), and Down()
- Migration files are conventionally named version_<unix_timestamp>.go
- You create a registry and manually register your migrations; ordering is handled by version
- An executions repository records applied versions in your storage backend
- The CLI boots with your registry, repository, migrations directory, and optional process-level locking

## CLI overview

Available commands include: help, up, down, blank, stats, force:up, force:down.

For build instructions and concrete usage examples of each command, see the _examples folder.

## Examples and getting started

Complete, runnable examples are provided under _examples for all supported backends (mysql, mongo, postgres). The examples include:

- Building the migrate binary with build tags
- Configuration via environment variables
- Running and rolling back migrations

Start with _examples/README.md for instructions.

## Recommendations & hints

- No DB-level locking is performed by the repository layer. In distributed setups, prefer controlling concurrency at the process or orchestration level.
- Write migrations to be idempotent when possible. Use transactions and checkpoints for large data changes to improve safety and retryability.
- When bootstrapping the CLI, prefer separate DB handles for the executions repository and the migrations’ own DB usage to avoid session conflicts in SQL databases.
