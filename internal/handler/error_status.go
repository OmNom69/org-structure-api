package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/OmNom69/org-structure-api/internal/service"
)

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

	http.Error(w, message, status)
}
