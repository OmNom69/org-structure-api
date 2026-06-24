package handler

import (
	"errors"
	"net/http"

	"github.com/OmNom69/org-structure-api/internal/service"
)

func departmentServiceErrorStatus(err error) int {
	switch {
	case errors.Is(err, service.ErrDepartmentNotFound),
		errors.Is(err, service.ErrReassignTargetNotFound):
		return http.StatusNotFound

	case errors.Is(err, service.ErrInvalidDepartmentID),
		errors.Is(err, service.ErrInvalidDeleteMode),
		errors.Is(err, service.ErrReassignTargetRequired),
		errors.Is(err, service.ErrCannotReassignToSelf),
		errors.Is(err, service.ErrDepartmentWouldCreateCycle):
		return http.StatusBadRequest

	default:
		return http.StatusInternalServerError
	}
}
