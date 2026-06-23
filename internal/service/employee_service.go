package service

import (
	"errors"
	"time"

	"github.com/OmNom69/org-structure-api/internal/model"
	"github.com/OmNom69/org-structure-api/internal/repository"
	"github.com/OmNom69/org-structure-api/internal/validator"
)

var (
	ErrNothingToUpdate     = errors.New("nothing to update")
	ErrInvalidDepartmentID = errors.New("invalid department id")
)

type EmployeeService struct {
	employeeRepo   *repository.EmployeeRepository
	departmentRepo *repository.DepartmentRepository
}

func NewEmployeeService(
	employeeRepo *repository.EmployeeRepository,
	departmentRepo *repository.DepartmentRepository,
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
	ID           uint
	FullName     *string
	Position     *string
	DepartmentID *uint
	HiredAt      *string
}

// create

func (s *EmployeeService) CreateEmployee(input CreateEmployeeInput) (*model.Employee, error) {
	if _, err := s.departmentRepo.GetByID(input.DepartmentID); err != nil {
		return nil, err
	}

	fullName, err := validator.RequiredString(input.FullName, "full_name")
	if err != nil {
		return nil, err
	}

	position, err := validator.RequiredString(input.Position, "position")
	if err != nil {
		return nil, err
	}

	var hiredAt *time.Time

	if input.HiredAt != nil {
		parsedHiredAt, err := time.Parse("2006-01-02", *input.HiredAt)
		if err != nil {
			return nil, err
		}

		hiredAt = &parsedHiredAt
	}

	employee := model.Employee{
		DepartmentID: input.DepartmentID,
		FullName:     fullName,
		Position:     position,
		HiredAt:      hiredAt,
	}

	if err := s.employeeRepo.Create(&employee); err != nil {
		return nil, err
	}

	return &employee, nil
}

// get all employees

func (s *EmployeeService) GetEmployees() ([]model.Employee, error) {
	return s.employeeRepo.GetAllEmployees()
}

// get employee by ID

func (s *EmployeeService) GetEmployee(id uint) (*model.Employee, error) {
	return s.employeeRepo.GetByID(id)
}

// delete

func (s *EmployeeService) DeleteEmployee(id uint) error {
	if _, err := s.employeeRepo.GetByID(id); err != nil {
		return err
	}

	if err := s.employeeRepo.DeleteByID(id); err != nil {
		return err
	}

	return nil
}

// patch

func (s *EmployeeService) PatchEmployee(input PatchEmployeeInput) (*model.Employee, error) {
	employee, err := s.employeeRepo.GetByID(input.ID)
	if err != nil {
		return nil, err
	}

	if input.FullName == nil &&
		input.Position == nil &&
		input.DepartmentID == nil &&
		input.HiredAt == nil {
		return nil, ErrNothingToUpdate
	}

	if input.FullName != nil {
		fullName, err := validator.RequiredString(*input.FullName, "full_name")
		if err != nil {
			return nil, err
		}

		employee.FullName = fullName
	}

	if input.Position != nil {
		position, err := validator.RequiredString(*input.Position, "position")
		if err != nil {
			return nil, err
		}

		employee.Position = position
	}

	if input.DepartmentID != nil {
		if *input.DepartmentID == 0 {
			return nil, ErrInvalidDepartmentID
		}

		if _, err := s.departmentRepo.GetByID(*input.DepartmentID); err != nil {
			return nil, err
		}

		employee.DepartmentID = *input.DepartmentID
	}

	if input.HiredAt != nil {
		hiredAt, err := time.Parse("2006-01-02", *input.HiredAt)
		if err != nil {
			return nil, err
		}

		employee.HiredAt = &hiredAt
	}

	if err := s.employeeRepo.Update(employee); err != nil {
		return nil, err
	}

	return employee, nil
}
