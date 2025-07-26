package main

import (
	"context"
	"fmt"
	"reflect"
	"github.com/golibry/go-migrations/migration"
)

// Example migrations that demonstrate auto-discovery
type Migration1712953077 struct {
	DB  string
	Ctx context.Context
}

func (m *Migration1712953077) Version() uint64 { return 1712953077 }
func (m *Migration1712953077) Up() error {
	fmt.Printf("Running migration %d with DB: %s\n", m.Version(), m.DB)
	return nil
}
func (m *Migration1712953077) Down() error {
	fmt.Printf("Rolling back migration %d\n", m.Version())
	return nil
}

type Migration1712953080 struct {
	DB string
}

func (m *Migration1712953080) Version() uint64 { return 1712953080 }
func (m *Migration1712953080) Up() error {
	fmt.Printf("Running migration %d with DB: %s\n", m.Version(), m.DB)
	return nil
}
func (m *Migration1712953080) Down() error {
	fmt.Printf("Rolling back migration %d\n", m.Version())
	return nil
}

type Migration1712953083 struct {
	DB  string
	Ctx context.Context
}

func (m *Migration1712953083) Version() uint64 { return 1712953083 }
func (m *Migration1712953083) Up() error {
	fmt.Printf("Running migration %d with DB: %s\n", m.Version(), m.DB)
	return nil
}
func (m *Migration1712953083) Down() error {
	fmt.Printf("Rolling back migration %d\n", m.Version())
	return nil
}

func main() {
	fmt.Println("=== Auto-Discovery Migration Example ===")
	
	// Note: For this demo, we don't need actual migration files on disk
	// The auto-discovery works with the migration types we define in code
	
	// Simulate database connection and context
	db := "postgresql://localhost:5432/mydb"
	ctx := context.Background()
	
	fmt.Println("\n=== Manual Registration (Old Way) ===")
	manualRegistry := migration.NewGenericRegistry()
	
	// Manual registration - tedious and error-prone
	manualRegistry.Register(&Migration1712953077{DB: db, Ctx: ctx})
	manualRegistry.Register(&Migration1712953080{DB: db})
	manualRegistry.Register(&Migration1712953083{DB: db, Ctx: ctx})
	
	fmt.Printf("Manually registered %d migrations\n", manualRegistry.Count())
	
	fmt.Println("\n=== Auto-Discovery (New Way) ===")
	
	// For this demo, we'll use the simpler auto-discovery with GenericRegistry
	// to avoid directory validation complexity
	config := &migration.AutoDiscoveryConfig{
		PackageTypes: []interface{}{
			&Migration1712953077{},
			&Migration1712953080{},
			&Migration1712953083{},
		},
		DependencyProvider: func(migrationType reflect.Type) []reflect.Value {
			// Provide dependencies based on what each migration type needs
			dependencies := []reflect.Value{reflect.ValueOf(db)}
			
			// Check if the migration needs context
			for i := 0; i < migrationType.NumField(); i++ {
				field := migrationType.Field(i)
				if field.Name == "Ctx" && field.Type.String() == "context.Context" {
					dependencies = append(dependencies, reflect.ValueOf(ctx))
					break
				}
			}
			
			return dependencies
		},
	}
	
	// Discover migrations using the config
	discoveredMigrations := migration.DiscoverMigrations(config)
	autoRegistry := migration.NewGenericRegistry()
	for _, mig := range discoveredMigrations {
		autoRegistry.Register(mig)
	}
	
	fmt.Printf("Auto-discovered %d migrations\n", autoRegistry.Count())
	
	fmt.Println("\n=== Running Discovered Migrations ===")
	for _, migration := range autoRegistry.OrderedMigrations() {
		fmt.Printf("Executing migration %d...\n", migration.Version())
		if err := migration.Up(); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
	
	fmt.Println("\n=== Benefits of Auto-Discovery ===")
	fmt.Println("✓ No need to manually register each migration")
	fmt.Println("✓ Automatic dependency injection based on struct fields")
	fmt.Println("✓ Type-safe migration discovery using reflection")
	fmt.Println("✓ Reduces boilerplate code and human error")
	fmt.Println("✓ Easy to add new migrations - just create the struct!")
}