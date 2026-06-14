package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/OmNom69/org-structure-api/internal/model"
	"github.com/OmNom69/org-structure-api/internal/repository"
)

type CreateEmployeeRequest struct {
	FullName string  `json:"full_name"`
	Position string  `json:"position"`
	HiredAt  *string `json:"hired_at"`
}

type EmployeeHandler struct {
	employeeRepo   *repository.EmployeeRepository
	departmentRepo *repository.DepartmentRepository
}

func NewEmployeeHandler(
	employeeRepo *repository.EmployeeRepository,
	departmentRepo *repository.DepartmentRepository,
) *EmployeeHandler {
	return &EmployeeHandler{
		employeeRepo:   employeeRepo,
		departmentRepo: departmentRepo,
	}
}

func (h *EmployeeHandler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	departmentIDStr := r.PathValue("id")

	departmentID, err := strconv.Atoi(departmentIDStr)
	if err != nil || departmentID <= 0 {
		http.Error(w, "invalid department id", http.StatusBadRequest)
		return
	}

	if _, err := h.departmentRepo.GetByID(uint(departmentID)); err != nil {
		http.Error(w, "department not found", http.StatusNotFound)
		return
	}

	var req CreateEmployeeRequest

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	fullName, err := validateRequiredString(req.FullName, "full_name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	position, err := validateRequiredString(req.Position, "position")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var hiredAt *time.Time

	if req.HiredAt != nil {
		parsedHiredAt, err := time.Parse("2006-01-02", *req.HiredAt)
		if err != nil {
			http.Error(w, "invalid hired_at format, use YYYY-MM-DD", http.StatusBadRequest)
			return
		}

		hiredAt = &parsedHiredAt
	}

	employee := model.Employee{
		DepartmentID: uint(departmentID),
		FullName:     fullName,
		Position:     position,
		HiredAt:      hiredAt,
	}

	if err := h.employeeRepo.Create(&employee); err != nil {
		http.Error(w, "Failed to create employee", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(employee); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
