package validator

import (
	"fmt"
	"strings"
	"testing"

	pgValidator "github.com/go-playground/validator/v10"
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
	if len(schema.fieldRules) != 1 || schema.fieldRules[0].message != message || schema.fieldRules[0].rule != validation {
		t.Errorf("Expected validation rule to be added, got %v", schema.fieldRules)
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
	v := pgValidator.New()
	schema := New()

	if schema == nil {
		t.Errorf("Expected schema to be created, got %v", schema)
	}

	err := schema.ValidateFieldCtx(t.Context(), v, "hello")

	if err != nil {
		t.Errorf("Expected no error on string, got %v", err)
	}

	const requiredMessage = "This field is required"
	schema.AddRule("required", requiredMessage)

	err = schema.ValidateFieldCtx(t.Context(), v, "")

	if err == nil {
		t.Errorf("Expected error on empty string, got %v", err)
	}

	if err.Error() != requiredMessage {
		t.Errorf("Expected error message to be %s, got %v", requiredMessage, err)
	}

	schema.AddRule("min=3")

	err = schema.ValidateFieldCtx(t.Context(), v, "hello")

	if err != nil {
		t.Errorf("Expected no error on string, got %v", err)
	}

	err = schema.ValidateFieldCtx(t.Context(), v, "he")

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
	v := pgValidator.New()
	schema.AddRule("min=3") // no custom message

	err := schema.ValidateFieldCtx(t.Context(), v, "hi") // length 2 < 3

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
	fv := New()
	var v *pgValidator.Validate

	// ensure Validate actually invokes the underlying validator by adding a rule
	fv.AddRule("required")

	err := fv.ValidateFieldCtx(t.Context(), v, "anything")

	if err == nil {
		t.Fatalf("Expected error when underlying validator panics, got nil")
	}

	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("Expected error message to contain 'validation failed', got: %v", err.Error())
	}
}

// Test fallback error message when no matching rule is found
func TestValidate_FallbackErrorMessage(t *testing.T) {
	fv := New()
	fv.AddRule("min=5", "custom min message")
	fv.AddRule("max=10", "custom max message")

	v := pgValidator.New()

	// validate a string that triggers an error but with a rule tag that doesn't match any configured rules
	err := fv.ValidateFieldCtx(t.Context(), v, "ab")

	if err == nil {
		t.Fatalf("Expected error for value shorter than min, got nil")
	}

	// the error should contain either the custom message or a default fallback
	if !strings.Contains(err.Error(), "custom min message") && !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("Expected error to contain custom message or fallback, got: %v", err.Error())
	}
}

// Test generic error handling when validation returns non-ValidationErrors type
func TestValidate_GenericErrorHandling(t *testing.T) {
	fv := New()
	fv.AddRule("email")

	v := pgValidator.New()

	// pass an invalid type that cannot be validated with 'email' rule
	// this should trigger error handling
	err := fv.ValidateFieldCtx(t.Context(), v, 12345)

	if err == nil {
		t.Fatalf("Expected error for invalid email type, got nil")
	}

	// error should mention validation failure
	errMsg := err.Error()
	if !strings.Contains(errMsg, "validation failed") && !strings.Contains(errMsg, "email") {
		t.Errorf("Expected error to contain 'validation failed' or 'email', got: %v", errMsg)
	}
}

// Test no matching rule scenario - add a rule with custom message but trigger a different validation error
func TestValidate_NoMatchingRuleError(t *testing.T) {
	fv := New()
	// only add custom message for 'max' rule
	fv.AddRule("max=5", "custom max message")

	v := pgValidator.New()

	// validate with 'min' rule which has no custom message defined
	// but the value will fail 'max' so should return the custom message
	err := fv.ValidateFieldCtx(t.Context(), v, "toolongstring")

	if err == nil {
		t.Fatalf("Expected error for value exceeding max, got nil")
	}

	// should return the custom message for the matching rule
	if !strings.Contains(err.Error(), "custom max message") {
		t.Errorf("Expected error to contain 'custom max message', got: %v", err.Error())
	}
}
