package validator

import "fmt"

// Helper method for validating number
func (s *FieldValidator) IsNumber(message ...string) *FieldValidator {
	return s.AddRule("number", message...)
}

// Helper method for validating boolean
func (s *FieldValidator) IsBoolean(message ...string) *FieldValidator {
	return s.AddRule("boolean", message...)
}

// Helper method for validating min number or min length string
func (s *FieldValidator) Min(val int, message ...string) *FieldValidator {
	return s.AddRule(fmt.Sprintf("min=%d", val), message...)
}

// Helper method for validating max number or max length string
func (s *FieldValidator) Max(val int, message ...string) *FieldValidator {
	return s.AddRule(fmt.Sprintf("max=%d", val), message...)
}

// Helper method for validating required fields
func (s *FieldValidator) Required(message ...string) *FieldValidator {
	return s.AddRule("required", message...)
}

// Helper method for validating email
func (s *FieldValidator) Email(message ...string) *FieldValidator {
	return s.AddRule("email", message...)
}

// Helper method for validating Optional fields
func (s *FieldValidator) Optional() *FieldValidator {
	return s.AddRule("omitempty")
}

// Helper method for validating URL
func (s *FieldValidator) URL(message ...string) *FieldValidator {
	return s.AddRule("url", message...)
}

// Helper method for validating UUID
func (s *FieldValidator) UUID(message ...string) *FieldValidator {
	return s.AddRule("uuid", message...)
}

// Helper method for validating Length of string
func (s *FieldValidator) Length(length int, message ...string) *FieldValidator {
	return s.AddRule(fmt.Sprintf("len=%d", length), message...)
}
