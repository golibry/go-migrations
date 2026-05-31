// Package handler provides functionality for executing migrations and managing their execution state.
//
// This package defines the MigrationsHandler, which is the core service for running migrations,
// and the ExecutionPlan, which determines what migrations need to be executed. It also includes
// utilities for handling user input and managing migration execution state.
//
// The handler package is typically used by client code that acts as a user entrypoint,
// such as a CLI application or a web server.
package handler

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/golibry/go-migrations/execution"
	"github.com/golibry/go-migrations/migration"
)

// ExecutedMigration represents a migration and its execution state.
// It combines a Migration (the code to be executed) with a MigrationExecution
// (the record of when it was executed and whether it completed).
type ExecutedMigration struct {
	// Migration is the migration that was executed
	Migration migration.Migration

	// Execution contains information about when the migration was executed
	// and whether it completed successfully. It may be nil if the migration
	// has not been executed yet.
	Execution *execution.MigrationExecution
}

// ExecutionPlan determines which migrations need to be executed and in what order.
// It maintains the state of all registered migrations and their execution status,
// and provides methods to query this state.
type ExecutionPlan struct {
	// orderedMigrations contains all registered migrations in order of their version numbers
	orderedMigrations []migration.Migration

	// orderedExecutions contains all executed migrations in order of their version numbers
	orderedExecutions []execution.MigrationExecution
}

// NewPlan Creates a new ExecutionPlan. Errors if it finds that migrations and executions
// loaded from the provided registry & repository are in an inconsistent state. An inconsistent
// state can be: more executions in the repository than the total number of registered
// migrations
func NewPlan(
	registry migration.MigrationsRegistry,
	repository execution.Repository,
) (*ExecutionPlan, error) {
	genericErrMsg := "failed to create new execution plan"
	errHelpMsg := "Fix executions issues before trying to manipulate their state"

	executions, err := repository.LoadExecutions()
	if err != nil {
		return nil, fmt.Errorf(
			"%s, failed to load executions with error: %w. %s", genericErrMsg, err, errHelpMsg,
		)
	}

	sort.Slice(
		executions, func(i, j int) bool {
			return executions[i].Version < executions[j].Version
		},
	)

	plan := &ExecutionPlan{
		orderedMigrations: registry.OrderedMigrations(),
		orderedExecutions: executions,
	}

	if len(plan.orderedExecutions) > len(plan.orderedMigrations) {
		return nil, fmt.Errorf(
			"%s, there are more executions than registered migrations. %s",
			genericErrMsg, errHelpMsg,
		)
	}

	for i, exec := range plan.orderedExecutions {
		if !exec.Finished() && i != len(plan.orderedExecutions)-1 {
			return nil, fmt.Errorf(
				"%s, there are multiple executions which are not finished."+
					" Only the last execution should have an \"unfinished\" state. %s",
				genericErrMsg, errHelpMsg,
			)
		}

		if exec.Version != plan.orderedMigrations[i].Version() {
			return nil, fmt.Errorf(
				"%s, execution %d at index %d does not match with registered migration"+
					" %d at index %d. Migrations and executions are out of order. %s",
				genericErrMsg, exec.Version, i, plan.orderedMigrations[i].Version(), i, errHelpMsg,
			)
		}
	}

	return plan, err
}

func (plan *ExecutionPlan) RegisteredMigrationsCount() int {
	return len(plan.orderedMigrations)
}

func (plan *ExecutionPlan) FinishedExecutionsCount() int {
	count := len(plan.orderedExecutions)
	if count > 0 && !plan.orderedExecutions[count-1].Finished() {
		count--
	}
	return count
}

func (plan *ExecutionPlan) AllToBeExecuted() []migration.Migration {
	finishedExecCount := plan.FinishedExecutionsCount()

	if finishedExecCount < plan.RegisteredMigrationsCount() {
		return plan.orderedMigrations[finishedExecCount:]
	}

	return []migration.Migration{}
}

func (plan *ExecutionPlan) AllExecuted() []ExecutedMigration {
	var execMigrations []ExecutedMigration

	for i, exec := range plan.orderedExecutions {
		execMigrations = append(
			execMigrations, ExecutedMigration{
				Migration: plan.orderedMigrations[i],
				Execution: &exec,
			},
		)
	}

	return execMigrations
}

func (plan *ExecutionPlan) NextToExecute() migration.Migration {
	allToBeExec := plan.AllToBeExecuted()

	if len(allToBeExec) > 0 {
		return allToBeExec[0]
	}

	return nil
}

func (plan *ExecutionPlan) LastExecuted() ExecutedMigration {
	allExec := plan.AllExecuted()

	if len(allExec) > 0 {
		return allExec[len(allExec)-1]
	}

	return ExecutedMigration{}
}

func (plan *ExecutionPlan) Dirty() bool {
	return plan.UnfinishedExecution().Execution != nil
}

func (plan *ExecutionPlan) UnfinishedExecution() ExecutedMigration {
	if len(plan.orderedExecutions) == 0 {
		return ExecutedMigration{}
	}

	lastExecution := plan.orderedExecutions[len(plan.orderedExecutions)-1]
	if lastExecution.Finished() {
		return ExecutedMigration{}
	}

	return ExecutedMigration{
		Migration: plan.orderedMigrations[len(plan.orderedExecutions)-1],
		Execution: &lastExecution,
	}
}

type ExecutionPlanBuilder func(
	registry migration.MigrationsRegistry,
	repository execution.Repository,
) (*ExecutionPlan, error)

// MigrationsHandler A service which handles all migration related requests. Core service which
// should include all behaviour related to running the migrations
type MigrationsHandler struct {
	registry         migration.MigrationsRegistry
	repository       execution.Repository
	newExecutionPlan ExecutionPlanBuilder
	db               any
}

func NewHandler(
	registry migration.MigrationsRegistry,
	repository execution.Repository,
	newExecutionPlan ExecutionPlanBuilder,
) (*MigrationsHandler, error) {
	return NewHandlerWithDB(registry, repository, newExecutionPlan, nil)
}

func NewHandlerWithDB(
	registry migration.MigrationsRegistry,
	repository execution.Repository,
	newExecutionPlan ExecutionPlanBuilder,
	db any,
) (*MigrationsHandler, error) {
	err := repository.Init()

	if err != nil {
		return nil, fmt.Errorf(
			"could not create new migrations handler,"+
				" failed to initialize the repository with error: %w", err,
		)
	}

	if newExecutionPlan == nil {
		newExecutionPlan = NewPlan
	}

	return &MigrationsHandler{
		registry:         registry,
		repository:       repository,
		newExecutionPlan: newExecutionPlan,
		db:               db,
	}, nil
}

// NumOfRuns Type which is used to process the allowed user input for specifying the number
// of migrations to run
type NumOfRuns int

func NewNumOfRuns(num string) (NumOfRuns, error) {
	var parsedNum int

	if num == "all" {
		num = "99999"
	} else if len(strings.TrimSpace(num)) == 0 {
		num = "1"
	}

	parsedNum, err := strconv.Atoi(num)

	if err != nil {
		return NumOfRuns(1), errors.New(
			"failed to build new num of runs from provided string." +
				" Accepted values: integer number or \"all\"",
		)
	}

	if parsedNum <= 0 {
		parsedNum = 1
	}
	return NumOfRuns(parsedNum), nil
}

func (handler *MigrationsHandler) MigrateUp(
	ctx context.Context,
	numOfRuns NumOfRuns,
) ([]ExecutedMigration, error) {
	return withRepositoryLock(handler, func() ([]ExecutedMigration, error) {
		return handler.migrateUp(ctx, numOfRuns)
	})
}

func (handler *MigrationsHandler) migrateUp(
	ctx context.Context,
	numOfRuns NumOfRuns,
) ([]ExecutedMigration, error) {
	if handler.registry.Count() == 0 {
		return []ExecutedMigration{}, nil
	}

	errMsg := "failed to migrate all up"

	plan, err := handler.newExecutionPlan(handler.registry, handler.repository)
	if err != nil {
		return []ExecutedMigration{}, fmt.Errorf(
			"%s, failed to create execution plan with error: %w", errMsg, err,
		)
	}

	allToBeExec := plan.AllToBeExecuted()
	actualNumOfRuns := min(len(allToBeExec), int(numOfRuns))

	var handledMigrations []ExecutedMigration
	for i := 0; i < actualNumOfRuns; i++ {
		migrationToExec := allToBeExec[i]
		exec := execution.StartExecution(migrationToExec)
		if err = handler.repository.Save(*exec); err != nil {
			return handledMigrations, fmt.Errorf("%s, failed to save started execution: %w", errMsg, err)
		}

		if err = migrationToExec.Up(ctx, handler.db); err == nil {
			exec.FinishExecution()
		}

		handledMigrations = append(handledMigrations, ExecutedMigration{migrationToExec, exec})
		saveErr := error(nil)
		if exec.Finished() {
			saveErr = handler.repository.Save(*exec)
		}

		if err != nil {
			err = fmt.Errorf("%s, migration up failed: %w", errMsg, err)
			break
		}
		if saveErr != nil {
			err = fmt.Errorf("%s, failed to save finished execution: %w", errMsg, saveErr)
			break
		}
	}

	return handledMigrations, err
}

func (handler *MigrationsHandler) MigrateDown(
	ctx context.Context,
	numOfRuns NumOfRuns,
) ([]ExecutedMigration, error) {
	return withRepositoryLock(handler, func() ([]ExecutedMigration, error) {
		return handler.migrateDown(ctx, numOfRuns)
	})
}

func (handler *MigrationsHandler) migrateDown(
	ctx context.Context,
	numOfRuns NumOfRuns,
) ([]ExecutedMigration, error) {
	errMsg := "failed to migrate all down"

	plan, err := handler.newExecutionPlan(handler.registry, handler.repository)
	if err != nil {
		return []ExecutedMigration{}, fmt.Errorf(
			"%s, failed to create execution plan with error: %w", errMsg, err,
		)
	}

	execMigrations := plan.AllExecuted()
	slices.Reverse(execMigrations)
	actualNumOfRuns := min(len(execMigrations), int(numOfRuns))

	var handledMigrations []ExecutedMigration
	for i := 0; i < actualNumOfRuns; i++ {
		execMig := execMigrations[i]
		if err = execMig.Migration.Down(ctx, handler.db); err != nil {
			handledMigrations = append(handledMigrations, ExecutedMigration{execMig.Migration, nil})
			break
		}

		err = handler.repository.Remove(*execMig.Execution)

		if err != nil {
			handledMigrations = append(handledMigrations, ExecutedMigration{execMig.Migration, nil})
			break
		}

		handledMigrations = append(handledMigrations, execMig)
	}

	return handledMigrations, err
}

func (handler *MigrationsHandler) ForceUp(ctx context.Context, version uint64) (
	ExecutedMigration,
	error,
) {
	return withRepositoryLock(handler, func() (ExecutedMigration, error) {
		return handler.forceUp(ctx, version)
	})
}

func (handler *MigrationsHandler) forceUp(ctx context.Context, version uint64) (
	ExecutedMigration,
	error,
) {
	migrationToExec := handler.registry.Get(version)
	if migrationToExec == nil {
		return ExecutedMigration{nil, nil}, nil
	}

	exec := execution.StartExecution(migrationToExec)
	if err := handler.repository.Save(*exec); err != nil {
		return ExecutedMigration{migrationToExec, exec}, err
	}

	err := migrationToExec.Up(ctx, handler.db)
	if err == nil {
		exec.FinishExecution()
	}

	errSave := error(nil)
	if exec.Finished() {
		errSave = handler.repository.Save(*exec)
	}

	if err == nil {
		err = errSave
	} else if errSave != nil {
		err = fmt.Errorf("%w, %w", err, errSave)
	}

	return ExecutedMigration{migrationToExec, exec}, err
}

func (handler *MigrationsHandler) ForceDown(ctx context.Context, version uint64) (
	ExecutedMigration,
	error,
) {
	return withRepositoryLock(handler, func() (ExecutedMigration, error) {
		return handler.forceDown(ctx, version)
	})
}

func (handler *MigrationsHandler) forceDown(ctx context.Context, version uint64) (
	ExecutedMigration,
	error,
) {
	errMsg := "failed to migrate down forcefully"

	migrationToExec := handler.registry.Get(version)
	if migrationToExec == nil {
		return ExecutedMigration{nil, nil}, nil
	}

	exec, err := handler.repository.FindOne(version)
	if err != nil {
		return ExecutedMigration{migrationToExec, nil}, fmt.Errorf(
			"%s, failed to load execution with error: %w", errMsg, err,
		)
	}

	if exec == nil {
		return ExecutedMigration{migrationToExec, nil}, fmt.Errorf(
			"%s, execution not found. Maybe the migration was never executed", errMsg,
		)
	}

	if errDown := migrationToExec.Down(ctx, handler.db); errDown != nil {
		return ExecutedMigration{migrationToExec, nil}, fmt.Errorf(
			"%s, down() failed with error: %w", errMsg, errDown,
		)
	}

	err = handler.repository.Remove(*exec)

	return ExecutedMigration{migrationToExec, exec}, err
}

func (handler *MigrationsHandler) ForceFinish(version uint64) (*execution.MigrationExecution, error) {
	return withRepositoryLock(handler, func() (*execution.MigrationExecution, error) {
		exec, err := handler.repository.FindOne(version)
		if err != nil {
			return nil, err
		}
		if exec == nil {
			return nil, nil
		}

		exec.FinishExecution()
		if err = handler.repository.Save(*exec); err != nil {
			return nil, err
		}

		return exec, nil
	})
}

func (handler *MigrationsHandler) ForceRemove(version uint64) (*execution.MigrationExecution, error) {
	return withRepositoryLock(handler, func() (*execution.MigrationExecution, error) {
		exec, err := handler.repository.FindOne(version)
		if err != nil {
			return nil, err
		}
		if exec == nil {
			return nil, nil
		}

		if err = handler.repository.Remove(*exec); err != nil {
			return nil, err
		}

		return exec, nil
	})
}

func withRepositoryLock[T any](handler *MigrationsHandler, fn func() (T, error)) (T, error) {
	locker, ok := handler.repository.(execution.RepositoryLocker)
	if !ok {
		return fn()
	}

	unlock, err := locker.Lock()
	var zero T
	if err != nil {
		return zero, err
	}
	defer func() {
		_ = unlock()
	}()

	return fn()
}
