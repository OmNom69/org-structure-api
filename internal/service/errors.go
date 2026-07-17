package service

import (
	"errors"
	"fmt"

	"github.com/OmNom69/org-structure-api/internal/storage"
)

var (
	ErrValidation      = errors.New("validation failed")
	ErrNothingToUpdate = errors.New("nothing to update")

	ErrInvalidDepartmentID     = errors.New("invalid department id")
	ErrInvalidDepth            = errors.New("depth must be between 1 and 5")
	ErrDepartmentAlreadyExists = errors.New("department with this name already exists in this parent")

	ErrDepartmentNotFound         = errors.New("department not found")
	ErrInvalidDeleteMode          = errors.New("invalid mode")
	ErrReassignTargetRequired     = errors.New("reassign_to_department_id is required")
	ErrReassignTargetNotFound     = errors.New("reassign target department not found")
	ErrCannotReassignToSelf       = errors.New("cannot reassign department to itself")
	ErrDepartmentWouldCreateCycle = errors.New("department cannot be reassigned inside its own subtree")

	ErrInvalidParentDepartmentID        = errors.New("invalid parent department id")
	ErrParentDepartmentNotFound         = errors.New("parent department not found")
	ErrDepartmentCannotBeParentOfItself = errors.New("department cannot be parent of itself")
	ErrDepartmentMoveWouldCreateCycle   = errors.New("department cannot be moved inside its own subtree")

	ErrInvalidEmployeeID = errors.New("invalid employee id")
	ErrEmployeeNotFound  = errors.New("employee not found")
	ErrInvalidHiredAt    = errors.New("hired_at must use YYYY-MM-DD format")
)

func mapStorageError(err error, notFoundErr error) error {
	if errors.Is(err, storage.ErrNotFound) {
		return notFoundErr
	}

	return err
}

func wrapValidationError(err error) error {
	return fmt.Errorf("%w: %v", ErrValidation, err)
}

func wrapInvalidHiredAtError(err error) error {
	return fmt.Errorf("%w: %v", ErrInvalidHiredAt, err)
}
