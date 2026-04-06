package validation

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

func New() (*validator.Validate, error) {
	validate := validator.New(validator.WithRequiredStructEnabled())

	rules := []struct {
		tag string
		fn  validator.Func
	}{
		{tag: "spack_duration", fn: validatePositiveDuration},
		{tag: "spack_flexible_duration", fn: validatePositiveFlexibleDuration},
		{tag: "spack_relative_path", fn: validateRelativePath},
		{tag: "spack_widths", fn: validateWidths},
	}
	for _, rule := range rules {
		if err := validate.RegisterValidation(rule.tag, rule.fn); err != nil {
			return nil, fmt.Errorf("register %s validator: %w", rule.tag, err)
		}
	}

	return validate, nil
}

func validatePositiveDuration(fl validator.FieldLevel) bool {
	raw := strings.TrimSpace(fl.Field().String())
	if raw == "" {
		return true
	}
	d, err := time.ParseDuration(raw)
	return err == nil && d > 0
}

func validatePositiveFlexibleDuration(fl validator.FieldLevel) bool {
	raw := strings.TrimSpace(fl.Field().String())
	if raw == "" {
		return true
	}
	return ParseFlexibleDuration(raw) > 0
}

func validateRelativePath(fl validator.FieldLevel) bool {
	return IsRelativePath(fl.Field().String())
}

func validateWidths(fl validator.FieldLevel) bool {
	raw := strings.TrimSpace(fl.Field().String())
	if raw == "" {
		return true
	}
	return ParseWidths(raw).Len() > 0
}

func IsRelativePath(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	if strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "\\") {
		return false
	}
	return !filepath.IsAbs(raw)
}
