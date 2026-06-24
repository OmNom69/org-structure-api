package service

import "errors"

var (
	ErrNothingToUpdate         = errors.New("nothing to update")
	ErrInvalidDepartmentID     = errors.New("invalid department id")
	ErrInvalidDepth            = errors.New("depth must be between 1 and 5")
	ErrDepartmentAlreadyExists = errors.New("department with this name already exists in this parent")
)
