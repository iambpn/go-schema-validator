package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/iambpn/go-schema-validator/v3/internal/config"
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
	CustomValidate(ctx context.Context, data *T, sv *StructValidator[T]) []ValidationError
}

// ValidateStructCtx Method validates the data to generic struct that had
// implemented ValidationRules or CustomValidate interface.
//
// data can be either struct or io.Reader
// Returns pointer to validated struct and error if validation fails
func ValidateStructCtx[S any](ctx context.Context, data any, configs ...config.Config) (*S, []ValidationError) {
	val := new(S)

	// check if data is reader
	if reader, ok := data.(io.Reader); ok {
		err := json.NewDecoder(reader).Decode(val)

		if err != nil {
			valErrs := []ValidationError{}
			valErr := ValidationError{
				Messages: []string{fmt.Sprintf("failed to decode '%T' struct from provided reader: %s", val, err)},
			}
			valErrs = append(valErrs, valErr)
			return nil, valErrs
		}
	} else if structVal, ok := data.(S); ok {
		val = &structVal
	} else {
		valErrs := []ValidationError{}
		valErr := ValidationError{
			Messages: []string{fmt.Sprintf("data must be either io.Reader or %T struct", *val)},
		}
		valErrs = append(valErrs, valErr)

		return nil, valErrs
	}

	// validate struct with validation rules
	if validateRuleInf, ok := any(val).(ValidationRules[S]); ok {
		sv := NewStruct[S]()
		validateRuleInf.ValidationRules(sv)

		var valErrs []ValidationError = nil
		if validateInf, ok := any(val).(CustomValidate[S]); ok {
			// user specified validation
			valErrs = validateInf.CustomValidate(ctx, val, sv)
		} else {
			// default struct validation
			valErrs = sv.ValidateCtx(ctx, val, configs...)
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
		err := validateInf.CustomValidate(ctx, val, sv)

		if err != nil {
			// return validation error
			return nil, err
		}

		return val, nil
	}

	valErrs := []ValidationError{}
	valErr := ValidationError{
		Messages: []string{fmt.Sprintf("struct '%T' does not implement neither ValidationRules[%T] nor CustomValidate[%T] interface from (Go-Schema-Validator)", val, *val, *val)},
	}
	valErrs = append(valErrs, valErr)
	return nil, valErrs
}
