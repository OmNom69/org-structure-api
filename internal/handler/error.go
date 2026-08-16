package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/OmNom69/org-structure-api/internal/service"
)

type ErrorResponse struct {
	Error ErrorDetails `json:"error"`
}

type ErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func serviceErrorStatus(err error) int {
	switch {
	case errors.Is(err, service.ErrDepartmentNotFound),
		errors.Is(err, service.ErrReassignTargetNotFound),
		errors.Is(err, service.ErrParentDepartmentNotFound),
		errors.Is(err, service.ErrEmployeeNotFound):
		return http.StatusNotFound

	case errors.Is(err, service.ErrDepartmentAlreadyExists):
		return http.StatusConflict

	case errors.Is(err, service.ErrValidation),
		errors.Is(err, service.ErrNothingToUpdate),
		errors.Is(err, service.ErrInvalidDepartmentID),
		errors.Is(err, service.ErrInvalidEmployeeID),
		errors.Is(err, service.ErrInvalidDepth),
		errors.Is(err, service.ErrInvalidDeleteMode),
		errors.Is(err, service.ErrReassignTargetRequired),
		errors.Is(err, service.ErrCannotReassignToSelf),
		errors.Is(err, service.ErrDepartmentWouldCreateCycle),
		errors.Is(err, service.ErrInvalidParentDepartmentID),
		errors.Is(err, service.ErrDepartmentCannotBeParentOfItself),
		errors.Is(err, service.ErrDepartmentMoveWouldCreateCycle),
		errors.Is(err, service.ErrInvalidHiredAt):
		return http.StatusBadRequest

	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout

	case errors.Is(err, context.Canceled):
		return http.StatusRequestTimeout

	default:
		return http.StatusInternalServerError
	}
}

func serviceErrorCode(err error) string {
	switch {
	case errors.Is(err, service.ErrDepartmentNotFound):
		return "department_not_found"

	case errors.Is(err, service.ErrReassignTargetNotFound):
		return "reassign_target_not_found"

	case errors.Is(err, service.ErrParentDepartmentNotFound):
		return "parent_department_not_found"

	case errors.Is(err, service.ErrEmployeeNotFound):
		return "employee_not_found"

	case errors.Is(err, service.ErrDepartmentAlreadyExists):
		return "department_already_exists"

	case errors.Is(err, service.ErrNothingToUpdate):
		return "nothing_to_update"

	case errors.Is(err, service.ErrInvalidDepartmentID):
		return "invalid_department_id"

	case errors.Is(err, service.ErrInvalidEmployeeID):
		return "invalid_employee_id"

	case errors.Is(err, service.ErrInvalidDepth):
		return "invalid_depth"

	case errors.Is(err, service.ErrInvalidDeleteMode):
		return "invalid_delete_mode"

	case errors.Is(err, service.ErrReassignTargetRequired):
		return "reassign_target_required"

	case errors.Is(err, service.ErrCannotReassignToSelf):
		return "cannot_reassign_to_self"

	case errors.Is(err, service.ErrDepartmentWouldCreateCycle),
		errors.Is(err, service.ErrDepartmentMoveWouldCreateCycle):
		return "department_cycle_detected"

	case errors.Is(err, service.ErrInvalidParentDepartmentID):
		return "invalid_parent_department_id"

	case errors.Is(err, service.ErrDepartmentCannotBeParentOfItself):
		return "department_cannot_be_parent_of_itself"

	case errors.Is(err, service.ErrInvalidHiredAt):
		return "invalid_hired_at"

	case errors.Is(err, service.ErrValidation):
		return "validation_error"

	case errors.Is(err, context.DeadlineExceeded):
		return "request_deadline_exceeded"

	case errors.Is(err, context.Canceled):
		return "request_canceled"

	default:
		return "internal_server_error"
	}
}

func writeJSONError(
	ctx context.Context,
	logger *slog.Logger,
	w http.ResponseWriter,
	status int,
	code string,
	message string,
) {
	response := ErrorResponse{
		Error: ErrorDetails{
			Code:    code,
			Message: message,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.ErrorContext(
			ctx,
			"failed to encode error response",
			slog.Int("status", status),
			slog.String("code", code),
			slog.Any("error", err),
		)
	}
}

func writeServiceError(
	ctx context.Context,
	logger *slog.Logger,
	w http.ResponseWriter,
	err error,
	operation string,
) {
	status := serviceErrorStatus(err)

	if status >= http.StatusInternalServerError {
		logger.ErrorContext(
			ctx,
			"service operation failed",
			slog.String("operation", operation),
			slog.Int("status", status),
			slog.Any("error", err),
		)
	}

	message := err.Error()

	if status == http.StatusInternalServerError {
		message = "internal server error"
	}

	code := serviceErrorCode(err)

	writeJSONError(
		ctx,
		logger,
		w,
		status,
		code,
		message,
	)
}
