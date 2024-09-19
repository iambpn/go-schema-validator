package schema

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

type validationRule struct {
	rule    string
	message string
}

type Schema struct {
	validator *validator.Validate
	rules     []validationRule
}

func New() *Schema {
	return &Schema{
		validator: validator.New(),
		rules:     []validationRule{},
	}
}

// Generic method for adding validations
// ...message for optional error message
func (s *Schema) AddValidation(rule string, message ...string) *Schema {
	msg := ""
	if len(message) > 0 {
		msg = message[0]
	}
	s.rules = append(s.rules, validationRule{rule: rule, message: msg})
	return s
}

// Method to compile rules into a single string
func (s *Schema) compileRules() string {
	var rules []string
	for _, rule := range s.rules {
		rules = append(rules, rule.rule)
	}
	return strings.Join(rules, ",")
}

func (s *Schema) Struct() *structValidation {
	return &structValidation{
		validator: s.validator,
		rules:     make(map[string]*Schema),
	}
}

func (s *Schema) Validate(value interface{}) (err error) {
	// recover from panics and return them as errors
	defer func() {
		if r := recover(); r != nil {
			errTxt := fmt.Sprintf("%v", r)
			err = fmt.Errorf("validation failed: %w", errors.New(errTxt))
		}
	}()

	var errs validator.ValidationErrors

	err = s.validator.Var(value, s.compileRules())

	if err != nil {
		if errors.As(err, &errs) {
			for _, e := range errs {
				for _, rule := range s.rules {
					if strings.HasPrefix(rule.rule, e.Tag()) {
						if rule.message != "" {
							return errors.New(rule.message)
						}
						break
					}
				}
			}
		}
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}
