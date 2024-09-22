package schema

import (
	"fmt"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	schema := New()

	if schema == nil {
		t.Errorf("Expected schema to be created, got %v", schema)
	}
}

func TestAddValidation(t *testing.T) {
	schema := New()

	const validation = "required"
	const message = "This field is required"

	schema.AddValidation(validation, message, "lol")

	if len(schema.rules) != 1 && schema.rules[0].message == message && schema.rules[0].rule == validation {
		t.Errorf("Expected validation rule to be added, got %v", schema.rules)
	}
}

func TestCompileRules(t *testing.T) {
	schema := New()

	const validation1 = "required"
	const validation2 = "min=3"

	schema.AddValidation(validation1)
	schema.AddValidation(validation2)

	if schema.compileRules() != fmt.Sprintf("%s,%s", validation1, validation2) {
		t.Errorf("Expected validation rule to be compiled, got %v", schema.compileRules())
	}
}

func TestStruct(t *testing.T) {
	schema := New().Struct()

	if schema == nil {
		t.Errorf("Expected struct to be created, got %v", schema)
	}

	if reflect.TypeOf(schema) != reflect.TypeOf(&structValidation{}) {
		t.Errorf("Expected struct to be of type StructValidation, got %v", reflect.TypeOf(schema))
	}
}

func TestValidate(t *testing.T) {
	schema := New()

	if schema == nil {
		t.Errorf("Expected schema to be created, got %v", schema)
	}

	err := schema.Validate("hello")

	if err != nil {
		t.Errorf("Expected no error on string, got %v", err)
	}

	const requiredMessage = "This field is required"
	schema.AddValidation("required", requiredMessage)

	err = schema.Validate("")

	if err == nil {
		t.Errorf("Expected error on empty string, got %v", err)
	}

	if err.Error() != requiredMessage {
		t.Errorf("Expected error message to be %s, got %v", requiredMessage, err)
	}

	schema.AddValidation("min=3")

	err = schema.Validate("hello")

	if err != nil {
		t.Errorf("Expected no error on string, got %v", err)
	}

	err = schema.Validate("he")

	if err == nil {
		t.Errorf("Expected error on string, got %v", err)
	}

	if err.Error() == "" {
		t.Errorf("Expected error message to be non empty, got \"\"")
	}
}
