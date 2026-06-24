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
