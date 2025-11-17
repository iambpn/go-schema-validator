package validator

import (
	"fmt"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	schema := New()

	if schema == nil {
		t.Errorf("Expected schema to be created, got %v", schema)
	}
}

func TestAddRule(t *testing.T) {
	schema := New()

	const validation = "required"
	const message = "This field is required"

	schema.AddRule(validation, message, "lol")

	// corrected condition: fail if length is not 1 OR stored values don't match
	if len(schema.rules) != 1 || schema.rules[0].message != message || schema.rules[0].rule != validation {
		t.Errorf("Expected validation rule to be added, got %v", schema.rules)
	}
}

func TestCompileRules(t *testing.T) {
	schema := New()

	const validation1 = "required"
	const validation2 = "min=3"

	schema.AddRule(validation1)
	schema.AddRule(validation2)

	if schema.compileRules() != fmt.Sprintf("%s,%s", validation1, validation2) {
		t.Errorf("Expected validation rule to be compiled, got %v", schema.compileRules())
	}
}

func TestValidatorValidate(t *testing.T) {
	schema := New()

	if schema == nil {
		t.Errorf("Expected schema to be created, got %v", schema)
	}

	err := schema.Validate("hello")

	if err != nil {
		t.Errorf("Expected no error on string, got %v", err)
	}

	const requiredMessage = "This field is required"
	schema.AddRule("required", requiredMessage)

	err = schema.Validate("")

	if err == nil {
		t.Errorf("Expected error on empty string, got %v", err)
	}

	if err.Error() != requiredMessage {
		t.Errorf("Expected error message to be %s, got %v", requiredMessage, err)
	}

	schema.AddRule("min=3")

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

// when no custom message is provided, the default message for the failing tag should be returned.
func TestValidate_DefaultMessage(t *testing.T) {
	schema := New()
	schema.AddRule("min=3") // no custom message

	err := schema.Validate("hi") // length 2 < 3

	if err == nil {
		t.Fatalf("Expected error for value shorter than min, got nil")
	}

	// default message includes the tag name; ensure it mentions the failing tag
	if !strings.Contains(err.Error(), "failed 'min' validation") && !strings.Contains(err.Error(), "min") {
		t.Errorf("Expected default error to mention 'min' validation, got: %v", err.Error())
	}
}

// Validate should recover from panics and return an error containing "validation failed".
func TestValidate_PanicRecovered(t *testing.T) {
	v := New()

	// ensure Validate actually invokes the underlying validator by adding a rule
	v.AddRule("required")

	// simulate a panic by nil-ing the internal validator (causes a nil function call panic)
	v.pgValidate = nil

	err := v.Validate("anything")

	if err == nil {
		t.Fatalf("Expected error when underlying validator panics, got nil")
	}

	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("Expected error message to contain 'validation failed', got: %v", err.Error())
	}
}
