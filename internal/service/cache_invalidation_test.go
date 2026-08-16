package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/OmNom69/org-structure-api/internal/model"
)

type mutationDepartmentRepositoryFake struct {
	DepartmentRepository

	events *[]string

	createErr            error
	updateErr            error
	deleteErr            error
	reassignAndDeleteErr error

	onCreateSuccess func()
}

func (f *mutationDepartmentRepositoryFake) Create(_ context.Context, department *model.Department) error {
	if f.createErr != nil {
		f.record("department.create.error")
		return f.createErr
	}

	department.ID = 101
	f.record("department.create.success")
	if f.onCreateSuccess != nil {
		f.onCreateSuccess()
	}

	return nil
}

func (f *mutationDepartmentRepositoryFake) GetByID(_ context.Context, id uint) (*model.Department, error) {
	return &model.Department{
		ID:   id,
		Name: "Existing Department",
	}, nil
}

func (f *mutationDepartmentRepositoryFake) Update(_ context.Context, _ *model.Department) error {
	if f.updateErr != nil {
		f.record("department.update.error")
		return f.updateErr
	}

	f.record("department.update.success")
	return nil
}

func (f *mutationDepartmentRepositoryFake) DeleteByID(_ context.Context, _ uint) error {
	if f.deleteErr != nil {
		f.record("department.delete.error")
		return f.deleteErr
	}

	f.record("department.delete.success")
	return nil
}

func (f *mutationDepartmentRepositoryFake) ReassignAndDelete(_ context.Context, _ uint, _ uint) error {
	if f.reassignAndDeleteErr != nil {
		f.record("department.reassign_and_delete.error")
		return f.reassignAndDeleteErr
	}

	f.record("department.reassign_and_delete.success")
	return nil
}

func (f *mutationDepartmentRepositoryFake) ExistsByNameAndParent(_ context.Context, _ string, _ *uint) (bool, error) {
	return false, nil
}

func (f *mutationDepartmentRepositoryFake) ExistsByNameAndParentExceptID(_ context.Context, _ string, _ *uint, _ uint) (bool, error) {
	return false, nil
}

func (f *mutationDepartmentRepositoryFake) WouldCreateCycle(_ context.Context, _ uint, _ uint) (bool, error) {
	return false, nil
}

func (f *mutationDepartmentRepositoryFake) record(event string) {
	*f.events = append(*f.events, event)
}

type mutationEmployeeRepositoryFake struct {
	EmployeeRepository

	events *[]string

	createErr error
	updateErr error
	deleteErr error
}

func (f *mutationEmployeeRepositoryFake) Create(_ context.Context, employee *model.Employee) error {
	if f.createErr != nil {
		f.record("employee.create.error")
		return f.createErr
	}

	employee.ID = 201
	f.record("employee.create.success")
	return nil
}

func (f *mutationEmployeeRepositoryFake) GetByID(_ context.Context, id uint) (*model.Employee, error) {
	return &model.Employee{
		ID:           id,
		DepartmentID: 1,
		FullName:     "Existing Employee",
		Position:     "Developer",
	}, nil
}

func (f *mutationEmployeeRepositoryFake) Update(_ context.Context, _ *model.Employee) error {
	if f.updateErr != nil {
		f.record("employee.update.error")
		return f.updateErr
	}

	f.record("employee.update.success")
	return nil
}

func (f *mutationEmployeeRepositoryFake) DeleteByID(_ context.Context, _ uint) error {
	if f.deleteErr != nil {
		f.record("employee.delete.error")
		return f.deleteErr
	}

	f.record("employee.delete.success")
	return nil
}

func (f *mutationEmployeeRepositoryFake) record(event string) {
	*f.events = append(*f.events, event)
}

type mutationCacheStoreFake struct {
	events *[]string

	incrementErr         error
	incrementKeys        []string
	incrementContextErrs []error
	getCalls             int
	setCalls             int
}

func (f *mutationCacheStoreFake) Get(_ context.Context, _ string) ([]byte, bool, error) {
	f.getCalls++
	return nil, false, nil
}

func (f *mutationCacheStoreFake) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	f.setCalls++
	return nil
}

func (f *mutationCacheStoreFake) Increment(ctx context.Context, key string) (int64, error) {
	f.incrementKeys = append(f.incrementKeys, key)
	f.incrementContextErrs = append(f.incrementContextErrs, ctx.Err())
	*f.events = append(*f.events, "cache.increment")

	if f.incrementErr != nil {
		return 0, f.incrementErr
	}

	return int64(len(f.incrementKeys)), nil
}

type mutationInvalidationOperation struct {
	name         string
	successEvent string
	failureEvent string
	run          func(
		context.Context,
		*mutationDepartmentRepositoryFake,
		*mutationEmployeeRepositoryFake,
		*mutationCacheStoreFake,
	) error
	setRepositoryError func(
		*mutationDepartmentRepositoryFake,
		*mutationEmployeeRepositoryFake,
		error,
	)
}

func TestMutationInvalidationAfterSuccessfulDatabaseWrite(t *testing.T) {
	for _, operation := range mutationInvalidationOperations() {
		t.Run(operation.name, func(t *testing.T) {
			departmentRepo, employeeRepo, cacheStore, events := newMutationInvalidationFakes()

			err := operation.run(
				context.Background(),
				departmentRepo,
				employeeRepo,
				cacheStore,
			)
			if err != nil {
				t.Fatalf("mutation error = %v", err)
			}

			assertMutationIncrement(t, cacheStore, 1)
			assertMutationEvents(
				t,
				*events,
				[]string{operation.successEvent, "cache.increment"},
			)
		})
	}
}

func TestMutationInvalidationSkipsFailedDatabaseWrite(t *testing.T) {
	for _, operation := range mutationInvalidationOperations() {
		t.Run(operation.name, func(t *testing.T) {
			departmentRepo, employeeRepo, cacheStore, events := newMutationInvalidationFakes()
			repositoryErr := errors.New("repository write failed")
			operation.setRepositoryError(departmentRepo, employeeRepo, repositoryErr)

			err := operation.run(
				context.Background(),
				departmentRepo,
				employeeRepo,
				cacheStore,
			)
			if !errors.Is(err, repositoryErr) {
				t.Fatalf("mutation error = %v, want wrapped %v", err, repositoryErr)
			}

			assertMutationIncrement(t, cacheStore, 0)
			assertMutationEvents(t, *events, []string{operation.failureEvent})
		})
	}
}

func TestMutationInvalidationErrorDoesNotChangeDatabaseSuccess(t *testing.T) {
	for _, operation := range mutationInvalidationOperations() {
		t.Run(operation.name, func(t *testing.T) {
			departmentRepo, employeeRepo, cacheStore, events := newMutationInvalidationFakes()
			cacheStore.incrementErr = errors.New("redis unavailable")

			err := operation.run(
				context.Background(),
				departmentRepo,
				employeeRepo,
				cacheStore,
			)
			if err != nil {
				t.Fatalf("mutation error = %v, want database success", err)
			}

			assertMutationIncrement(t, cacheStore, 1)
			assertMutationEvents(
				t,
				*events,
				[]string{operation.successEvent, "cache.increment"},
			)
		})
	}
}

func TestMutationInvalidationUsesNonCanceledContextAfterDatabaseSuccess(t *testing.T) {
	departmentRepo, employeeRepo, cacheStore, events := newMutationInvalidationFakes()
	ctx, cancel := context.WithCancel(context.Background())
	departmentRepo.onCreateSuccess = cancel

	departmentService := NewDepartmentService(
		departmentRepo,
		employeeRepo,
		cacheStore,
		time.Minute,
	)

	_, err := departmentService.CreateDepartment(ctx, CreateDepartmentInput{Name: "Platform"})
	if err != nil {
		t.Fatalf("CreateDepartment() error = %v", err)
	}

	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatalf("original context error = %v, want context.Canceled", ctx.Err())
	}

	assertMutationIncrement(t, cacheStore, 1)
	if len(cacheStore.incrementContextErrs) != 1 || cacheStore.incrementContextErrs[0] != nil {
		t.Fatalf(
			"Increment() context errors = %v, want one non-canceled context",
			cacheStore.incrementContextErrs,
		)
	}

	assertMutationEvents(
		t,
		*events,
		[]string{"department.create.success", "cache.increment"},
	)
}

func mutationInvalidationOperations() []mutationInvalidationOperation {
	return []mutationInvalidationOperation{
		{
			name:         "create department",
			successEvent: "department.create.success",
			failureEvent: "department.create.error",
			run: func(
				ctx context.Context,
				departmentRepo *mutationDepartmentRepositoryFake,
				employeeRepo *mutationEmployeeRepositoryFake,
				cacheStore *mutationCacheStoreFake,
			) error {
				departmentService := NewDepartmentService(
					departmentRepo,
					employeeRepo,
					cacheStore,
					time.Minute,
				)
				_, err := departmentService.CreateDepartment(
					ctx,
					CreateDepartmentInput{Name: "Platform"},
				)
				return err
			},
			setRepositoryError: func(
				departmentRepo *mutationDepartmentRepositoryFake,
				_ *mutationEmployeeRepositoryFake,
				err error,
			) {
				departmentRepo.createErr = err
			},
		},
		{
			name:         "patch department",
			successEvent: "department.update.success",
			failureEvent: "department.update.error",
			run: func(
				ctx context.Context,
				departmentRepo *mutationDepartmentRepositoryFake,
				employeeRepo *mutationEmployeeRepositoryFake,
				cacheStore *mutationCacheStoreFake,
			) error {
				departmentService := NewDepartmentService(
					departmentRepo,
					employeeRepo,
					cacheStore,
					time.Minute,
				)
				name := "Renamed Department"
				_, err := departmentService.PatchDepartment(
					ctx,
					PatchDepartmentInput{ID: 1, Name: &name},
				)
				return err
			},
			setRepositoryError: func(
				departmentRepo *mutationDepartmentRepositoryFake,
				_ *mutationEmployeeRepositoryFake,
				err error,
			) {
				departmentRepo.updateErr = err
			},
		},
		{
			name:         "delete department with cascade",
			successEvent: "department.delete.success",
			failureEvent: "department.delete.error",
			run: func(
				ctx context.Context,
				departmentRepo *mutationDepartmentRepositoryFake,
				employeeRepo *mutationEmployeeRepositoryFake,
				cacheStore *mutationCacheStoreFake,
			) error {
				departmentService := NewDepartmentService(
					departmentRepo,
					employeeRepo,
					cacheStore,
					time.Minute,
				)
				_, err := departmentService.DeleteDepartment(
					ctx,
					DeleteDepartmentInput{ID: 1, Mode: "cascade"},
				)
				return err
			},
			setRepositoryError: func(
				departmentRepo *mutationDepartmentRepositoryFake,
				_ *mutationEmployeeRepositoryFake,
				err error,
			) {
				departmentRepo.deleteErr = err
			},
		},
		{
			name:         "delete department with reassignment",
			successEvent: "department.reassign_and_delete.success",
			failureEvent: "department.reassign_and_delete.error",
			run: func(
				ctx context.Context,
				departmentRepo *mutationDepartmentRepositoryFake,
				employeeRepo *mutationEmployeeRepositoryFake,
				cacheStore *mutationCacheStoreFake,
			) error {
				departmentService := NewDepartmentService(
					departmentRepo,
					employeeRepo,
					cacheStore,
					time.Minute,
				)
				targetID := uint(2)
				_, err := departmentService.DeleteDepartment(
					ctx,
					DeleteDepartmentInput{
						ID:                     1,
						Mode:                   "reassign",
						ReassignToDepartmentID: &targetID,
					},
				)
				return err
			},
			setRepositoryError: func(
				departmentRepo *mutationDepartmentRepositoryFake,
				_ *mutationEmployeeRepositoryFake,
				err error,
			) {
				departmentRepo.reassignAndDeleteErr = err
			},
		},
		{
			name:         "create employee",
			successEvent: "employee.create.success",
			failureEvent: "employee.create.error",
			run: func(
				ctx context.Context,
				departmentRepo *mutationDepartmentRepositoryFake,
				employeeRepo *mutationEmployeeRepositoryFake,
				cacheStore *mutationCacheStoreFake,
			) error {
				employeeService := NewEmployeeService(employeeRepo, departmentRepo, cacheStore)
				_, err := employeeService.CreateEmployee(
					ctx,
					CreateEmployeeInput{
						DepartmentID: 1,
						FullName:     "New Employee",
						Position:     "Developer",
					},
				)
				return err
			},
			setRepositoryError: func(
				_ *mutationDepartmentRepositoryFake,
				employeeRepo *mutationEmployeeRepositoryFake,
				err error,
			) {
				employeeRepo.createErr = err
			},
		},
		{
			name:         "patch employee",
			successEvent: "employee.update.success",
			failureEvent: "employee.update.error",
			run: func(
				ctx context.Context,
				departmentRepo *mutationDepartmentRepositoryFake,
				employeeRepo *mutationEmployeeRepositoryFake,
				cacheStore *mutationCacheStoreFake,
			) error {
				employeeService := NewEmployeeService(employeeRepo, departmentRepo, cacheStore)
				position := "Team Lead"
				_, err := employeeService.PatchEmployee(
					ctx,
					PatchEmployeeInput{
						ID:          1,
						Position:    &position,
						PositionSet: true,
					},
				)
				return err
			},
			setRepositoryError: func(
				_ *mutationDepartmentRepositoryFake,
				employeeRepo *mutationEmployeeRepositoryFake,
				err error,
			) {
				employeeRepo.updateErr = err
			},
		},
		{
			name:         "delete employee",
			successEvent: "employee.delete.success",
			failureEvent: "employee.delete.error",
			run: func(
				ctx context.Context,
				departmentRepo *mutationDepartmentRepositoryFake,
				employeeRepo *mutationEmployeeRepositoryFake,
				cacheStore *mutationCacheStoreFake,
			) error {
				employeeService := NewEmployeeService(employeeRepo, departmentRepo, cacheStore)
				return employeeService.DeleteEmployee(ctx, 1)
			},
			setRepositoryError: func(
				_ *mutationDepartmentRepositoryFake,
				employeeRepo *mutationEmployeeRepositoryFake,
				err error,
			) {
				employeeRepo.deleteErr = err
			},
		},
	}
}

func newMutationInvalidationFakes() (
	*mutationDepartmentRepositoryFake,
	*mutationEmployeeRepositoryFake,
	*mutationCacheStoreFake,
	*[]string,
) {
	events := []string{}

	return &mutationDepartmentRepositoryFake{events: &events},
		&mutationEmployeeRepositoryFake{events: &events},
		&mutationCacheStoreFake{events: &events},
		&events
}

func assertMutationIncrement(t *testing.T, cacheStore *mutationCacheStoreFake, wantCalls int) {
	t.Helper()

	if len(cacheStore.incrementKeys) != wantCalls {
		t.Fatalf(
			"Increment() calls = %d, want %d; keys = %v",
			len(cacheStore.incrementKeys),
			wantCalls,
			cacheStore.incrementKeys,
		)
	}

	for _, key := range cacheStore.incrementKeys {
		if key != departmentTreeEpochKey {
			t.Fatalf("Increment() key = %q, want %q", key, departmentTreeEpochKey)
		}
	}

	if cacheStore.getCalls != 0 || cacheStore.setCalls != 0 {
		t.Fatalf(
			"mutation cache calls: Get=%d Set=%d, want both zero",
			cacheStore.getCalls,
			cacheStore.setCalls,
		)
	}
}

func assertMutationEvents(t *testing.T, got, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mutation events = %v, want %v", got, want)
	}
}
