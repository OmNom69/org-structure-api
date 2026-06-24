package service

import "errors"

var (
	ErrNothingToUpdate         = errors.New("nothing to update")
	ErrInvalidDepartmentID     = errors.New("invalid department id")
	ErrInvalidDepth            = errors.New("depth must be between 1 and 5")
	ErrDepartmentAlreadyExists = errors.New("department with this name already exists in this parent")

	ErrDepartmentNotFound         = errors.New("department not found")
	ErrInvalidDeleteMode          = errors.New("invalid mode")
	ErrReassignTargetRequired     = errors.New("reassign_to_department_id is required")
	ErrReassignTargetNotFound     = errors.New("reassign target department not found")
	ErrCannotReassignToSelf       = errors.New("cannot reassign department to itself")
	ErrDepartmentWouldCreateCycle = errors.New("department cannot be reassigned inside its own subtree")
)
