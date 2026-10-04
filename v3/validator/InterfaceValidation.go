package validator

import (
	"context"
	"fmt"
	"io"

	pgValidator "github.com/go-playground/validator/v10"
	"github.com/iambpn/go-schema-validator/v3/config"
)

// ValidationRules Interface for adding validation rule for struct validation
// T is the struct type
// Add the validation rules inside the ValidationRules method
type ValidationRules[T any] interface {
	ValidationRules(sv *StructValidator[T])
}

// CustomValidate Interface for custom struct validation logic
// T is the struct type
// Implement the CustomValidate method to add custom validation logic
type CustomValidate[T any] interface {
	CustomValidate(ctx context.Context, v *pgValidator.Validate, data *T, sv *StructValidator[T]) ValidationErrors
}

// ValidateStructCtx Method validates the data to generic struct that had
// implemented ValidationRules or CustomValidate interface.
//
// data can be either struct, pointer to struct or io.Reader.
// Returns pointer to validated struct and error if validation fails
func ValidateStructCtx[S any](ctx context.Context, v *pgValidator.Validate, data any, configs ...config.Config) (*S, ValidationErrors) {
	val := new(S)

	// check if data is reader
	if reader, ok := data.(io.Reader); ok {
		err := decodeJSON(reader, val)

		if err != nil {
			valErrs := ValidationErrors{
				"error": ValidationError{
					Field:    "error",
					Messages: []string{fmt.Sprintf("failed to decode '%T' struct from provided reader: %s", val, err)},
				},
			}
			return nil, valErrs
		}
	} else if structVal, ok := toStructPointer[S](data); ok {
		val = structVal
	} else {
		valErrs := ValidationErrors{
			"error": ValidationError{
				Field:    "error",
				Messages: []string{fmt.Sprintf("data must be either io.Reader or %T struct", *val)},
			},
		}

		return nil, valErrs
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

	valErrs := ValidationErrors{
		"error": ValidationError{
			Field:    "error",
			Messages: []string{fmt.Sprintf("struct '%T' does not implement neither ValidationRules[%T] nor CustomValidate[%T] interface from (Go-Schema-Validator)", val, *val, *val)},
		},
	}

	return nil, valErrs
}
