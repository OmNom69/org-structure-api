package service

import "errors"

var (
	ErrNothingToUpdate         = errors.New("nothing to update")
	ErrInvalidDepartmentID     = errors.New("invalid department id")
	ErrDepartmentAlreadyExists = errors.New("department with this name already exists in this parent")
)
