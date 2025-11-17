package validator

import (
	"encoding/json"
	"fmt"
	"io"
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
	CustomValidate(data *T, sv *StructValidator[T]) error
}

// ValidateStruct Method validates the data to generic struct that had
// implemented ValidationRules or CustomValidate interface.
//
// data can be either struct or io.Reader
// Returns pointer to validated struct and error if validation fails
func ValidateStruct[S any](data any) (*S, error) {
	val := new(S)

	// check if data is reader
	if reader, ok := data.(io.Reader); ok {
		err := json.NewDecoder(reader).Decode(val)

		if err != nil {
			return nil, fmt.Errorf("failed to decode '%T' struct from provided reader: %w", val, err)
		}
	} else if structVal, ok := data.(S); ok {
		val = &structVal
	} else {
		return nil, fmt.Errorf("data must be either io.Reader or %T struct", *val)
	}

	// validate struct with validation rules
	if validateRuleInf, ok := any(val).(ValidationRules[S]); ok {
		sv := NewStruct[S]()
		validateRuleInf.ValidationRules(sv)

		var err error
		if validateInf, ok := any(val).(CustomValidate[S]); ok {
			// user specified validation
			err = validateInf.CustomValidate(val, sv)
		} else {
			// default struct validation
			err = sv.Validate(val)
		}

		if err != nil {
			// return validation error
			return nil, err
		}

		return val, nil
	}

	// validate struct with custom validation only
	if validateInf, ok := any(val).(CustomValidate[S]); ok {
		// user specified validation without validation rules
		sv := NewStruct[S]()
		err := validateInf.CustomValidate(val, sv)

		if err != nil {
			// return validation error
			return nil, err
		}

		return val, nil
	}

	return nil, fmt.Errorf("type %T does not implement neither ValidationRule[%T] nor Validate[%T] interface from (Go-Schema-Validator)", val, *val, *val)
}
