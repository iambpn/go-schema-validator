package validator

import (
	"fmt"
	"slices"
)

// IsNumber adds the "number" rule: the value must be a number, or a string that holds a number.
func (s *FieldValidator) IsNumber(message ...string) *FieldValidator {
	return s.AddRule("number", message...)
}

// IsBoolean adds the "boolean" rule: the value must be a bool, or a string that holds a boolean, such as "true".
func (s *FieldValidator) IsBoolean(message ...string) *FieldValidator {
	return s.AddRule("boolean", message...)
}

// Min adds the "min" rule: a number must be at least val, and a string, list, or map must have at least val items.
func (s *FieldValidator) Min(val int, message ...string) *FieldValidator {
	return s.AddRule(fmt.Sprintf("min=%d", val), message...)
}

// Max adds the "max" rule: a number must be at most val, and a string, list, or map must have at most val items.
func (s *FieldValidator) Max(val int, message ...string) *FieldValidator {
	return s.AddRule(fmt.Sprintf("max=%d", val), message...)
}

// Required adds the "required" rule: the value must not be the zero value of its type.
func (s *FieldValidator) Required(message ...string) *FieldValidator {
	return s.AddRule("required", message...)
}

// Email adds the "email" rule: the value must be a valid email address.
func (s *FieldValidator) Email(message ...string) *FieldValidator {
	return s.AddRule("email", message...)
}

// Optional adds the "omitempty" rule: a zero value, such as "" or 0, skips all other rules.
//
// The omitempty rule is added before the other rules, because go-playground only skips the rules that come after it.
func (s *FieldValidator) Optional() *FieldValidator {
	s.fieldRules = slices.Insert(s.fieldRules, 0, validationRule{rule: "omitempty"})
	return s
}

// URL adds the "url" rule: the value must be a valid URL.
func (s *FieldValidator) URL(message ...string) *FieldValidator {
	return s.AddRule("url", message...)
}

// UUID adds the "uuid" rule: the value must be a valid UUID.
func (s *FieldValidator) UUID(message ...string) *FieldValidator {
	return s.AddRule("uuid", message...)
}

// Length adds the "len" rule: a string, list, or map must have exactly length items, and a number must equal length.
func (s *FieldValidator) Length(length int, message ...string) *FieldValidator {
	return s.AddRule(fmt.Sprintf("len=%d", length), message...)
}
