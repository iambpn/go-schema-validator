package validator

import "fmt"

// Helper method for validating number
func (s *Validator) Int(message ...string) *Validator {
	return s.AddRule("number", message...)
}

// Helper method for validating min number or min length string
func (s *Validator) Min(val int, message ...string) *Validator {
	return s.AddRule(fmt.Sprintf("min=%d", val), message...)
}

// Helper method for validating max number or max length string
func (s *Validator) Max(val int, message ...string) *Validator {
	return s.AddRule(fmt.Sprintf("max=%d", val), message...)
}

// Helper method for validating required fields
func (s *Validator) Required(message ...string) *Validator {
	return s.AddRule("required", message...)
}

// Helper method for validating email
func (s *Validator) Email(message ...string) *Validator {
	return s.AddRule("email", message...)
}
