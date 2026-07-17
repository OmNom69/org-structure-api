package service

import (
	"context"

	"github.com/OmNom69/org-structure-api/internal/model"
)

type DepartmentRepository interface {
	Create(ctx context.Context, department *model.Department) error
	GetByID(ctx context.Context, id uint) (*model.Department, error)
	Update(ctx context.Context, department *model.Department) error
	DeleteByID(ctx context.Context, id uint) error

	ReassignAndDelete(
		ctx context.Context,
		fromDepartmentID uint,
		toDepartmentID uint,
	) error

	GetChildren(ctx context.Context, parentID uint) ([]model.Department, error)

	ExistsByNameAndParent(
		ctx context.Context,
		name string,
		parentID *uint,
	) (bool, error)

	ExistsByNameAndParentExceptID(
		ctx context.Context,
		name string,
		parentID *uint,
		excludeID uint,
	) (bool, error)

	WouldCreateCycle(
		ctx context.Context,
		departmentID uint,
		newParentID uint,
	) (bool, error)
}

type EmployeeRepository interface {
	Create(ctx context.Context, employee *model.Employee) error
	GetByID(ctx context.Context, id uint) (*model.Employee, error)
	GetAllEmployees(ctx context.Context) ([]model.Employee, error)
	Update(ctx context.Context, employee *model.Employee) error
	DeleteByID(ctx context.Context, id uint) error
	GetEmployeesForTree(ctx context.Context, departmentID uint) ([]model.Employee, error)
}
