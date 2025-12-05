package validator

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pgValidator "github.com/go-playground/validator/v10"
)

type validationRule struct {
	rule    string
	message string
}

type FieldValidator struct {
	fieldRules []validationRule
}

func New() *FieldValidator {
	return &FieldValidator{
		fieldRules: []validationRule{},
	}
}

// Generic method for adding validation rules.
//
// Visit https://github.com/go-playground/validator to know more about available rules
func (v *FieldValidator) AddRule(rule string, message ...string) *FieldValidator {
	msg := ""
	if len(message) > 0 {
		msg = message[0]
	}
	v.fieldRules = append(v.fieldRules, validationRule{rule: rule, message: msg})
	return v
}

// Method to compile rules into a single string
func (v *FieldValidator) compileRules() string {
	var rules []string
	for _, rule := range v.fieldRules {
		rules = append(rules, rule.rule)
	}
	return strings.Join(rules, ",")
}

// ValidateFieldCtxWithValidator Method to validate a non-struct value with a provided validator instance
// This method recovers from panics and returns them as errors
func (v *FieldValidator) ValidateFieldCtx(ctx context.Context, pgVal *pgValidator.Validate, value any) (err error) {
	// recover from panics and return them as errors
	defer func() {
		if r := recover(); r != nil {
			errTxt := fmt.Sprintf("%v", r)
			err = fmt.Errorf("validation failed: %w", errors.New(errTxt))
		}
	}()

	var errs pgValidator.ValidationErrors
	err = pgVal.VarCtx(ctx, value, v.compileRules())

	if err == nil {
		return nil
	}

	if !errors.As(err, &errs) {
		return fmt.Errorf("validation failed: %w", err)
	}

	for _, e := range errs {
		for _, rule := range v.fieldRules {
			if strings.HasPrefix(rule.rule, e.Tag()) {
				if rule.message != "" {
					return errors.New(rule.message)
				}

				// default error message: specify validation tag, user value and field
				return fmt.Errorf("field '%s' failed '%s' validation", e.Field(), e.Tag())
			}
		}
	}

	// default error message for just in case
	return fmt.Errorf("validation failed: %w", err)
}
