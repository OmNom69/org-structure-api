package service

import (
	"context"
	"time"

	"errors"

	"github.com/OmNom69/org-structure-api/internal/model"
	"github.com/OmNom69/org-structure-api/internal/validator"
)

type EmployeeService struct {
	employeeRepo   EmployeeRepository
	departmentRepo DepartmentRepository
}

func NewEmployeeService(
	employeeRepo EmployeeRepository,
	departmentRepo DepartmentRepository,
) *EmployeeService {
	return &EmployeeService{
		employeeRepo:   employeeRepo,
		departmentRepo: departmentRepo,
	}
}

type CreateEmployeeInput struct {
	DepartmentID uint
	FullName     string
	Position     string
	HiredAt      *string
}

type PatchEmployeeInput struct {
	ID uint

	FullName    *string
	FullNameSet bool

	Position    *string
	PositionSet bool

	DepartmentID    *uint
	DepartmentIDSet bool

	HiredAt    *string
	HiredAtSet bool
}

// create

func (s *EmployeeService) CreateEmployee(ctx context.Context, input CreateEmployeeInput) (*model.Employee, error) {
	if input.DepartmentID == 0 {
		return nil, ErrInvalidDepartmentID
	}

	fullName, err := validator.RequiredString(input.FullName, "full_name")
	if err != nil {
		return nil, wrapValidationError(err)
	}

	position, err := validator.RequiredString(input.Position, "position")
	if err != nil {
		return nil, wrapValidationError(err)
	}

	var hiredAt *time.Time

	if input.HiredAt != nil {
		parsedHiredAt, err := time.Parse("2006-01-02", *input.HiredAt)
		if err != nil {
			return nil, wrapInvalidHiredAtError(err)
		}

		hiredAt = &parsedHiredAt
	}

	if _, err := s.departmentRepo.GetByID(ctx, input.DepartmentID); err != nil {
		return nil, mapStorageError(err, ErrDepartmentNotFound)
	}

	employee := model.Employee{
		DepartmentID: input.DepartmentID,
		FullName:     fullName,
		Position:     position,
		HiredAt:      hiredAt,
	}

	if err := s.employeeRepo.Create(ctx, &employee); err != nil {
		return nil, err
	}

	return &employee, nil
}

// get all employees

func (s *EmployeeService) GetEmployees(ctx context.Context) ([]model.Employee, error) {
	return s.employeeRepo.GetAllEmployees(ctx)
}

// get employee by ID

func (s *EmployeeService) GetEmployee(ctx context.Context, id uint) (*model.Employee, error) {
	if id == 0 {
		return nil, ErrInvalidEmployeeID
	}

	employee, err := s.employeeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, mapStorageError(err, ErrEmployeeNotFound)
	}

	return employee, nil
}

// delete

func (s *EmployeeService) DeleteEmployee(ctx context.Context, id uint) error {
	if _, err := s.GetEmployee(ctx, id); err != nil {
		return err
	}

	if err := s.employeeRepo.DeleteByID(ctx, id); err != nil {
		return err
	}

	return nil
}

// patch

func (s *EmployeeService) PatchEmployee(ctx context.Context, input PatchEmployeeInput) (*model.Employee, error) {
	if input.ID == 0 {
		return nil, ErrInvalidEmployeeID
	}

	if !input.FullNameSet &&
		!input.PositionSet &&
		!input.DepartmentIDSet &&
		!input.HiredAtSet {
		return nil, ErrNothingToUpdate
	}

	employee, err := s.GetEmployee(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if input.FullNameSet {
		if input.FullName == nil {
			return nil, wrapValidationError(errors.New("full_name cannot be null"))
		}

		fullName, err := validator.RequiredString(*input.FullName, "full_name")
		if err != nil {
			return nil, wrapValidationError(err)
		}

		employee.FullName = fullName
	}

	if input.PositionSet {
		if input.Position == nil {
			return nil, wrapValidationError(errors.New("position cannot be null"))
		}

		position, err := validator.RequiredString(*input.Position, "position")
		if err != nil {
			return nil, wrapValidationError(err)
		}

		employee.Position = position
	}

	if input.DepartmentIDSet {
		if input.DepartmentID == nil {
			return nil, wrapValidationError(errors.New("department_id cannot be null"))
		}

		if *input.DepartmentID == 0 {
			return nil, ErrInvalidDepartmentID
		}

		if _, err := s.departmentRepo.GetByID(ctx, *input.DepartmentID); err != nil {
			return nil, mapStorageError(err, ErrDepartmentNotFound)
		}

		employee.DepartmentID = *input.DepartmentID
	}

	if input.HiredAtSet {
		if input.HiredAt == nil {
			employee.HiredAt = nil
		} else {
			hiredAt, err := time.Parse("2006-01-02", *input.HiredAt)
			if err != nil {
				return nil, wrapInvalidHiredAtError(err)
			}

			employee.HiredAt = &hiredAt
		}
	}

	if err := s.employeeRepo.Update(ctx, employee); err != nil {
		return nil, err
	}

	return employee, nil
}
