package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/OmNom69/org-structure-api/internal/repository"
	"github.com/OmNom69/org-structure-api/internal/service"
)

type EmployeeHandler struct {
	employeeService *service.EmployeeService
	employeeRepo    *repository.EmployeeRepository
	departmentRepo  *repository.DepartmentRepository
}

func NewEmployeeHandler(
	employeeService *service.EmployeeService,
	employeeRepo *repository.EmployeeRepository,
	departmentRepo *repository.DepartmentRepository,
) *EmployeeHandler {
	return &EmployeeHandler{
		employeeService: employeeService,
		employeeRepo:    employeeRepo,
		departmentRepo:  departmentRepo,
	}
}

type CreateEmployeeRequest struct {
	FullName string  `json:"full_name"`
	Position string  `json:"position"`
	HiredAt  *string `json:"hired_at"`
}

type PatchEmployeeRequest struct {
	FullName     *string `json:"full_name"`
	Position     *string `json:"position"`
	DepartmentID *uint   `json:"department_id"`
	HiredAt      *string `json:"hired_at"`
}

// create

func (h *EmployeeHandler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	departmentIDStr := r.PathValue("id")

	departmentID, err := strconv.Atoi(departmentIDStr)
	if err != nil || departmentID <= 0 {
		http.Error(w, "invalid department id", http.StatusBadRequest)
		return
	}

	var req CreateEmployeeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	employee, err := h.employeeService.CreateEmployee(service.CreateEmployeeInput{
		DepartmentID: uint(departmentID),
		FullName:     req.FullName,
		Position:     req.Position,
		HiredAt:      req.HiredAt,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(employee); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// patch

func (h *EmployeeHandler) PatchEmployee(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid employee id", http.StatusBadRequest)
		return
	}

	employee, err := h.employeeRepo.GetByID(uint(id))
	if err != nil {
		http.Error(w, "employee not found", http.StatusNotFound)
		return
	}

	var req PatchEmployeeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.FullName == nil &&
		req.Position == nil &&
		req.DepartmentID == nil &&
		req.HiredAt == nil {
		http.Error(w, "nothing to update", http.StatusBadRequest)
		return
	}

	if req.FullName != nil {
		fullName, err := validateRequiredString(*req.FullName, "full_name")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		employee.FullName = fullName
	}

	if req.Position != nil {
		position, err := validateRequiredString(*req.Position, "position")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		employee.Position = position
	}

	if req.DepartmentID != nil {
		if *req.DepartmentID == 0 {
			http.Error(w, "invalid department id", http.StatusBadRequest)
			return
		}

		if _, err := h.departmentRepo.GetByID(*req.DepartmentID); err != nil {
			http.Error(w, "department not found", http.StatusNotFound)
			return
		}

		employee.DepartmentID = *req.DepartmentID
	}

	if req.HiredAt != nil {
		hiredAt, err := time.Parse("2006-01-02", *req.HiredAt)
		if err != nil {
			http.Error(w, "hired_at must be in YYYY-MM-DD format", http.StatusBadRequest)
			return
		}

		employee.HiredAt = &hiredAt
	}

	if err := h.employeeRepo.Update(employee); err != nil {
		http.Error(w, "Failed to update employee", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(employee); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// get all employees

func (h *EmployeeHandler) GetEmployees(w http.ResponseWriter, r *http.Request) {
	employees, err := h.employeeRepo.GetAllEmployees()
	if err != nil {
		http.Error(w, "Failed to get employees", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(employees); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// get employee by ID

func (h *EmployeeHandler) GetEmployee(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid employee id", http.StatusBadRequest)
		return
	}

	employee, err := h.employeeRepo.GetByID(uint(id))
	if err != nil {
		http.Error(w, "employee not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(employee); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// delete

func (h *EmployeeHandler) DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, "invalid employee id", http.StatusBadRequest)
		return
	}

	if _, err := h.employeeRepo.GetByID(uint(id)); err != nil {
		http.Error(w, "employee not found", http.StatusNotFound)
		return
	}

	if err := h.employeeRepo.DeleteByID(uint(id)); err != nil {
		http.Error(w, "Failed to delete employee", http.StatusInternalServerError)
		return
	}

	response := map[string]any{
		"message": "employee deleted",
		"id":      id,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
