package validator

import (
	"errors"
	"fmt"
	"strings"

	pgValidator "github.com/go-playground/validator/v10"
)

type validationRule struct {
	rule    string
	message string
}

type Validator struct {
	pgValidate *pgValidator.Validate
	rules      []validationRule
}

func New() *Validator {
	return &Validator{
		pgValidate: pgValidator.New(),
		rules:      []validationRule{},
	}
}

// Generic method for adding validation rules
func (v *Validator) AddRule(rule string, message ...string) *Validator {
	msg := ""
	if len(message) > 0 {
		msg = message[0]
	}
	v.rules = append(v.rules, validationRule{rule: rule, message: msg})
	return v
}

// Method to compile rules into a single string
func (v *Validator) compileRules() string {
	var rules []string
	for _, rule := range v.rules {
		rules = append(rules, rule.rule)
	}
	return strings.Join(rules, ",")
}

// Method to validate a non-struct value
func (v *Validator) Validate(value any) (err error) {
	// recover from panics and return them as errors
	defer func() {
		if r := recover(); r != nil {
			errTxt := fmt.Sprintf("%v", r)
			err = fmt.Errorf("validation failed: %w", errors.New(errTxt))
		}
	}()

	var errs pgValidator.ValidationErrors
	err = v.pgValidate.Var(value, v.compileRules())

	if err == nil {
		return nil
	}

	if !errors.As(err, &errs) {
		return fmt.Errorf("validation failed: %w", err)
	}

	for _, e := range errs {
		for _, rule := range v.rules {
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
