package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/OmNom69/org-structure-api/internal/model"
	"github.com/OmNom69/org-structure-api/internal/storage"
	"gorm.io/gorm"
)

type EmployeeRepository struct {
	db *gorm.DB
}

func NewEmployeeRepository(db *gorm.DB) *EmployeeRepository {
	return &EmployeeRepository{db: db}
}

// create

func (r *EmployeeRepository) Create(ctx context.Context, employee *model.Employee) error {
	return r.db.WithContext(ctx).Create(employee).Error
}

// get by ID

func (r *EmployeeRepository) GetByID(ctx context.Context, id uint) (*model.Employee, error) {
	var employee model.Employee

	err := r.db.WithContext(ctx).First(&employee, id).Error

	switch {
	case err == nil:
		return &employee, nil

	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, storage.ErrNotFound

	default:
		return nil, fmt.Errorf("get employee by id %d: %w", id, err)
	}
}

// get all employees

func (r *EmployeeRepository) GetAllEmployees(ctx context.Context) ([]model.Employee, error) {
	var employees []model.Employee

	if err := r.db.WithContext(ctx).Find(&employees).Error; err != nil {
		return nil, err
	}

	return employees, nil
}

// update

func (r *EmployeeRepository) Update(ctx context.Context, employee *model.Employee) error {
	return r.db.WithContext(ctx).Save(employee).Error
}

// include employees

func (r *EmployeeRepository) GetEmployeesForTree(
	ctx context.Context,
	departmentID uint,
) ([]model.Employee, error) {
	var employees []model.Employee

	if err := r.db.WithContext(ctx).Where("department_id = ?", departmentID).Find(&employees).Error; err != nil {
		return nil, err
	}
	return employees, nil
}

// delete

func (r *EmployeeRepository) DeleteByID(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Employee{}, id).Error
}
