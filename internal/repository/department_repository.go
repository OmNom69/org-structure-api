package repository

import (
	"github.com/OmNom69/org-structure-api/internal/model"
	"gorm.io/gorm"
)

type DepartmentRepository struct {
	db *gorm.DB
}

func NewDepartmentRepository(db *gorm.DB) *DepartmentRepository {
	return &DepartmentRepository{db: db}
}

// create

func (r *DepartmentRepository) Create(department *model.Department) error {
	return r.db.Create(department).Error
}

// getByID

func (r *DepartmentRepository) GetByID(id uint) (*model.Department, error) {
	var department model.Department

	if err := r.db.First(&department, id).Error; err != nil {
		return nil, err
	}
	return &department, nil
}

// update

func (r *DepartmentRepository) Update(department *model.Department) error {
	return r.db.Save(department).Error
}

// delete

func (r *DepartmentRepository) DeleteByID(id uint) error {
	return r.db.Delete(&model.Department{}, id).Error
}

// reassign department

func (r *DepartmentRepository) ReassignChildren(fromDepartmentID uint, toDepartmentID uint) error {
	return r.db.Model(&model.Department{}).
		Where("parent_id = ?", fromDepartmentID).
		Update("parent_id", toDepartmentID).Error
}

// reassign employee

func (r *DepartmentRepository) ReassignEmployees(fromDepartmentID uint, toDepartmentID uint) error {
	return r.db.Model(&model.Employee{}).
		Where("department_id = ?", fromDepartmentID).
		Update("department_id", toDepartmentID).Error
}

// сhildren of the department

func (r *DepartmentRepository) GetChildren(parentID uint) ([]model.Department, error) {
	var departments []model.Department

	if err := r.db.Where("parent_id = ?", parentID).Find(&departments).Error; err != nil {
		return nil, err
	}

	return departments, nil
}

// unique name

func (r *DepartmentRepository) ExistsByNameAndParent(name string, parentID *uint) (bool, error) {
	var count int64

	query := r.db.Model(&model.Department{}).Where("name = ?", name)

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

func (r *DepartmentRepository) ExistsByNameAndParentExceptID(name string, parentID *uint, excludeID uint) (bool, error) {
	var count int64

	query := r.db.Model(&model.Department{}).
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
