package validator

import (
	"fmt"
	"strings"
)

func RequiredString(value string, fieldName string) (string, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return "", fmt.Errorf("%s is required", fieldName)
	}

	if len(value) > 200 {
		return "", fmt.Errorf("%s must be less than 200 characters", fieldName)
	}

	return value, nil
}
