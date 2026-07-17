package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/OmNom69/org-structure-api/internal/service"
)

type EmployeeHandler struct {
	employeeService *service.EmployeeService
}

func NewEmployeeHandler(employeeService *service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{
		employeeService: employeeService,
	}
}

type CreateEmployeeRequest struct {
	FullName string  `json:"full_name"`
	Position string  `json:"position"`
	HiredAt  *string `json:"hired_at"`
}

type PatchEmployeeRequest struct {
	FullName     optionalJSONField[string] `json:"full_name"`
	Position     optionalJSONField[string] `json:"position"`
	DepartmentID optionalJSONField[uint]   `json:"department_id"`
	HiredAt      optionalJSONField[string] `json:"hired_at"`
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

	employee, err := h.employeeService.CreateEmployee(r.Context(), service.CreateEmployeeInput{
		DepartmentID: uint(departmentID),
		FullName:     req.FullName,
		Position:     req.Position,
		HiredAt:      req.HiredAt,
	})
	if err != nil {
		writeServiceError(w, err)
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

	var req PatchEmployeeRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	employee, err := h.employeeService.PatchEmployee(r.Context(), service.PatchEmployeeInput{
		ID: uint(id),

		FullName:    req.FullName.Value,
		FullNameSet: req.FullName.Set,

		Position:    req.Position.Value,
		PositionSet: req.Position.Set,

		DepartmentID:    req.DepartmentID.Value,
		DepartmentIDSet: req.DepartmentID.Set,

		HiredAt:    req.HiredAt.Value,
		HiredAtSet: req.HiredAt.Set,
	})
	if err != nil {
		writeServiceError(w, err)
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
	employees, err := h.employeeService.GetEmployees(r.Context())
	if err != nil {
		writeServiceError(w, err)
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

	employee, err := h.employeeService.GetEmployee(r.Context(), uint(id))
	if err != nil {
		writeServiceError(w, err)
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

	if err := h.employeeService.DeleteEmployee(r.Context(), uint(id)); err != nil {
		writeServiceError(w, err)
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
