package service

import (
	"github.com/OmNom69/org-structure-api/internal/model"
	"github.com/OmNom69/org-structure-api/internal/repository"
	"github.com/OmNom69/org-structure-api/internal/validator"
)

type DepartmentService struct {
	departmentRepo *repository.DepartmentRepository
	employeeRepo   *repository.EmployeeRepository
}

func NewDepartmentService(
	departmentRepo *repository.DepartmentRepository,
	employeeRepo *repository.EmployeeRepository,
) *DepartmentService {
	return &DepartmentService{
		departmentRepo: departmentRepo,
		employeeRepo:   employeeRepo,
	}
}

type CreateDepartmentInput struct {
	Name     string
	ParentID *uint
}

// create department

func (s *DepartmentService) CreateDepartment(input CreateDepartmentInput) (*model.Department, error) {
	name, err := validator.RequiredString(input.Name, "name")
	if err != nil {
		return nil, err
	}

	if input.ParentID != nil {
		if *input.ParentID == 0 {
			return nil, ErrInvalidDepartmentID
		}

		if _, err := s.departmentRepo.GetByID(*input.ParentID); err != nil {
			return nil, err
		}
	}

	exists, err := s.departmentRepo.ExistsByNameAndParent(name, input.ParentID)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, ErrDepartmentAlreadyExists
	}

	department := model.Department{
		Name:     name,
		ParentID: input.ParentID,
	}

	if err := s.departmentRepo.Create(&department); err != nil {
		return nil, err
	}

	return &department, nil
}
