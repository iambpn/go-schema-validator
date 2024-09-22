package schema

import "fmt"

// Helper method for validating number
func (s *Schema) Int(message ...string) *Schema {
	return s.AddValidation("number", message...)
}

// Helper method for validating min number or min length string
func (s *Schema) Min(val int, message ...string) *Schema {
	return s.AddValidation(fmt.Sprintf("min=%d", val), message...)
}

// Helper method for validating max number or max length string
func (s *Schema) Max(val int, message ...string) *Schema {
	return s.AddValidation(fmt.Sprintf("max=%d", val), message...)
}

// Helper method for validating required fields
func (s *Schema) Required(message ...string) *Schema {
	return s.AddValidation("required", message...)
}

// Helper method for validating email
func (s *Schema) Email(message ...string) *Schema {
	return s.AddValidation("email", message...)
}
