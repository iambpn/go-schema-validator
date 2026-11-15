// Package validator checks single values and structs with go-playground/validator rules, and returns your own error messages.
package validator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pgValidator "github.com/go-playground/validator/v10"
)

// paramEscapes turns the escape codes that go-playground allows in rule parameters into the characters it reports.
// "0x2C" stands for a comma and "0x7C" stands for a pipe.
var paramEscapes = strings.NewReplacer("0x2C", ",", "0x7C", "|")

// validationRule is one go-playground rule, such as "min=3", with its custom message.
// An empty message means that a failure of the rule returns the default message.
type validationRule struct {
	rule    string
	message string
}

// FieldValidator holds the rules and custom messages for one value.
// Create it with New, add rules with AddRule or the helper methods, and check a value with ValidateFieldCtx.
type FieldValidator struct {
	fieldRules []validationRule
}

// New returns a FieldValidator without rules.
func New() *FieldValidator {
	return &FieldValidator{
		fieldRules: []validationRule{},
	}
}

// AddRule adds go-playground rules with an optional custom message and returns v for chaining.
//
// rule can hold several rules separated by commas, such as "required,min=3". Each of these rules gets the same message.
// Only the first message is used. Without a message, a failure of the rule returns a default message that names the rule.
// Visit https://github.com/go-playground/validator to see the available rules.
func (v *FieldValidator) AddRule(rule string, message ...string) *FieldValidator {
	msg := ""
	if len(message) > 0 {
		msg = message[0]
	}

	// go-playground splits rules on commas in the same way, so each part is one rule.
	for part := range strings.SplitSeq(rule, ",") {
		v.fieldRules = append(v.fieldRules, validationRule{rule: part, message: msg})
	}

	return v
}

// compileRules joins all rules into one go-playground tag string, such as "required,min=3".
func (v *FieldValidator) compileRules() string {
	var rules []string
	for _, rule := range v.fieldRules {
		rules = append(rules, rule.rule)
	}
	return strings.Join(rules, ",")
}

// ruleTag returns the tag that go-playground reports when the rule fails.
func ruleTag(rule string) string {
	// Rules joined with "|" are reported with the full rule text, for example "eq=a|eq=b".
	if strings.Contains(rule, "|") {
		return rule
	}

	// Other rules are reported without their parameters, for example "min=3" is reported as "min".
	name, _, _ := strings.Cut(rule, "=")
	return name
}

// ruleParam returns the parameter that go-playground reports when the rule fails, for example "3" for "min=3".
func ruleParam(rule string) string {
	_, param, _ := strings.Cut(rule, "=")
	return paramEscapes.Replace(param)
}

// itemDepth returns how many list or map levels deep a failed value is, from the path that go-playground reports.
// For example, the path "[1][0]" means depth 2, and an empty path means the value itself.
// A map key that contains "[" gives a wrong depth. In that case the error gets the default message.
func itemDepth(namespace string) int {
	return strings.Count(namespace, "[")
}

// ValidateFieldCtx checks value against the rules and returns nil if value passes.
//
// If a rule fails, it returns the custom message of that rule, or a default message such as "value failed 'min' validation".
// It panics if a rule is not valid, such as "min=abc", or if pgVal is nil. These are bugs in the calling code.
func (v *FieldValidator) ValidateFieldCtx(ctx context.Context, pgVal *pgValidator.Validate, value any) error {
	return v.validateValue(ctx, pgVal, value, "")
}

// validateValue checks value against the rules and returns the error for the first failed rule.
// fieldName is the struct field name for the default message, or "" for a value that is not a struct field.
func (v *FieldValidator) validateValue(ctx context.Context, pgVal *pgValidator.Validate, value any, fieldName string) error {
	err := pgVal.VarCtx(ctx, value, v.compileRules())
	if err == nil {
		return nil
	}

	var fieldErrs pgValidator.ValidationErrors
	if !errors.As(err, &fieldErrs) {
		return fmt.Errorf("validation failed: %w", err)
	}

	// go-playground reports one failed rule for each value or list item. The first one is reported here.
	return v.errorFor(fieldErrs[0], fieldName)
}

// errorFor returns the custom message of the rule that failed, or the default message if that rule has none.
func (v *FieldValidator) errorFor(fieldErr pgValidator.FieldError, fieldName string) error {
	rule, found := v.failedRule(fieldErr)
	if found && rule.message != "" {
		return errors.New(rule.message)
	}

	return defaultError(fieldErr, fieldName)
}

// failedRule finds the rule that produced fieldErr.
//
// A rule matches if it has the same name and the same "dive" depth as the error.
// A rule that also has the same parameter wins, so "min=2" and "min=4" each get their own message.
// The second result is false if no rule matches.
func (v *FieldValidator) failedRule(fieldErr pgValidator.FieldError) (validationRule, bool) {
	errDepth := itemDepth(fieldErr.Namespace())
	ruleDepth := 0

	var firstNameMatch validationRule
	foundNameMatch := false

	for _, rule := range v.fieldRules {
		// Each "dive" rule moves the next rules one level down, to the items of a list or map.
		if rule.rule == "dive" {
			ruleDepth++
			continue
		}

		if ruleDepth != errDepth || ruleTag(rule.rule) != fieldErr.Tag() {
			continue
		}

		if ruleParam(rule.rule) == fieldErr.Param() {
			return rule, true
		}

		if !foundNameMatch {
			firstNameMatch = rule
			foundNameMatch = true
		}
	}

	return firstNameMatch, foundNameMatch
}

// defaultError returns the error for a failed rule without a custom message.
// For a struct field, the message names the field and item path, such as "field 'Tags[1]' failed 'min' validation".
// For any other value, it starts with "value", such as "value failed 'min' validation".
func defaultError(fieldErr pgValidator.FieldError, fieldName string) error {
	if fieldName == "" {
		return fmt.Errorf("value%s failed '%s' validation", fieldErr.Namespace(), fieldErr.Tag())
	}

	return fmt.Errorf("field '%s%s' failed '%s' validation", fieldName, fieldErr.Namespace(), fieldErr.Tag())
}
