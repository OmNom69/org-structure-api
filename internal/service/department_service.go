package service

import (
	"context"
	"time"

	"github.com/OmNom69/org-structure-api/internal/dto"
	"github.com/OmNom69/org-structure-api/internal/model"
	"github.com/OmNom69/org-structure-api/internal/validator"
)

type DepartmentService struct {
	departmentRepo DepartmentRepository
	employeeRepo   EmployeeRepository
	treeCache      departmentTreeCache
}

func NewDepartmentService(
	departmentRepo DepartmentRepository,
	employeeRepo EmployeeRepository,
	cacheStore CacheStore,
	cacheTTL time.Duration,
) *DepartmentService {
	return &DepartmentService{
		departmentRepo: departmentRepo,
		employeeRepo:   employeeRepo,
		treeCache:      newDepartmentTreeCache(cacheStore, cacheTTL),
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

type PatchDepartmentInput struct {
	ID          uint
	Name        *string
	ParentID    *uint
	ParentIDSet bool
}

func (s *DepartmentService) CreateDepartment(ctx context.Context, input CreateDepartmentInput) (*model.Department, error) {
	name, err := validator.RequiredString(input.Name, "name")
	if err != nil {
		return nil, wrapValidationError(err)
	}

	if input.ParentID != nil {
		if *input.ParentID == 0 {
			return nil, ErrInvalidParentDepartmentID
		}

		if _, err := s.departmentRepo.GetByID(ctx, *input.ParentID); err != nil {
			return nil, mapStorageError(err, ErrParentDepartmentNotFound)
		}
	}

	exists, err := s.departmentRepo.ExistsByNameAndParent(ctx, name, input.ParentID)
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

	if err := s.departmentRepo.Create(ctx, &department); err != nil {
		return nil, mapStorageConflict(err, ErrDepartmentAlreadyExists)
	}

	bumpDepartmentTreeEpoch(ctx, s.treeCache.store)

	return &department, nil
}

func (s *DepartmentService) GetDepartmentTree(ctx context.Context, id uint, depth int, includeEmployees bool) (*dto.DepartmentTreeResponse, error) {
	if id == 0 {
		return nil, ErrInvalidDepartmentID
	}

	if depth < 1 || depth > 5 {
		return nil, ErrInvalidDepth
	}

	if s.treeCache.store == nil {
		return s.loadDepartmentTree(ctx, id, depth, includeEmployees)
	}

	return s.getDepartmentTreeCached(ctx, id, depth, includeEmployees)
}

func (s *DepartmentService) buildDepartmentTree(ctx context.Context, department *model.Department, depth int, includeEmployees bool) (dto.DepartmentTreeResponse, error) {
	response := dto.DepartmentTreeResponse{
		ID:        department.ID,
		Name:      department.Name,
		ParentID:  department.ParentID,
		CreatedAt: department.CreatedAt,
		Children:  []dto.DepartmentTreeResponse{},
	}

	if includeEmployees {
		employees, err := s.employeeRepo.ListByDepartmentID(ctx, department.ID)
		if err != nil {
			return dto.DepartmentTreeResponse{}, err
		}

		response.Employees = &employees
	}

	if depth <= 0 {
		return response, nil
	}

	children, err := s.departmentRepo.GetChildren(ctx, department.ID)
	if err != nil {
		return dto.DepartmentTreeResponse{}, err
	}

	for _, child := range children {
		childTree, err := s.buildDepartmentTree(ctx, &child, depth-1, includeEmployees)
		if err != nil {
			return dto.DepartmentTreeResponse{}, err
		}

		response.Children = append(response.Children, childTree)
	}

	return response, nil
}

func (s *DepartmentService) loadDepartmentTree(ctx context.Context, id uint, depth int, includeEmployees bool) (*dto.DepartmentTreeResponse, error) {
	department, err := s.departmentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, mapStorageError(err, ErrDepartmentNotFound)
	}

	tree, err := s.buildDepartmentTree(ctx, department, depth, includeEmployees)
	if err != nil {
		return nil, err
	}

	return &tree, nil
}

func (s *DepartmentService) DeleteDepartment(ctx context.Context, input DeleteDepartmentInput) (*dto.DeleteDepartmentResponse, error) {
	if input.ID == 0 {
		return nil, ErrInvalidDepartmentID
	}

	if _, err := s.departmentRepo.GetByID(ctx, input.ID); err != nil {
		return nil, mapStorageError(err, ErrDepartmentNotFound)
	}

	switch input.Mode {
	case "cascade":
		if err := s.departmentRepo.DeleteByID(ctx, input.ID); err != nil {
			return nil, err
		}

		bumpDepartmentTreeEpoch(ctx, s.treeCache.store)

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

		if _, err := s.departmentRepo.GetByID(ctx, reassignToID); err != nil {
			return nil, mapStorageError(err, ErrReassignTargetNotFound)
		}

		wouldCreateCycle, err := s.departmentRepo.WouldCreateCycle(ctx, input.ID, reassignToID)
		if err != nil {
			return nil, err
		}

		if wouldCreateCycle {
			return nil, ErrDepartmentWouldCreateCycle
		}

		if err := s.departmentRepo.ReassignAndDelete(ctx, input.ID, reassignToID); err != nil {
			return nil, mapStorageConflict(err, ErrDepartmentAlreadyExists)
		}

		bumpDepartmentTreeEpoch(ctx, s.treeCache.store)

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

func (s *DepartmentService) PatchDepartment(ctx context.Context, input PatchDepartmentInput) (*model.Department, error) {
	if input.ID == 0 {
		return nil, ErrInvalidDepartmentID
	}

	department, err := s.departmentRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, mapStorageError(err, ErrDepartmentNotFound)
	}

	if input.Name == nil && !input.ParentIDSet {
		return nil, ErrNothingToUpdate
	}

	if input.Name != nil {
		name, err := validator.RequiredString(*input.Name, "name")
		if err != nil {
			return nil, wrapValidationError(err)
		}

		department.Name = name
	}

	if input.ParentIDSet {
		if input.ParentID == nil {
			department.ParentID = nil
		} else {
			parentID := *input.ParentID

			if parentID == 0 {
				return nil, ErrInvalidParentDepartmentID
			}

			if parentID == input.ID {
				return nil, ErrDepartmentCannotBeParentOfItself
			}

			if _, err := s.departmentRepo.GetByID(ctx, parentID); err != nil {
				return nil, mapStorageError(err, ErrParentDepartmentNotFound)
			}

			wouldCreateCycle, err := s.departmentRepo.WouldCreateCycle(ctx, input.ID, parentID)
			if err != nil {
				return nil, err
			}

			if wouldCreateCycle {
				return nil, ErrDepartmentMoveWouldCreateCycle
			}

			department.ParentID = &parentID
		}
	}

	exists, err := s.departmentRepo.ExistsByNameAndParentExceptID(
		ctx,
		department.Name,
		department.ParentID,
		department.ID,
	)
	if err != nil {
		return nil, err
	}

	if exists {
		return nil, ErrDepartmentAlreadyExists
	}

	if err := s.departmentRepo.Update(ctx, department); err != nil {
		return nil, mapStorageConflict(err, ErrDepartmentAlreadyExists)
	}

	bumpDepartmentTreeEpoch(ctx, s.treeCache.store)

	return department, nil
}
