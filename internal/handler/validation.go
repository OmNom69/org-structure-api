package handler

import "github.com/OmNom69/org-structure-api/internal/validator"

func validateRequiredString(value string, fieldName string) (string, error) {
	return validator.RequiredString(value, fieldName)
}
