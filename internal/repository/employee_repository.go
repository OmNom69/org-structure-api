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

// create

func (r *EmployeeRepository) Create(employee *model.Employee) error {
	return r.db.Create(employee).Error
}

// get by ID

func (r *EmployeeRepository) GetByID(id uint) (*model.Employee, error) {
	var employee model.Employee

	if err := r.db.First(&employee, id).Error; err != nil {
		return nil, err
	}

	return &employee, nil
}

// get all employees

func (r *EmployeeRepository) GetAllEmployees() ([]model.Employee, error) {
	var employees []model.Employee

	if err := r.db.Find(&employees).Error; err != nil {
		return nil, err
	}

	return employees, nil
}

// update

func (r *EmployeeRepository) Update(employee *model.Employee) error {
	return r.db.Save(employee).Error
}

// include employees

func (r *EmployeeRepository) GetEmployeesForTree(departmentID uint) ([]model.Employee, error) {
	var employees []model.Employee

	if err := r.db.Where("department_id = ?", departmentID).Find(&employees).Error; err != nil {
		return nil, err
	}
	return employees, nil
}

// delete

func (r *EmployeeRepository) DeleteByID(id uint) error {
	return r.db.Delete(&model.Employee{}, id).Error
}
