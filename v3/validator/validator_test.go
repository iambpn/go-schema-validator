package validator

import (
	"fmt"
	"testing"

	pgValidator "github.com/go-playground/validator/v10"
)

// TestNew checks that New returns a FieldValidator.
func TestNew(t *testing.T) {
	schema := New()

	if schema == nil {
		t.Errorf("Expected schema to be created, got %v", schema)
	}
}

// TestAddRule checks that AddRule stores the rule and only the first message.
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

// TestCompileRules checks that compileRules joins the rules with commas.
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

// TestValidatorValidate checks that ValidateFieldCtx passes without rules, and returns errors as rules are added.
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

// TestValidate_DefaultMessage checks the default message for a failed rule without a custom message.
func TestValidate_DefaultMessage(t *testing.T) {
	schema := New()
	v := pgValidator.New()
	schema.AddRule("min=3") // no custom message

	err := schema.ValidateFieldCtx(t.Context(), v, "hi") // length 2 < 3

	if err == nil {
		t.Fatalf("Expected error for value shorter than min, got nil")
	}

	expected := "value failed 'min' validation"
	if err.Error() != expected {
		t.Errorf("Expected default error '%s', got: %v", expected, err)
	}
}

// assertPanics fails the test unless run panics.
func assertPanics(t *testing.T, run func()) {
	t.Helper()

	defer func() {
		if recover() == nil {
			t.Fatalf("Expected a panic, got none")
		}
	}()

	run()
}

// TestValidate_PanicsOnNilValidator checks that ValidateFieldCtx panics when the go-playground validator is nil.
func TestValidate_PanicsOnNilValidator(t *testing.T) {
	fv := New().AddRule("required")
	var v *pgValidator.Validate

	assertPanics(t, func() {
		_ = fv.ValidateFieldCtx(t.Context(), v, "anything")
	})
}

// TestValidate_PanicsOnInvalidRule checks that ValidateFieldCtx panics for a rule with a bad parameter.
func TestValidate_PanicsOnInvalidRule(t *testing.T) {
	fv := New().AddRule("min=abc", "custom message")
	v := pgValidator.New()

	assertPanics(t, func() {
		_ = fv.ValidateFieldCtx(t.Context(), v, "anything")
	})
}

// TestValidate_ReturnsMessageOfFailedRule checks that the custom message of the failed rule is returned, not the message of another rule.
func TestValidate_ReturnsMessageOfFailedRule(t *testing.T) {
	fv := New()
	fv.AddRule("min=5", "custom min message")
	fv.AddRule("max=10", "custom max message")

	v := pgValidator.New()

	err := fv.ValidateFieldCtx(t.Context(), v, "ab")

	if err == nil || err.Error() != "custom min message" {
		t.Errorf("Expected 'custom min message', got: %v", err)
	}
}

// TestValidate_DefaultMessageForNonStringValue checks the default message when the email rule fails for a number.
func TestValidate_DefaultMessageForNonStringValue(t *testing.T) {
	fv := New()
	fv.AddRule("email")

	v := pgValidator.New()

	err := fv.ValidateFieldCtx(t.Context(), v, 12345)

	expected := "value failed 'email' validation"
	if err == nil || err.Error() != expected {
		t.Errorf("Expected '%s', got: %v", expected, err)
	}
}

// TestValidate_ReturnsCustomMaxMessage checks that a failed max rule returns its custom message.
func TestValidate_ReturnsCustomMaxMessage(t *testing.T) {
	fv := New()
	// only add custom message for 'max' rule
	fv.AddRule("max=5", "custom max message")

	v := pgValidator.New()

	err := fv.ValidateFieldCtx(t.Context(), v, "toolongstring")

	if err == nil || err.Error() != "custom max message" {
		t.Errorf("Expected 'custom max message', got: %v", err)
	}
}

// TestValidate_MessageMatchesExactRule checks that a rule name that starts with another rule name, such as lte and lt, gets the right message.
func TestValidate_MessageMatchesExactRule(t *testing.T) {
	v := pgValidator.New()

	// 'lte' starts with 'lt', but only the failed 'lt' rule should give its message
	err := New().
		AddRule("lte=100", "lte message").
		AddRule("lt=5", "lt message").
		ValidateFieldCtx(t.Context(), v, 50)

	if err == nil || err.Error() != "lt message" {
		t.Errorf("Expected 'lt message', got: %v", err)
	}

	err = New().
		AddRule("alphanum", "alphanum message").
		AddRule("alpha", "alpha message").
		ValidateFieldCtx(t.Context(), v, "abc1")

	if err == nil || err.Error() != "alpha message" {
		t.Errorf("Expected 'alpha message', got: %v", err)
	}
}

// TestValidate_MessageForOrRule checks that a rule joined with "|" returns its custom message.
func TestValidate_MessageForOrRule(t *testing.T) {
	v := pgValidator.New()

	err := New().
		AddRule("eq=a|eq=b", "must be a or b").
		ValidateFieldCtx(t.Context(), v, "c")

	if err == nil || err.Error() != "must be a or b" {
		t.Errorf("Expected 'must be a or b', got: %v", err)
	}
}

// TestAddRule_SplitsCommaSeparatedRules checks that AddRule splits a comma-separated rule string and gives each rule the message.
func TestAddRule_SplitsCommaSeparatedRules(t *testing.T) {
	schema := New().AddRule("required,min=3", "custom message")

	if len(schema.fieldRules) != 2 {
		t.Fatalf("Expected 2 rules, got %v", schema.fieldRules)
	}

	for _, rule := range schema.fieldRules {
		if rule.message != "custom message" {
			t.Errorf("Expected each rule to keep the custom message, got %v", schema.fieldRules)
		}
	}
}

// TestValidate_MessageForCommaSeparatedRules checks that a comma-separated rule string returns its custom message.
func TestValidate_MessageForCommaSeparatedRules(t *testing.T) {
	v := pgValidator.New()

	err := New().
		AddRule("required,min=3", "custom message").
		ValidateFieldCtx(t.Context(), v, "ab")

	if err == nil || err.Error() != "custom message" {
		t.Errorf("Expected 'custom message', got: %v", err)
	}
}

// TestValidate_MessageForSameRuleNameWithDifferentParams checks that two rules with the same name, such as min=2 and min=4, each get their own message.
func TestValidate_MessageForSameRuleNameWithDifferentParams(t *testing.T) {
	v := pgValidator.New()

	err := New().
		Min(2, "min 2 message").
		Max(5, "max message").
		Min(4, "min 4 message").
		ValidateFieldCtx(t.Context(), v, "abc")

	if err == nil || err.Error() != "min 4 message" {
		t.Errorf("Expected 'min 4 message', got: %v", err)
	}
}

// TestValidate_MessageForDiveRules checks that a rule before "dive" and a rule after it each get their own message.
func TestValidate_MessageForDiveRules(t *testing.T) {
	v := pgValidator.New()

	schema := New().
		AddRule("min=1", "need at least one tag").
		AddRule("dive").
		AddRule("min=3", "each tag needs 3 characters")

	err := schema.ValidateFieldCtx(t.Context(), v, []string{})

	if err == nil || err.Error() != "need at least one tag" {
		t.Errorf("Expected 'need at least one tag', got: %v", err)
	}

	err = schema.ValidateFieldCtx(t.Context(), v, []string{"abc", "ab"})

	if err == nil || err.Error() != "each tag needs 3 characters" {
		t.Errorf("Expected 'each tag needs 3 characters', got: %v", err)
	}
}

// TestValidate_DefaultMessageForDiveItem checks that the default message for a list item shows the item index.
func TestValidate_DefaultMessageForDiveItem(t *testing.T) {
	v := pgValidator.New()

	err := New().
		AddRule("dive,min=3").
		ValidateFieldCtx(t.Context(), v, []string{"abc", "ab"})

	expected := "value[1] failed 'min' validation"
	if err == nil || err.Error() != expected {
		t.Errorf("Expected '%s', got: %v", expected, err)
	}
}

// TestValidate_MessageForEscapedParam checks that a rule with an escaped comma in its parameter returns its custom message.
func TestValidate_MessageForEscapedParam(t *testing.T) {
	v := pgValidator.New()

	err := New().
		AddRule("contains=0x2C", "must contain a comma").
		ValidateFieldCtx(t.Context(), v, "no comma")

	if err == nil || err.Error() != "must contain a comma" {
		t.Errorf("Expected 'must contain a comma', got: %v", err)
	}
}
