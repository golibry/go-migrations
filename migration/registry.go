package migration

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// MigrationsRegistry allows implementations to manage a collection of migration files.
// Implementations should act as a single source for all created migrations.
type MigrationsRegistry interface {
	// Register must push a migration in the registry. It should fail with error if the
	// migration can't be registered, for example, it its version overlaps with an
	// already registered migration
	Register(migration Migration) error

	// OrderedVersions must return a list of all registered migration versions,
	// ordered in ascending order. Can be used to determine the order in which the migrations
	// should run.
	OrderedVersions() []uint64

	// OrderedMigrations must return a list of all registered migrations,
	// ordered in ascending order by using their version. Can be used to determine the
	// order in which the migrations should run.
	OrderedMigrations() []Migration

	// Get must find and return the migration from the registry, by using its version.
	Get(version uint64) Migration

	// Count must return the total number of registered migrations.
	Count() int
}

// GenericRegistry is a generic implementation for MigrationsRegistry
type GenericRegistry struct {
	migrations map[uint64]Migration
}

// NewGenericRegistry creates a new, empty registry
func NewGenericRegistry() *GenericRegistry {
	return &GenericRegistry{make(map[uint64]Migration)}
}

func (registry *GenericRegistry) Register(migration Migration) error {
	if _, ok := registry.migrations[migration.Version()]; ok {
		return errors.New(
			"failed to register new migration. The migration is already registered",
		)
	}

	registry.migrations[migration.Version()] = migration
	return nil
}

func (registry *GenericRegistry) OrderedVersions() []uint64 {
	var versions []uint64
	for _, mig := range registry.migrations {
		versions = append(versions, mig.Version())
	}
	slices.Sort(versions)
	return versions
}

func (registry *GenericRegistry) OrderedMigrations() []Migration {
	var orderedMigrations []Migration
	for _, mig := range registry.migrations {
		orderedMigrations = append(orderedMigrations, mig)
	}

	sort.Slice(
		orderedMigrations, func(i, j int) bool {
			return orderedMigrations[i].Version() < orderedMigrations[j].Version()
		},
	)

	return orderedMigrations
}

func (registry *GenericRegistry) Get(version uint64) Migration {
	if mig, ok := registry.migrations[version]; ok {
		return mig
	}
	return nil
}

func (registry *GenericRegistry) Count() int {
	return len(registry.migrations)
}

// DirMigrationsRegistry is an implementation of MigrationsRegistry. It will include
// all migrations available in the specified directory (see struct builder function, there
// you can specify the used directory).
type DirMigrationsRegistry struct {
	GenericRegistry
	dirPath MigrationsDirPath
}

// NewEmptyDirMigrationsRegistry builds an empty migrations registry which can be used
// for the use case where migrations are saved in a directory.
func NewEmptyDirMigrationsRegistry(dirPath MigrationsDirPath) *DirMigrationsRegistry {
	return &DirMigrationsRegistry{*NewGenericRegistry(), dirPath}
}

// NewDirMigrationsRegistry builds a migrations registry with all migrations available
// in the specified directory. Panics if it detects that allMigrations argument does not
// match with whatever migration files exist in the specified dirPath
func NewDirMigrationsRegistry(
	dirPath MigrationsDirPath,
	allMigrations []Migration,
) *DirMigrationsRegistry {
	migRegistry := NewEmptyDirMigrationsRegistry(dirPath)

	for _, mig := range allMigrations {
		if regErr := migRegistry.Register(mig); regErr != nil {
			panic(
				fmt.Errorf(
					"failed to register migration %d: %w", mig.Version(), regErr,
				),
			)
		}
	}

	migRegistry.AssertValidRegistry()
	return migRegistry
}

// HasAllMigrationsRegistered checks if everything from the migrations directory has been
// registered in the registry.
// If it returns false, next 2 return values show which file names are missing and which
// file names are extra, compare to the registered migrations.
// Errors if reading the directory fails (maybe insufficient permissions?)
func (registry *DirMigrationsRegistry) HasAllMigrationsRegistered() (
	bool, []string, []string, error,
) {
	dirEntries, err := os.ReadDir(string(registry.dirPath))
	if err != nil {
		return false, []string{}, []string{}, fmt.Errorf(
			"failed to check if all migrations have been registered."+
				" Dir entries read failed with error: %w", err,
		)
	}

	registeredCopy := make(map[uint64]Migration)
	for _, mig := range registry.migrations {
		registeredCopy[mig.Version()] = mig
	}

	var missing, extra []string
	for _, item := range dirEntries {
		if item.IsDir() || !strings.HasPrefix(item.Name(), FileNamePrefix+FileNameSeparator) {
			continue
		}

		fname := strings.TrimLeft(item.Name(), FileNamePrefix+FileNameSeparator)
		version, err := strconv.Atoi(strings.TrimRight(fname, ".go"))

		if err != nil {
			continue
		}

		if _, ok := registeredCopy[uint64(version)]; ok {
			delete(registeredCopy, uint64(version))
		} else {
			missing = append(missing, item.Name())
		}
	}

	for version := range registeredCopy {
		extra = append(extra, FileNamePrefix+FileNameSeparator+strconv.Itoa(int(version))+".go")
	}

	return len(missing) == 0 && len(extra) == 0, missing, extra, nil
}

// AssertValidRegistry checks if there are any issues with the list of registered
// migrations and panics if it finds any
func (registry *DirMigrationsRegistry) AssertValidRegistry() {
	allRegistered, notRegistered, extraRegistered, registryErr :=
		registry.HasAllMigrationsRegistered()

	if registryErr != nil {
		panic(fmt.Errorf("registry has invalid state: %w", registryErr))
	}

	if !allRegistered {
		notRegisteredMigrations := strings.Join(notRegistered, ", ")
		extraMigrations := strings.Join(extraRegistered, ", ")
		if notRegisteredMigrations == "" {
			notRegisteredMigrations = "none"
		}
		if extraMigrations == "" {
			extraMigrations = "none"
		}

		panic(
			fmt.Errorf(
				"registry has invalid state. %s. Not registered: %s. Extra migrations: %s",
				"You must register all migrations before running migrations",
				notRegisteredMigrations,
				extraMigrations,
			),
		)
	}
}

// DependencyProvider is a function type that provides dependencies for migration instantiation.
// It receives the migration type and returns a slice of values to be used as constructor arguments.
type DependencyProvider func(migrationType reflect.Type) []reflect.Value

// AutoDiscoveryConfig holds configuration for auto-discovery of migrations.
type AutoDiscoveryConfig struct {
	// PackageTypes is a slice of example instances from the package to scan.
	// The reflection system will use these to determine which package to scan.
	PackageTypes []interface{}

	// DependencyProvider provides dependencies for migration instantiation.
	DependencyProvider DependencyProvider
}

// NewDirMigrationsRegistryWithAutoDiscovery creates a new DirMigrationsRegistry
// using reflection-based auto-discovery to find and register all migrations
// in the specified packages.
//
// This function scans the provided packages for types that implement the Migration
// interface and automatically instantiates and registers them using the provided
// dependency injection system.
//
// Parameters:
//   - dirPath: The directory path where migration files are located
//   - config: Configuration for auto-discovery including package types and dependency provider
//
// Returns:
//   - *DirMigrationsRegistry: A registry with all discovered migrations registered
//
// Example usage:
//
//	config := &AutoDiscoveryConfig{
//	    PackageTypes: []interface{}{&migrations.Migration1712953077{}},
//	    DependencyProvider: func(migrationType reflect.Type) []reflect.Value {
//	        // Return appropriate dependencies based on migration type
//	        return []reflect.Value{reflect.ValueOf(db), reflect.ValueOf(ctx)}
//	    },
//	}
//	registry := NewDirMigrationsRegistryWithAutoDiscovery(dirPath, config)
func NewDirMigrationsRegistryWithAutoDiscovery(
	dirPath MigrationsDirPath,
	config *AutoDiscoveryConfig,
) *DirMigrationsRegistry {
	discoveredMigrations := DiscoverMigrations(config)
	return NewDirMigrationsRegistry(dirPath, discoveredMigrations)
}

// NewAutoDiscoveryDirMigrationsRegistry creates a new DirMigrationsRegistry with
// a simplified auto-discovery API that automatically finds all migration types
// by scanning the provided package examples.
//
// This is a convenience function that makes it easier to use auto-discovery
// without manually configuring all the details.
//
// Parameters:
//   - dirPath: The directory path where migration files are located
//   - dependencyProvider: Function that provides dependencies for migration instantiation
//   - packageExamples: Example instances from packages to scan (e.g., &migrations.Migration1712953077{})
//
// Returns:
//   - *DirMigrationsRegistry: A registry with all discovered migrations registered
//
// Example usage:
//
//	registry := NewAutoDiscoveryDirMigrationsRegistry(
//	    dirPath,
//	    func(migrationType reflect.Type) []reflect.Value {
//	        return []reflect.Value{reflect.ValueOf(db), reflect.ValueOf(ctx)}
//	    },
//	    &migrations.Migration1712953077{}, // This tells the system to scan the migrations package
//	)
func NewAutoDiscoveryDirMigrationsRegistry(
	dirPath MigrationsDirPath,
	dependencyProvider DependencyProvider,
	packageExamples ...interface{},
) *DirMigrationsRegistry {
	config := &AutoDiscoveryConfig{
		PackageTypes:       packageExamples,
		DependencyProvider: dependencyProvider,
	}
	return NewDirMigrationsRegistryWithAutoDiscovery(dirPath, config)
}

// DiscoverMigrations uses reflection to find all types that implement the Migration
// interface in the specified packages and instantiates them using the dependency provider.
// This function is exported to allow for more flexible usage of the auto-discovery system.
func DiscoverMigrations(config *AutoDiscoveryConfig) []Migration {
	var migrations []Migration
	migrationInterface := reflect.TypeOf((*Migration)(nil)).Elem()

	// Get unique packages from the provided package types
	packages := make(map[string]bool)
	for _, pkgType := range config.PackageTypes {
		pkgPath := reflect.TypeOf(pkgType).Elem().PkgPath()
		packages[pkgPath] = true
	}

	// For each package, scan for migration types
	for _, pkgType := range config.PackageTypes {
		pkgValue := reflect.ValueOf(pkgType)
		if pkgValue.Kind() == reflect.Ptr {
			pkgValue = pkgValue.Elem()
		}

		pkgTypeInfo := pkgValue.Type()

		// Check if this type implements Migration interface
		if reflect.PtrTo(pkgTypeInfo).Implements(migrationInterface) {
			// Get dependencies from the provider
			dependencies := config.DependencyProvider(pkgTypeInfo)

			// Create new instance of the migration type
			migrationPtr := reflect.New(pkgTypeInfo)
			migrationValue := migrationPtr.Elem()

			// Set field values using provided dependencies
			if err := setMigrationFields(migrationValue, dependencies); err != nil {
				panic(
					fmt.Errorf(
						"failed to set dependencies for migration %s: %w",
						pkgTypeInfo.Name(), err,
					),
				)
			}

			// Convert to Migration interface
			migration := migrationPtr.Interface().(Migration)
			migrations = append(migrations, migration)
		}
	}

	return migrations
}

// setMigrationFields sets the fields of a migration struct using the provided dependencies.
// It matches dependencies to struct fields based on type compatibility.
func setMigrationFields(migrationValue reflect.Value, dependencies []reflect.Value) error {
	migrationStruct := migrationValue.Type()

	// Create a map of dependency types to values for an easy lookup
	depMap := make(map[reflect.Type]reflect.Value)
	for _, dep := range dependencies {
		depMap[dep.Type()] = dep
	}

	// Set each field in the migration struct
	for i := 0; i < migrationStruct.NumField(); i++ {
		field := migrationStruct.Field(i)
		fieldValue := migrationValue.Field(i)

		// Skip unexported fields
		if !fieldValue.CanSet() {
			continue
		}

		// Find matching dependency by type
		if dep, exists := depMap[field.Type]; exists {
			fieldValue.Set(dep)
		} else {
			// Try to find compatible interface types
			for depType, depValue := range depMap {
				if depType.AssignableTo(field.Type) {
					fieldValue.Set(depValue)
					break
				}
			}
		}
	}

	return nil
}
