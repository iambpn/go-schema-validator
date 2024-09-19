package schema

import (
	"testing"
)

type User struct {
	Name string
	Age  int
}

func TestField(t *testing.T) {
	structVal := structValidation{
		validator: nil,
		rules:     make(map[string]*Schema),
	}

	nameField := "Name"
	ageField := "Age"

	nameRequiredMessage := "Name is required"
	ageGt18Message := "Age must be at least 18"

	structVal.Field(nameField, New().AddValidation("required", nameRequiredMessage))
	structVal.Field(ageField, New().AddValidation("min=18", ageGt18Message))

	if len(structVal.rules) != 2 {
		t.Errorf("Expected 2 fields, got %v", len(structVal.rules))
	}

	for fieldName := range structVal.rules {
		if fieldName != nameField && fieldName != ageField {
			t.Errorf("Expected field name to be %s or %s, got %s", nameField, ageField, fieldName)
		}
	}
}

func TestValidateStruct(t *testing.T) {
	structVal := structValidation{
		validator: nil,
		rules:     make(map[string]*Schema),
	}

	nameField := "Name"
	ageField := "Age"

	nameRequiredMessage := "Name is required"
	ageGt18Message := "Age must be at least 18"

	structVal.Field(nameField, New().AddValidation("required", nameRequiredMessage))
	structVal.Field(ageField, New().AddValidation("min=18", ageGt18Message))

	err := structVal.Validate("")

	if err == nil {
		t.Errorf("Expected error on empty struct, got %v", err)
	}

	noUser := User{
		Name: "John",
		Age:  18,
	}

	err = structVal.Validate(noUser)

	if err != nil {
		t.Errorf("Expected no error on valid struct, got %v", err)
	}

	noNameUser := User{
		Name: "",
		Age:  18,
	}

	err = structVal.Validate(noNameUser)

	if err == nil {
		t.Errorf("Expected error on empty name, got %v", err)
	}

	if err.Error() != nameRequiredMessage {
		t.Errorf("Expected error message to be '%s', got '%v'", nameRequiredMessage, err)
	}

	noAgeUser := User{
		Name: "John",
		Age:  0,
	}

	err = structVal.Validate(noAgeUser)

	if err == nil {
		t.Errorf("Expected error on zero age, got %v", err)
	}

	if err.Error() != ageGt18Message {
		t.Errorf("Expected error message to be '%s', got '%v'", ageGt18Message, err)
	}
}
