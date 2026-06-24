package service

import (
	"github.com/OmNom69/org-structure-api/internal/dto"
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

type DeleteDepartmentInput struct {
	ID                     uint
	Mode                   string
	ReassignToDepartmentID *uint
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

// get department tree

func (s *DepartmentService) GetDepartmentTree(
	id uint,
	depth int,
	includeEmployees bool,
) (*dto.DepartmentTreeResponse, error) {
	if depth < 1 || depth > 5 {
		return nil, ErrInvalidDepth
	}

	department, err := s.departmentRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	tree, err := s.buildDepartmentTree(department, depth, includeEmployees)
	if err != nil {
		return nil, err
	}

	return &tree, nil
}

// helper func

func (s *DepartmentService) buildDepartmentTree(
	department *model.Department,
	depth int,
	includeEmployees bool,
) (dto.DepartmentTreeResponse, error) {
	response := dto.DepartmentTreeResponse{
		ID:        department.ID,
		Name:      department.Name,
		ParentID:  department.ParentID,
		CreatedAt: department.CreatedAt,
		Children:  []dto.DepartmentTreeResponse{},
	}

	if includeEmployees {
		employees, err := s.employeeRepo.GetEmployeesForTree(department.ID)
		if err != nil {
			return dto.DepartmentTreeResponse{}, err
		}

		response.Employees = employees
	}

	if depth <= 0 {
		return response, nil
	}

	children, err := s.departmentRepo.GetChildren(department.ID)
	if err != nil {
		return dto.DepartmentTreeResponse{}, err
	}

	for _, child := range children {
		childTree, err := s.buildDepartmentTree(&child, depth-1, includeEmployees)
		if err != nil {
			return dto.DepartmentTreeResponse{}, err
		}

		response.Children = append(response.Children, childTree)
	}

	return response, nil
}

// delete department

func (s *DepartmentService) DeleteDepartment(input DeleteDepartmentInput) (*dto.DeleteDepartmentResponse, error) {
	if input.ID == 0 {
		return nil, ErrInvalidDepartmentID
	}

	if _, err := s.departmentRepo.GetByID(input.ID); err != nil {
		return nil, ErrDepartmentNotFound
	}

	switch input.Mode {
	case "cascade":
		if err := s.departmentRepo.DeleteByID(input.ID); err != nil {
			return nil, err
		}

		return &dto.DeleteDepartmentResponse{
			Message: "department deleted",
			ID:      input.ID,
			Mode:    input.Mode,
		}, nil

	case "reassign":
		if input.ReassignToDepartmentID == nil {
			return nil, ErrReassignTargetRequired
		}

		reassignToID := *input.ReassignToDepartmentID

		if reassignToID == 0 {
			return nil, ErrInvalidDepartmentID
		}

		if reassignToID == input.ID {
			return nil, ErrCannotReassignToSelf
		}

		if _, err := s.departmentRepo.GetByID(reassignToID); err != nil {
			return nil, ErrReassignTargetNotFound
		}

		wouldCreateCycle, err := s.departmentRepo.WouldCreateCycle(input.ID, reassignToID)
		if err != nil {
			return nil, err
		}

		if wouldCreateCycle {
			return nil, ErrDepartmentWouldCreateCycle
		}

		if err := s.departmentRepo.ReassignAndDelete(input.ID, reassignToID); err != nil {
			return nil, err
		}

		return &dto.DeleteDepartmentResponse{
			Message:                "department deleted",
			ID:                     input.ID,
			Mode:                   input.Mode,
			ReassignToDepartmentID: input.ReassignToDepartmentID,
		}, nil

	default:
		return nil, ErrInvalidDeleteMode
	}
}
