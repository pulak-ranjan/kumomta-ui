package validation

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type Validator struct {
	errors []ValidationError
}

func New() *Validator {
	return &Validator{}
}

func (v *Validator) Required(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.errors = append(v.errors, ValidationError{field, "is required"})
	}
	return v
}

func (v *Validator) MaxLength(field, value string, max int) *Validator {
	if utf8.RuneCountInString(value) > max {
		v.errors = append(v.errors, ValidationError{field, fmt.Sprintf("must be at most %d characters", max)})
	}
	return v
}

func (v *Validator) MinLength(field, value string, min int) *Validator {
	if utf8.RuneCountInString(value) < min {
		v.errors = append(v.errors, ValidationError{field, fmt.Sprintf("must be at least %d characters", min)})
	}
	return v
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func (v *Validator) Email(field, value string) *Validator {
	if !emailRegex.MatchString(value) {
		v.errors = append(v.errors, ValidationError{field, "must be a valid email address"})
	}
	return v
}

func (v *Validator) NoScriptTags(field, value string) *Validator {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "<script") || strings.Contains(lower, "javascript:") {
		v.errors = append(v.errors, ValidationError{field, "contains potentially dangerous content"})
	}
	return v
}

func (v *Validator) Valid() bool {
	return len(v.errors) == 0
}

func (v *Validator) Errors() []ValidationError {
	return v.errors
}

func (v *Validator) AddError(field, message string) {
	v.errors = append(v.errors, ValidationError{field, message})
}
