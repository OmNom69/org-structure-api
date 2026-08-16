package service

import (
	"context"
	"time"

	"github.com/OmNom69/org-structure-api/internal/model"
)

type CacheStore interface {
	Get(ctx context.Context, key string) (value []byte, found bool, err error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Increment(ctx context.Context, key string) (int64, error)
}

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
	List(ctx context.Context) ([]model.Employee, error)
	Update(ctx context.Context, employee *model.Employee) error
	DeleteByID(ctx context.Context, id uint) error
	ListByDepartmentID(ctx context.Context, departmentID uint) ([]model.Employee, error)
}
