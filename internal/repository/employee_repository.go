package repository

import (
	"github.com/OmNom69/org-structure-api/internal/model"
	"gorm.io/gorm"
)

type EmployeeRepository struct {
	db *gorm.DB
}

func NewEmployeeRepository(db *gorm.DB) *EmployeeRepository {
	return &EmployeeRepository{db: db}
}

// Create

func (r *EmployeeRepository) Create(employee *model.Employee) error {
	return r.db.Create(employee).Error
}

// include employees

func (r *EmployeeRepository) GetEmployees(departmentID uint) ([]model.Employee, error) {
	var employees []model.Employee

	if err := r.db.Where("department_id = ?", departmentID).Find(&employees).Error; err != nil {
		return nil, err
	}
	return employees, nil
}
