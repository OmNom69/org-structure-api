package service

import (
	"time"

	"github.com/OmNom69/org-structure-api/internal/model"
	"github.com/OmNom69/org-structure-api/internal/repository"
	"github.com/OmNom69/org-structure-api/internal/validator"
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
