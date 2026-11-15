package validator

import (
	"context"
	"fmt"
	"io"

	pgValidator "github.com/go-playground/validator/v10"
	"github.com/iambpn/go-schema-validator/v3/config"
)

// ValidationRules lets the struct type T declare its own field rules.
// Implement the ValidationRules method and add the rules to sv with AddFieldRules.
// ValidateStructCtx calls it each time it checks a T.
type ValidationRules[T any] interface {
	ValidationRules(sv *StructValidator[T])
}

// CustomValidate lets the struct type T run its own checks, such as checks that compare two fields.
//
// ValidateStructCtx calls CustomValidate with the decoded struct in data, and with sv.
// If T also implements ValidationRules, sv holds those rules, and CustomValidate must call sv.ValidateCtx to run them.
// Return nil if the struct passes.
type CustomValidate[T any] interface {
	CustomValidate(ctx context.Context, v *pgValidator.Validate, data *T, sv *StructValidator[T]) ValidationErrors
}

// ValidateStructCtx checks data as a struct of type S that implements ValidationRules, CustomValidate, or both.
//
// data can be an S, a non-nil *S, or an io.Reader with one JSON value. For a *S input, it returns the same pointer.
// It returns a pointer to the checked struct if the struct passes.
// It returns an "error" entry if data has another type, if the JSON input is not usable, or if S implements neither interface.
func ValidateStructCtx[S any](ctx context.Context, v *pgValidator.Validate, data any, configs ...config.Config) (*S, ValidationErrors) {
	val, dataErrs := structFromData[S](data)
	if dataErrs != nil {
		return nil, dataErrs
	}

	// validate struct with validation rules
	if validateRuleInf, ok := any(val).(ValidationRules[S]); ok {
		sv := NewStruct[S]()
		sv.configs = configs
		validateRuleInf.ValidationRules(sv)

		var valErrs ValidationErrors = nil
		if validateInf, ok := any(val).(CustomValidate[S]); ok {
			// user specified validation
			valErrs = validateInf.CustomValidate(ctx, v, val, sv)
		} else {
			// default struct validation
			valErrs = sv.ValidateCtx(ctx, v, val, configs...)
		}

		if valErrs != nil {
			// return validation error
			return nil, valErrs
		}

		return val, nil
	}

	// validate struct with custom validation only
	if validateInf, ok := any(val).(CustomValidate[S]); ok {
		// user specified validation without validation rules
		sv := NewStruct[S]()
		sv.configs = configs
		valErrs := validateInf.CustomValidate(ctx, v, val, sv)

		if valErrs != nil {
			// return validation error
			return nil, valErrs
		}

		return val, nil
	}

	return nil, newGeneralError(fmt.Sprintf("struct '%T' implements neither ValidationRules[%T] nor CustomValidate[%T] from go-schema-validator", val, *val, *val))
}

// structFromData returns a pointer to the struct in data.
//
// data can be an S, a non-nil *S, or an io.Reader with one JSON value.
// The struct types are checked first, so a struct with a Read method is checked as a struct and is not read as JSON.
func structFromData[S any](data any) (*S, ValidationErrors) {
	switch typed := data.(type) {
	case S:
		return &typed, nil
	case *S:
		if typed != nil {
			return typed, nil
		}
	case io.Reader:
		structVal := new(S)

		if err := decodeJSON(typed, structVal); err != nil {
			return nil, newGeneralError(err.Error())
		}

		return structVal, nil
	}

	var zero S
	return nil, newGeneralError(fmt.Sprintf("data must be either io.Reader or %T struct", zero))
}
