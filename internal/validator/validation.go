package validator

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

func RequiredString(value string, fieldName string) (string, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return "", fmt.Errorf("%s is required", fieldName)
	}

	if utf8.RuneCountInString(value) > 200 {
		return "", fmt.Errorf("%s must be less than 200 characters", fieldName)
	}

	return value, nil
}
