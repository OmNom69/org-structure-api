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

	department, err := h.departmentRepo.GetByID(uint(id))
	if err != nil {
		http.Error(w, "department not found", http.StatusNotFound)
		return
	}

	var raw map[string]json.RawMessage

	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if _, okName := raw["name"]; !okName {
		if _, okParent := raw["parent_id"]; !okParent {
			http.Error(w, "nothing to update", http.StatusBadRequest)
			return
		}
	}

	nameRaw, ok := raw["name"]
	if ok {
		var nameValue string

		if err := json.Unmarshal(nameRaw, &nameValue); err != nil {
			http.Error(w, "invalid name", http.StatusBadRequest)
			return
		}

		name, err := validateRequiredString(nameValue, "name")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		department.Name = name
	}

	parentRaw, ok := raw["parent_id"]
	if ok {
		if string(parentRaw) == "null" {
			department.ParentID = nil
		} else {
			var parentID uint

			if err := json.Unmarshal(parentRaw, &parentID); err != nil {
				http.Error(w, "invalid parent_id", http.StatusBadRequest)
				return
			}

			if parentID == 0 {
				http.Error(w, "invalid parent department id", http.StatusBadRequest)
				return
			}

			if parentID == uint(id) {
				http.Error(w, "department cannot be parent of itself", http.StatusBadRequest)
				return
			}

			if _, err := h.departmentRepo.GetByID(parentID); err != nil {
				http.Error(w, "parent department not found", http.StatusNotFound)
				return
			}

			wouldCreateCycle, err := h.departmentRepo.WouldCreateCycle(uint(id), parentID)
			if err != nil {
				http.Error(w, "Failed to check department cycle", http.StatusInternalServerError)
				return
			}

			if wouldCreateCycle {
				http.Error(w, "department cannot be moved inside its own subtree", http.StatusBadRequest)
				return
			}

			department.ParentID = &parentID
		}
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

		wouldCreateCycle, err := h.departmentRepo.WouldCreateCycle(uint(id), uint(reassignToID))
		if err != nil {
			http.Error(w, "Failed to check department cycle", http.StatusInternalServerError)
			return
		}

		if wouldCreateCycle {
			http.Error(w, "department cannot be reassigned inside its own subtree", http.StatusBadRequest)
			return
		}

		if err := h.departmentRepo.ReassignAndDelete(uint(id), uint(reassignToID)); err != nil {
			http.Error(w, "Failed to reassign and delete department", http.StatusInternalServerError)
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
