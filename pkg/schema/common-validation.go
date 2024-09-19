package schema

import "fmt"

func (s *Schema) Int(message ...string) *Schema {
	return s.AddValidation("number", message...)
}

func (s *Schema) Min(val int, message ...string) *Schema {
	return s.AddValidation(fmt.Sprintf("min=%d", val), message...)
}

func (s *Schema) Max(val int, message ...string) *Schema {
	return s.AddValidation(fmt.Sprintf("max=%d", val), message...)
}

func (s *Schema) Required(message ...string) *Schema {
	return s.AddValidation("required", message...)
}

func (s *Schema) Email(message ...string) *Schema {
	return s.AddValidation("email", message...)
}
