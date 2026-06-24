package dto

import (
	"time"

	"github.com/OmNom69/org-structure-api/internal/model"
)

type DepartmentTreeResponse struct {
	ID        uint                     `json:"id"`
	Name      string                   `json:"name"`
	ParentID  *uint                    `json:"parent_id"`
	CreatedAt time.Time                `json:"created_at"`
	Employees []model.Employee         `json:"employees,omitempty"`
	Children  []DepartmentTreeResponse `json:"children"`
}

type DeleteDepartmentResponse struct {
	Message                string `json:"message"`
	ID                     uint   `json:"id"`
	Mode                   string `json:"mode"`
	ReassignToDepartmentID *uint  `json:"reassign_to_department_id,omitempty"`
}
