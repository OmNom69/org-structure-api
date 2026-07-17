package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/OmNom69/org-structure-api/internal/model"
	"github.com/OmNom69/org-structure-api/internal/storage"
	"gorm.io/gorm"
)

type DepartmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

// create

func (r *DepartmentRepository) Create(ctx context.Context, department *model.Department) error {
	return r.db.WithContext(ctx).Create(department).Error
}

// getByID

func (r *DepartmentRepository) GetByID(ctx context.Context, id uint) (*model.Department, error) {
	var department model.Department

	err := r.db.WithContext(ctx).First(&department, id).Error

	switch {
	case err == nil:
		return &department, nil

	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, storage.ErrNotFound

	default:
		return nil, fmt.Errorf("get department by id %d: %w", id, err)
	}
}

// update

func (r *DepartmentRepository) Update(ctx context.Context, department *model.Department) error {
	return r.db.WithContext(ctx).Save(department).Error
}

// delete

func (r *DepartmentRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Department{}, id).Error
}

// transaction | reassign and delete

func (r *DepartmentRepository) ReassignAndDelete(
	ctx context.Context,
	fromDepartmentID uint,
	toDepartmentID uint,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Department{}).
			Where("parent_id = ?", fromDepartmentID).
			Update("parent_id", toDepartmentID).Error; err != nil {
			return err
		}

		if err := tx.Model(&model.Employee{}).
			Where("department_id = ?", fromDepartmentID).
			Update("department_id", toDepartmentID).Error; err != nil {
			return err
		}

		if err := tx.Delete(&model.Department{}, fromDepartmentID).Error; err != nil {
			return err
		}

		return nil
	})
}

// сhildren of the department

func (r *DepartmentRepository) GetChildren(ctx context.Context, parentID uint) ([]model.Department, error) {
	var departments []model.Department

	if err := r.db.WithContext(ctx).Where("parent_id = ?", parentID).Find(&departments).Error; err != nil {
		return nil, err
	}

	return departments, nil
}

// unique name

func (r *DepartmentRepository) ExistsByNameAndParent(
	ctx context.Context,
	name string,
	parentID *uint,
) (bool, error) {
	var count int64

	query := r.db.WithContext(ctx).Model(&model.Department{}).Where("name = ?", name)

	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// unique name for PATCH

func (r *DepartmentRepository) ExistsByNameAndParentExceptID(
	ctx context.Context,
	name string,
	parentID *uint,
	excludeID uint,
) (bool, error) {
	var count int64

	query := r.db.WithContext(ctx).Model(&model.Department{}).
		Where("name = ?", name).
		Where("id <> ?", excludeID)

	if parentID == nil {
		query = query.Where("parent_id IS NULL")
	} else {
		query = query.Where("parent_id = ?", *parentID)
	}

	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// would create cycle?

func (r *DepartmentRepository) WouldCreateCycle(
	ctx context.Context,
	departmentID uint,
	newParentID uint,
) (bool, error) {
	currentID := newParentID

	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}

		if currentID == departmentID {
			return true, nil
		}

		currentDepartment, err := r.GetByID(ctx, currentID)
		if err != nil {
			return false, err
		}

		if currentDepartment.ParentID == nil {
			return false, nil
		}

		currentID = *currentDepartment.ParentID
	}
}
