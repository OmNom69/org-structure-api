package service

import (
	"context"
	"errors"
	"testing"

	"github.com/OmNom69/org-structure-api/internal/model"
)

type fakeDepartmentRepository struct {
	DepartmentRepository

	department *model.Department
	err        error
	receivedID uint
}

type fakeEmployeeRepository struct {
	EmployeeRepository

	employee        *model.Employee
	employees       []model.Employee
	createdEmployee *model.Employee
	err             error
	receivedID      uint
}

// Department

func (f *fakeDepartmentRepository) GetByID(ctx context.Context, id uint) (*model.Department, error) {
	f.receivedID = id

	return f.department, f.err
}

// Employee

func (f *fakeEmployeeRepository) GetByID(ctx context.Context, id uint) (*model.Employee, error) {
	f.receivedID = id

	return f.employee, f.err
}

func (f *fakeEmployeeRepository) List(ctx context.Context) ([]model.Employee, error) {

	return f.employees, f.err
}

func (f *fakeEmployeeRepository) Create(ctx context.Context, employee *model.Employee) error {
	f.createdEmployee = employee
	employee.ID = 10

	return f.err
}

func TestEmployeeService_GetEmployee_InvalidID(t *testing.T) {
	employeeService := NewEmployeeService(nil, nil, nil)

	employee, err := employeeService.GetEmployee(context.Background(), 0)

	if employee != nil {
		t.Fatalf("expected nil employee, got %+v", employee)
	}

	if !errors.Is(err, ErrInvalidEmployeeID) {
		t.Fatalf("expected ErrInvalidEmployeeID, got %v", err)
	}
}

func TestEmployeeService_GetEmployee_Success(t *testing.T) {
	expectedEmployee := &model.Employee{
		ID:           7,
		DepartmentID: 3,
		FullName:     "Иван Петров",
		Position:     "Backend Developer",
	}

	employeeRepo := &fakeEmployeeRepository{
		employee: expectedEmployee,
	}

	employeeService := NewEmployeeService(employeeRepo, nil, nil)

	employee, err := employeeService.GetEmployee(context.Background(), 7)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if employee != expectedEmployee {
		t.Fatalf("expected employee %+v, got %+v", expectedEmployee, employee)
	}

	if employeeRepo.receivedID != 7 {
		t.Fatalf("expected repository to receive ID 7, got %d", employeeRepo.receivedID)
	}
}

func TestEmployeeService_GetEmployees_Success(t *testing.T) {
	expectedEmployees := []model.Employee{
		{
			ID:       1,
			FullName: "Иван Петров",
			Position: "Backend Developer",
		},
		{
			ID:       2,
			FullName: "Анна Смирнова",
			Position: "Team Lead",
		},
	}

	employeeRepo := &fakeEmployeeRepository{
		employees: expectedEmployees,
	}

	employeeService := NewEmployeeService(employeeRepo, nil, nil)

	employees, err := employeeService.GetEmployees(context.Background())

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(employees) != 2 {
		t.Fatalf("expected 2 employees, got %d", len(employees))
	}

	if employees[0].ID != 1 {
		t.Fatalf("expected first employee ID 1, got %d", employees[0].ID)
	}

	if employees[1].ID != 2 {
		t.Fatalf("expected second employee ID 2, got %d", employees[1].ID)
	}
}

func TestEmployeeService_CreateEmployee_Success(t *testing.T) {
	employeeRepo := &fakeEmployeeRepository{}

	departmentRepo := &fakeDepartmentRepository{
		department: &model.Department{
			ID:   3,
			Name: "Backend",
		},
	}

	employeeService := NewEmployeeService(employeeRepo, departmentRepo, nil)

	employee, err := employeeService.CreateEmployee(
		context.Background(),
		CreateEmployeeInput{
			DepartmentID: 3,
			FullName:     "Иван Петров",
			Position:     "Backend Developer",
		},
	)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if departmentRepo.receivedID != 3 {
		t.Fatalf("expected department ID 3, got %d", departmentRepo.receivedID)
	}

	if employeeRepo.createdEmployee == nil {
		t.Fatal("expected employee repository Create to be called")
	}

	if employee.ID != 10 {
		t.Fatalf("expected employee ID 10, got %d", employee.ID)
	}

	if employee.FullName != "Иван Петров" {
		t.Fatalf("expected Иван Петров, got %s", employee.FullName)
	}
}
