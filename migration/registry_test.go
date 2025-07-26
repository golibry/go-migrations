package migration

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"testing"

	"github.com/stretchr/testify/suite"
)

type RegistryTestSuite struct {
	suite.Suite
	migrationsDirPath string
}

func TestRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}

func (suite *RegistryTestSuite) SetupTest() {
	suite.migrationsDirPath = os.TempDir() + string(os.PathSeparator) + "migrationsTestDir"

	if err := os.RemoveAll(suite.migrationsDirPath); err != nil {
		panic("could not cleanup test migrations dir")
	}

	if err := os.MkdirAll(suite.migrationsDirPath, os.ModeDir); err != nil {
		panic("could not create test migrations dir")
	}
}

func (suite *RegistryTestSuite) TearDownTest() {
	_ = os.RemoveAll(suite.migrationsDirPath)
}

func (suite *RegistryTestSuite) TestItCanRegisterMigration() {
	version := uint64(1234)
	dm := &DummyMigration{version}
	registry := NewGenericRegistry()
	_ = registry.Register(dm)
	suite.Assert().Equal(dm, registry.Get(version))
}

func (suite *RegistryTestSuite) TestItFailsToRegisterDuplicateMigration() {
	version := uint64(1234)
	dm1 := &DummyMigration{version}
	dm2 := &DummyMigration{version}
	registry := NewGenericRegistry()
	_ = registry.Register(dm1)
	err := registry.Register(dm2)
	suite.Assert().ErrorContains(err, "already registered")
}

func (suite *RegistryTestSuite) TestItCanProvideOrderedRegisteredVersions() {
	versions := []uint64{123, 124, 125}
	registry := NewGenericRegistry()
	_ = registry.Register(&DummyMigration{versions[1]})
	_ = registry.Register(&DummyMigration{versions[0]})
	_ = registry.Register(&DummyMigration{versions[2]})
	suite.Assert().Equal(versions, registry.OrderedVersions())
}

func (suite *RegistryTestSuite) TestItCanProvideOrderedRegisteredMigrations() {
	expectedMigrations := []Migration{
		&DummyMigration{123}, &DummyMigration{124}, &DummyMigration{125},
	}
	registry := NewGenericRegistry()
	_ = registry.Register(expectedMigrations[1])
	_ = registry.Register(expectedMigrations[0])
	_ = registry.Register(expectedMigrations[2])
	suite.Assert().Equal(expectedMigrations, registry.OrderedMigrations())
}

func (suite *RegistryTestSuite) TestItCanGetSpecificRegisteredVersion() {
	registry := NewGenericRegistry()
	for i := 0; i < 999; i++ {
		_ = registry.Register(&DummyMigration{uint64(i)})
	}
	for i := 0; i < 999; i++ {
		mig := registry.Get(uint64(i))
		suite.Assert().Equal(uint64(i), mig.Version())
	}
}

func (suite *RegistryTestSuite) TestItCanCountRegisteredMigrations() {
	registry := NewGenericRegistry()
	expectedCount := 321
	for i := 0; i < 321; i++ {
		_ = registry.Register(&DummyMigration{uint64(i)})
	}
	suite.Assert().Equal(expectedCount, registry.Count())
}

func (suite *RegistryTestSuite) TestItCanValidateAllDirMigrationsAreRegistered() {
	migDir, _ := NewMigrationsDirPath(suite.migrationsDirPath)
	dirRegistry := NewEmptyDirMigrationsRegistry(migDir)

	for i := 1; i < 11; i++ {
		newVersion := uint64(i)
		_ = dirRegistry.Register(&DummyMigration{newVersion})

		migFn := FileNamePrefix + FileNameSeparator + strconv.Itoa(int(newVersion)) + ".go"
		newFilePath := filepath.Join(suite.migrationsDirPath, migFn)
		fp, _ := os.OpenFile(newFilePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		_ = fp.Close()
	}

	allRegistered, missing, extra, err := dirRegistry.HasAllMigrationsRegistered()
	suite.Assert().True(allRegistered)
	suite.Assert().Nil(missing)
	suite.Assert().Nil(extra)
	suite.Assert().Nil(err)
}

func (suite *RegistryTestSuite) TestItCanComputeExtraAndMissingRegisteredMigrations() {
	migDir, _ := NewMigrationsDirPath(suite.migrationsDirPath)
	dirRegistry := NewEmptyDirMigrationsRegistry(migDir)

	for i := 1; i < 5; i++ {
		newVersion := uint64(i)
		migFn := FileNamePrefix + FileNameSeparator + strconv.Itoa(int(newVersion)) + ".go"
		newFilePath := filepath.Join(suite.migrationsDirPath, migFn)
		fp, _ := os.OpenFile(newFilePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		_ = fp.Close()
	}

	_ = dirRegistry.Register(&DummyMigration{1})
	_ = dirRegistry.Register(&DummyMigration{2})
	_ = dirRegistry.Register(&DummyMigration{7})
	_ = dirRegistry.Register(&DummyMigration{8})

	expectedMissing := []string{
		FileNamePrefix + FileNameSeparator + "3.go",
		FileNamePrefix + FileNameSeparator + "4.go",
	}
	expectedExtra := []string{
		FileNamePrefix + FileNameSeparator + "7.go",
		FileNamePrefix + FileNameSeparator + "8.go",
	}

	allRegistered, missing, extra, _ := dirRegistry.HasAllMigrationsRegistered()
	slices.Sort(missing)
	slices.Sort(extra)

	suite.Assert().False(allRegistered)
	suite.Assert().Equal(expectedMissing, missing)
	suite.Assert().Equal(expectedExtra, extra)
}

// Test migrations for auto-discovery testing
type TestMigrationWithDB struct {
	DB            string
	VersionNumber uint64
}

func (m *TestMigrationWithDB) Version() uint64 { return m.VersionNumber }
func (m *TestMigrationWithDB) Up() error       { return nil }
func (m *TestMigrationWithDB) Down() error     { return nil }

type TestMigrationWithContext struct {
	Ctx           context.Context
	VersionNumber uint64
}

func (m *TestMigrationWithContext) Version() uint64 { return m.VersionNumber }
func (m *TestMigrationWithContext) Up() error       { return nil }
func (m *TestMigrationWithContext) Down() error     { return nil }

type TestMigrationWithMultipleDeps struct {
	DB            string
	Ctx           context.Context
	VersionNumber uint64
}

func (m *TestMigrationWithMultipleDeps) Version() uint64 { return m.VersionNumber }
func (m *TestMigrationWithMultipleDeps) Up() error       { return nil }
func (m *TestMigrationWithMultipleDeps) Down() error     { return nil }

func (suite *RegistryTestSuite) TestAutoDiscoveryWithSingleDependency() {
	migDir, _ := NewMigrationsDirPath(suite.migrationsDirPath)

	// Create test migration files
	migFn := FileNamePrefix + FileNameSeparator + "1001.go"
	newFilePath := filepath.Join(suite.migrationsDirPath, migFn)
	fp, _ := os.OpenFile(newFilePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	_ = fp.Close()

	testDB := "test-database"

	registry := NewAutoDiscoveryDirMigrationsRegistry(
		migDir,
		func(migrationType reflect.Type) []reflect.Value {
			return []reflect.Value{
				reflect.ValueOf(testDB),
				reflect.ValueOf(uint64(1001)),
			}
		},
		&TestMigrationWithDB{},
	)

	suite.Assert().Equal(1, registry.Count())
	migration := registry.Get(1001)
	suite.Assert().NotNil(migration)
	suite.Assert().Equal(uint64(1001), migration.Version())

	// Verify dependency injection worked
	testMig, ok := migration.(*TestMigrationWithDB)
	suite.Assert().True(ok)
	suite.Assert().Equal(testDB, testMig.DB)
}

func (suite *RegistryTestSuite) TestAutoDiscoveryWithMultipleDependencies() {
	migDir, _ := NewMigrationsDirPath(suite.migrationsDirPath)

	// Create test migration files
	migFn := FileNamePrefix + FileNameSeparator + "1002.go"
	newFilePath := filepath.Join(suite.migrationsDirPath, migFn)
	fp, _ := os.OpenFile(newFilePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	_ = fp.Close()

	testDB := "test-database"
	testCtx := context.Background()

	registry := NewAutoDiscoveryDirMigrationsRegistry(
		migDir,
		func(migrationType reflect.Type) []reflect.Value {
			return []reflect.Value{
				reflect.ValueOf(testDB),
				reflect.ValueOf(testCtx),
				reflect.ValueOf(uint64(1002)),
			}
		},
		&TestMigrationWithMultipleDeps{},
	)

	suite.Assert().Equal(1, registry.Count())
	migration := registry.Get(1002)
	suite.Assert().NotNil(migration)
	suite.Assert().Equal(uint64(1002), migration.Version())

	// Verify dependency injection worked
	testMig, ok := migration.(*TestMigrationWithMultipleDeps)
	suite.Assert().True(ok)
	suite.Assert().Equal(testDB, testMig.DB)
	suite.Assert().Equal(testCtx, testMig.Ctx)
}

func (suite *RegistryTestSuite) TestAutoDiscoveryWithMultipleMigrationTypes() {
	migDir, _ := NewMigrationsDirPath(suite.migrationsDirPath)

	// Create test migration files
	for _, version := range []string{"1003", "1004"} {
		migFn := FileNamePrefix + FileNameSeparator + version + ".go"
		newFilePath := filepath.Join(suite.migrationsDirPath, migFn)
		fp, _ := os.OpenFile(newFilePath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
		_ = fp.Close()
	}

	testDB := "test-database"
	testCtx := context.Background()

	registry := NewAutoDiscoveryDirMigrationsRegistry(
		migDir,
		func(migrationType reflect.Type) []reflect.Value {
			// Provide different dependencies based on a migration type
			switch migrationType.Name() {
			case "TestMigrationWithDB":
				return []reflect.Value{
					reflect.ValueOf(testDB),
					reflect.ValueOf(uint64(1003)),
				}
			case "TestMigrationWithContext":
				return []reflect.Value{
					reflect.ValueOf(testCtx),
					reflect.ValueOf(uint64(1004)),
				}
			default:
				return []reflect.Value{}
			}
		},
		&TestMigrationWithDB{},
		&TestMigrationWithContext{},
	)

	suite.Assert().Equal(2, registry.Count())

	// Verify both migrations were discovered and configured correctly
	migration1 := registry.Get(1003)
	suite.Assert().NotNil(migration1)
	testMig1, ok := migration1.(*TestMigrationWithDB)
	suite.Assert().True(ok)
	suite.Assert().Equal(testDB, testMig1.DB)

	migration2 := registry.Get(1004)
	suite.Assert().NotNil(migration2)
	testMig2, ok := migration2.(*TestMigrationWithContext)
	suite.Assert().True(ok)
	suite.Assert().Equal(testCtx, testMig2.Ctx)
}
