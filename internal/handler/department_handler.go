package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/OmNom69/org-structure-api/internal/repository"
	"github.com/OmNom69/org-structure-api/internal/service"
)

type DepartmentHandler struct {
	departmentService *service.DepartmentService
	departmentRepo    *repository.DepartmentRepository
	employeeRepo      *repository.EmployeeRepository
}

func NewDepartmentHandler(
	departmentService *service.DepartmentService,
	departmentRepo *repository.DepartmentRepository,
	employeeRepo *repository.EmployeeRepository,
) *DepartmentHandler {
	return &DepartmentHandler{
		departmentService: departmentService,
		departmentRepo:    departmentRepo,
		employeeRepo:      employeeRepo,
	}
}

type CreateDepartmentRequest struct {
	Name     string `json:"name"`
	ParentID *uint  `json:"parent_id"`
}

// create department

func (h *DepartmentHandler) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	var req CreateDepartmentRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	department, err := h.departmentService.CreateDepartment(service.CreateDepartmentInput{
		Name:     req.Name,
		ParentID: req.ParentID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		parsedDepth, err := strconv.Atoi(depthStr)
		if err != nil {
			http.Error(w, "invalid depth", http.StatusBadRequest)
			return
		}

		depth = parsedDepth
	}

	includeEmployees := true

	includeEmployeesStr := r.URL.Query().Get("include_employees")
	if includeEmployeesStr != "" {
		parsedIncludeEmployees, err := strconv.ParseBool(includeEmployeesStr)
		if err != nil {
			http.Error(w, "invalid include_employees", http.StatusBadRequest)
			return
		}

		includeEmployees = parsedIncludeEmployees
	}

	departmentTree, err := h.departmentService.GetDepartmentTree(
		uint(id),
		depth,
		includeEmployees,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(departmentTree); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// patch department

func (h *DepartmentHandler) PatchDepartment(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid department id", http.StatusBadRequest)
		return
	}

	var raw map[string]json.RawMessage

	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	input := service.PatchDepartmentInput{
		ID: uint(id),
	}

	nameRaw, ok := raw["name"]
	if ok {
		var nameValue string

		if err := json.Unmarshal(nameRaw, &nameValue); err != nil {
			http.Error(w, "invalid name", http.StatusBadRequest)
			return
		}

		input.Name = &nameValue
	}

	parentRaw, ok := raw["parent_id"]
	if ok {
		input.ParentIDSet = true

		if string(parentRaw) == "null" {
			input.ParentID = nil
		} else {
			var parentID uint

			if err := json.Unmarshal(parentRaw, &parentID); err != nil {
				http.Error(w, "invalid parent_id", http.StatusBadRequest)
				return
			}

			input.ParentID = &parentID
		}
	}

	department, err := h.departmentService.PatchDepartment(input)
	if err != nil {
		http.Error(w, err.Error(), departmentServiceErrorStatus(err))
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

	mode := r.URL.Query().Get("mode")

	var reassignToDepartmentID *uint

	reassignToStr := r.URL.Query().Get("reassign_to_department_id")
	if reassignToStr != "" {
		reassignToID, err := strconv.Atoi(reassignToStr)
		if err != nil || reassignToID <= 0 {
			http.Error(w, "invalid reassign_to_department_id", http.StatusBadRequest)
			return
		}

		id := uint(reassignToID)
		reassignToDepartmentID = &id
	}

	response, err := h.departmentService.DeleteDepartment(service.DeleteDepartmentInput{
		ID:                     uint(id),
		Mode:                   mode,
		ReassignToDepartmentID: reassignToDepartmentID,
	})
	if err != nil {
		http.Error(w, err.Error(), departmentServiceErrorStatus(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
