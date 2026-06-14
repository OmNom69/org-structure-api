package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/OmNom69/org-structure-api/internal/model"
	"github.com/OmNom69/org-structure-api/internal/repository"
)

type DepartmentHandler struct {
	departmentRepo *repository.DepartmentRepository
	employeeRepo   *repository.EmployeeRepository
}

func NewDepartmentHandler(
	departmentRepo *repository.DepartmentRepository,
	employeeRepo *repository.EmployeeRepository,
) *DepartmentHandler {
	return &DepartmentHandler{
		departmentRepo: departmentRepo,
		employeeRepo:   employeeRepo,
	}
}

type DepartmentTreeResponse struct {
	ID        uint                     `json:"id"`
	Name      string                   `json:"name"`
	ParentID  *uint                    `json:"parent_id"`
	CreatedAt time.Time                `json:"created_at"`
	Employees []model.Employee         `json:"employees,omitempty"`
	Children  []DepartmentTreeResponse `json:"children"`
}
type CreateDepartmentRequest struct {
	Name     string `json:"name"`
	ParentID *uint  `json:"parent_id"`
}
type PatchDepartmentRequest struct {
	Name     *string `json:"name"`
	ParentID *uint   `json:"parent_id"`
}

// create department

func (h *DepartmentHandler) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	var req CreateDepartmentRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	name, err := validateRequiredString(req.Name, "name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.ParentID != nil {
		if *req.ParentID == 0 {
			http.Error(w, "invalid parent department id", http.StatusBadRequest)
			return
		}

		if _, err := h.departmentRepo.GetByID(*req.ParentID); err != nil {
			http.Error(w, "parent department not found", http.StatusNotFound)
			return
		}
	}

	exists, err := h.departmentRepo.ExistsByNameAndParent(name, req.ParentID)
	if err != nil {
		http.Error(w, "Failed to check department uniqueness", http.StatusInternalServerError)
		return
	}

	if exists {
		http.Error(w, "department with this name already exists in this parent", http.StatusConflict)
		return
	}

	department := model.Department{
		Name:     name,
		ParentID: req.ParentID,
	}

	if err := h.departmentRepo.Create(&department); err != nil {
		http.Error(w, "Failed to create department", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(department); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

}

// get department

func (h *DepartmentHandler) GetDepartment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid department id", http.StatusBadRequest)
		return
	}

	depth := 1

	depthStr := r.URL.Query().Get("depth")
	if depthStr != "" {
		depth, err = strconv.Atoi(depthStr)
		if err != nil || depth < 1 || depth > 5 {
			http.Error(w, "depth must be between 1 and 5", http.StatusBadRequest)
			return
		}
	}

	includeEmployees := true

	includeEmployeesStr := r.URL.Query().Get("include_employees")
	if includeEmployeesStr != "" {
		includeEmployees, err = strconv.ParseBool(includeEmployeesStr)
		if err != nil {
			http.Error(w, "include_employees must be true or false", http.StatusBadRequest)
			return
		}
	}

	department, err := h.departmentRepo.GetByID(uint(id))
	if err != nil {
		http.Error(w, "department not found", http.StatusNotFound)
		return
	}

	response, err := h.buildDepartmentTree(*department, depth, includeEmployees)
	if err != nil {
		http.Error(w, "Failed to build department tree", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// helper func

func (h *DepartmentHandler) buildDepartmentTree(
	department model.Department,
	depth int,
	includeEmployees bool,
) (DepartmentTreeResponse, error) {

	response := DepartmentTreeResponse{
		ID:        department.ID,
		Name:      department.Name,
		ParentID:  department.ParentID,
		CreatedAt: department.CreatedAt,
		Children:  []DepartmentTreeResponse{},
	}

	if includeEmployees {
		employees, err := h.employeeRepo.GetEmployees(department.ID)
		if err != nil {
			return DepartmentTreeResponse{}, err
		}

		response.Employees = employees
	}

	if depth == 0 {
		return response, nil
	}

	children, err := h.departmentRepo.GetChildren(department.ID)
	if err != nil {
		return DepartmentTreeResponse{}, err
	}

	for _, child := range children {
		childTree, err := h.buildDepartmentTree(child, depth-1, includeEmployees)
		if err != nil {
			return DepartmentTreeResponse{}, err
		}

		response.Children = append(response.Children, childTree)
	}

	return response, nil
}

//patch department

func (h *DepartmentHandler) PatchDepartment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid department id", http.StatusBadRequest)
		return
	}

	department, err := h.departmentRepo.GetByID(uint(id))
	if err != nil {
		http.Error(w, "department not found", http.StatusNotFound)
		return
	}

	var req PatchDepartmentRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		name, err := validateRequiredString(*req.Name, "name")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		department.Name = name
	}

	if req.ParentID != nil {
		if *req.ParentID == 0 {
			http.Error(w, "invalid parent department id", http.StatusBadRequest)
			return
		}

		if *req.ParentID == uint(id) {
			http.Error(w, "department cannot be parent of itself", http.StatusBadRequest)
			return
		}

		if _, err := h.departmentRepo.GetByID(*req.ParentID); err != nil {
			http.Error(w, "parent department not found", http.StatusNotFound)
			return
		}

		department.ParentID = req.ParentID
	}

	exists, err := h.departmentRepo.ExistsByNameAndParentExceptID(
		department.Name,
		department.ParentID,
		department.ID,
	)
	if err != nil {
		http.Error(w, "Failed to check department uniqueness", http.StatusInternalServerError)
		return
	}

	if exists {
		http.Error(w, "department with this name already exists in this parent", http.StatusConflict)
		return
	}

	if err := h.departmentRepo.Update(department); err != nil {
		http.Error(w, "Failed to update department", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(department); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// delete department (cascade or reassign)

func (h *DepartmentHandler) DeleteDepartment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid department id", http.StatusBadRequest)
		return
	}

	if _, err := h.departmentRepo.GetByID(uint(id)); err != nil {
		http.Error(w, "department not found", http.StatusNotFound)
		return
	}

	mode := r.URL.Query().Get("mode")

	switch mode {
	case "cascade":
		if err := h.departmentRepo.DeleteByID(uint(id)); err != nil {
			http.Error(w, "Failed to delete department", http.StatusInternalServerError)
			return
		}

	case "reassign":
		reassignToStr := r.URL.Query().Get("reassign_to_department_id")
		if reassignToStr == "" {
			http.Error(w, "reassign_to_department_id is required", http.StatusBadRequest)
			return
		}

		reassignToID, err := strconv.Atoi(reassignToStr)
		if err != nil || reassignToID <= 0 {
			http.Error(w, "invalid reassign_to_department_id", http.StatusBadRequest)
			return
		}

		if reassignToID == id {
			http.Error(w, "cannot reassign department to itself", http.StatusBadRequest)
			return
		}

		if _, err := h.departmentRepo.GetByID(uint(reassignToID)); err != nil {
			http.Error(w, "reassign target department not found", http.StatusNotFound)
			return
		}

		if err := h.departmentRepo.ReassignChildren(uint(id), uint(reassignToID)); err != nil {
			http.Error(w, "Failed to reassign child departments", http.StatusInternalServerError)
			return
		}

		if err := h.departmentRepo.ReassignEmployees(uint(id), uint(reassignToID)); err != nil {
			http.Error(w, "Failed to reassign employees", http.StatusInternalServerError)
			return
		}

		if err := h.departmentRepo.DeleteByID(uint(id)); err != nil {
			http.Error(w, "Failed to delete department", http.StatusInternalServerError)
			return
		}

	default:
		http.Error(w, "invalid mode", http.StatusBadRequest)
		return
	}

	response := map[string]any{
		"message": "department deleted",
		"id":      id,
		"mode":    mode,
	}

	if mode == "reassign" {
		response["reassign_to_department_id"] = r.URL.Query().Get("reassign_to_department_id")
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
