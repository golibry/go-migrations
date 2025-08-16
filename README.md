# Go Migrations

**A database migrations tool & library**. It's targeted for Go projects, but it can be used as a 
standalone tool for any use case. It gives **flexibility** by allowing you to **organize and run 
database schema changes from Go files**.
So you are free to put any functionality you want in 
these migration files, even if they are not strictly related to database schema management.  
Migrated from https://github.com/rsgcata/go-migrations

## Use case, features, usage  

_**TLDR**_: read **README** file from **_examples** folder  

"**Go migrations**" allows users to **run Go functions sequentially** and it "remembers" the functions that were run so you can go back to a specific Go function or continue with new ones.  
It is mainly targeted for **database schema management**, but it **can be used for running basic, 
sequential workflows**.  

Each **migration** (or a step in a sequential workflow) **is a Go file** which must include a struct 
that implements the generic "Migration" interface. So the struct must define the implementation 
for the Version(), Up() and Down() methods. The file name for this migration file must have the 
following format: version_{unix timestamp seconds}.go. The timestamp part must be the same as the 
one returned by the Version() function. It's best to use the helper CLI command ```blank``` to 
generate blank migration files.  
- **Version()** must return the migration identifier. If the migration file was generated 
  automatically from this tool (see examples README how to play with the tool), this method 
  should not be changed.  
- **Up()** must include the logic for making some changes on the database schema like adding a new 
  column or a new table.  
- **Down()** must include the logic to revert the changes done by Up()  

The project does not include pre-built binaries, so you will have to prepare a main entrypoint 
file and build a binary on your own. **To make this easy, there are a few examples that you can 
use, in the _examples directory**.  
**Build tags** for storage integrations: **mysql** (works with mariadb also), **mongo**, 
**postgres** (more will be added)

## Manual Registration

Go Migrations uses a **manual registration** approach where you explicitly register each migration in your code. This provides full control over the migration setup process and dependency injection.

### How Migration Registration Works

You need to create a registry and register all your migrations with their required dependencies:

```go
// Manual registration - explicit and controlled
allMigrations := []migration.Migration{
    &migrations.Migration1712953077{Db: db},
    &migrations.Migration1712953080{Db: db},
    &migrations.Migration1712953083{Db: db, Ctx: ctx},
}
registry := migration.NewDirMigrationsRegistry(dirPath, allMigrations)
```

### Benefits of Manual Registration

✅ **Explicit control** - You decide exactly how each migration is configured  
✅ **Clear dependencies** - Easy to see what each migration requires  
✅ **Type safety** - Compile-time verification of migration setup  
✅ **Simple debugging** - Straightforward to troubleshoot registration issues  
✅ **No reflection overhead** - Direct instantiation without runtime introspection  

### Migration Structure Requirements

Your migrations must implement the `Migration` interface:

```go
// Migration struct with dependencies as fields
type Migration1712953077 struct {
    Db  *sql.DB           // Database connection
    Ctx context.Context   // Context (optional)
}

// Must implement Migration interface
func (m *Migration1712953077) Version() uint64 { return 1712953077 }
func (m *Migration1712953077) Up() error { /* migration logic */ }
func (m *Migration1712953077) Down() error { /* rollback logic */ }
```

### Complete Working Examples

See the complete working examples with different database types:

- **MySQL**: `_examples/mysql/main.go` - Shows SQL database integration
- **MongoDB**: `_examples/mongo/main.go` - Shows NoSQL database integration

Both examples demonstrate proper manual registration with dependency injection.

## CLI Usage Examples

The Go Migrations tool provides a command-line interface with several commands to manage your database migrations. Here's a detailed guide on how to use each command:

### Building the CLI

Before using the commands, you need to build the CLI binary. In your project directory:

```bash
# For MySQL
go build -tags mysql -o ./bin/migrate

# For MongoDB
go build -tags mongo -o ./bin/migrate

# For PostgresSQL
go build -tags postgres -o ./bin/migrate
```

### Available Commands

#### `help`

Displays help information about all available commands.

```bash
./bin/migrate help
```

#### `up`

Executes the `Up()` method for migrations that haven't been executed yet.

```bash
# Run one migration (default)
./bin/migrate up

# Run a specific number of migrations
./bin/migrate up --steps=3

# Run all pending migrations
./bin/migrate up --steps=all
```

#### `down`

Executes the `Down()` method for migrations that have been previously executed, effectively rolling them back.

```bash
# Roll back one migration (default)
./bin/migrate down

# Roll back a specific number of migrations
./bin/migrate down --steps=3

# Roll back all executed migrations
./bin/migrate down --steps=all
```

#### `force:up`

Forcefully executes the `Up()` method for a specific migration version, even if it has been 
executed before. This can be useful for re-running migrations that need to be applied again. This can be a destructive command. Executions and migrations state are not validated when running this command.  

```bash
./bin/migrate force:up --version=1712953077
```

#### `force:down`

Forcefully executes the `Down()` method for a specific migration version, even if it hasn't been 
executed or has already been rolled back. This can be a destructive command. Executions and 
migrations state are not validated when running this command.  

```bash
./bin/migrate force:down --version=1712953077
```

#### `blank`

Generates a new blank migration file in the configured migrations' directory.

```bash
./bin/migrate blank
```

#### `stats`

Displays statistics about registered migrations and their execution status. It also validates if the executions and migrations state are valid and consistent (if it's safe to run up or down).

```bash
./bin/migrate stats
```

## Testing

This section explains how to run all tests from the command line without needing to SSH into Docker containers.

### Prerequisites

Before running tests, you need to have the required database services running. You can use the provided docker-compose.yaml to start them:

```bash
# Start all database services
docker-compose up mongo mysql postgres -d

# Or start individual services
docker-compose up mongo -d     # For MongoDB tests
docker-compose up mysql -d     # For MySQL tests  
docker-compose up postgres -d  # For PostgresSQL tests
```

### Running Tests by Build Tag

The project has tests with different build tags that require specific database connections:

#### MongoDB Tests

```bash
# Run MongoDB tests
go test -tags mongo -v ./execution/repository/

# With custom environment variables
$env:MONGO_DSN="mongodb://localhost:27017"; $env:MONGO_DATABASE="migrations"; go test -tags mongo -v ./execution/repository/
```

**Environment Variables:**
- `MONGO_DSN`: MongoDB connection string (default: `mongodb://localhost:27017`)
- `MONGO_DATABASE`: Database name (default: `migrations`)

#### MySQL Tests

```bash
# Run MySQL tests
go test -tags mysql -v ./execution/repository/

# With custom environment variables
$env:MYSQL_DSN="root:123456789@tcp(localhost:3306)/migrations"; $env:MYSQL_DATABASE="migrations"; go test -tags mysql -v ./execution/repository/
```

**Environment Variables:**
- `MYSQL_DSN`: MySQL connection string (default: `root:123456789@tcp(localhost:3306)/migrations`)
- `MYSQL_DATABASE`: Database name (default: `migrations`)

#### PostgresSQL Tests

```bash
# Run PostgresSQL tests
go test -tags postgres -v ./execution/repository/

# With custom environment variables
$env:POSTGRES_DSN="postgres://postgres:123456789@localhost:5432/migrations?sslmode=disable"; $env:POSTGRES_DATABASE="migrations"; go test -tags postgres -v ./execution/repository/
```

**Environment Variables:**
- `POSTGRES_DSN`: PostgresSQL connection string (default: `postgres://postgres:123456789@localhost:5432/migrations?sslmode=disable`)
- `POSTGRES_DATABASE`: Database name (default: `migrations`)

### Running All Tests Together

To run all tests (with and without build tags) in one command:

```bash
# Start all database services first
docker-compose up mongo mysql postgres -d

# Run all tests with all build tags
go test -tags "mongo mysql postgres" -v ./...
```

### Important Notes

1. **Database Services**: Make sure the required database services are running before executing tests with build tags
2. **Test Isolation**: Each test suite creates and drops its own test database to ensure isolation
3. **Default Credentials**: The default connection strings use simple credentials suitable for development/testing
4. **Build Tags**: Tests with build tags (`mongo`, `mysql`, `postgres`) will only run when the corresponding tag is specified
5. **Environment Variables**: You can override default connection settings using environment variables

## Recommendations & hints

No database locking is done while persisting migration execution changes in the repository.
This is due to the fact that, in distributed systems, it's hard to manage cluster level
locking (for example, at the time of writing, year 2024, MariaDB does not support advisory locking or table locks with Galera Cluster).
It is preferred to give locking control to the caller, for example, if automatic migrations
are run via a process manager or scheduler, make sure they do not allow concurrent or parallel
runs.
Also, it is best to write your migrations to be idempotent.
The library was built with flexibility in mind, so you are free to add anything in the
Up() or Down() migration functions. For example, use SQL "... if not exists ..." clause to make
a table creation idempotent. If big tables need to be populated, use transactions or custom
checkpoints for data changes to allow retries from a checkpoint if part of the batched queries
failed.  

The only locking mechanism, at the moment, can be obtained at service level (OS level), by 
providing the right settings when bootstrapping the migration tool. The locking mechanism is 
based on file locks (maybe more will be added in the future, like Redis or Etcd for implementing 
 semaphore).  

When bootstrapping the migration cli, it is advised to use different db handles, one for the 
migration repository and another for your migration files (if you need any). This is due to the 
fact that, in the SQL database scenarios, some features like `LOCK TABLES`, if used in the 
migration files, may conflict with the migrations repository queries. So either you make sure 
you are doing some cleanup in Up(), Down() migration functions, if needed, or, use different db handles, connections. The examples from the _examples folder have been implemented with these aspects in mind.
