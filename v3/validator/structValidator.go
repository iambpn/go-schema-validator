package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	pgValidator "github.com/go-playground/validator/v10"
	"github.com/iambpn/go-schema-validator/v3/internal/config"
)

// ValidationError represents a validation error for a specific field
type ValidationError struct {
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

// ValidationErrors is type alias to slice of ValidationError
type ValidationErrors map[string]ValidationError

// Method to convert ValidationErrors to a map
func (ve ValidationErrors) ToErrorMap() map[string][]string {
	result := make(map[string][]string)
	for field, err := range ve {
		result[field] = err.Messages
	}
	return result
}

type StructValidator[T any] struct {
	rules map[string]*FieldValidator
}

// Method to create a struct validation
func NewStruct[T any]() *StructValidator[T] {
	return &StructValidator[T]{
		rules: make(map[string]*FieldValidator),
	}
}

// Method to add validation rules to struct property
func (sv *StructValidator[T]) AddFieldRules(name string, addRules func(*FieldValidator)) *StructValidator[T] {
	validator := New()

	addRules(validator)

	sv.rules[name] = validator

	return sv
}

// ValidateCtx Method to validate a struct with custom validator instance
func (sv *StructValidator[T]) ValidateCtx(ctx context.Context, v *pgValidator.Validate, structVal *T, configs ...config.Config) ValidationErrors {
	mergedConfig := config.MergeConfigs(configs...)

	val := reflect.ValueOf(structVal)

	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		valErrs := make(ValidationErrors)
		valErrs["error"] = ValidationError{
			Field:    "error",
			Messages: []string{"provided value is not a struct"},
		}
		return valErrs
	}

	valErrs := make(ValidationErrors)
	for fieldName, fv := range sv.rules {
		field := val.FieldByName(fieldName)

		// check if value is exist and is usable
		if !field.IsValid() {
			if mergedConfig[config.ReturnEarly] {
				return ValidationErrors{
					fieldName: ValidationError{
						Field:    fieldName,
						Messages: []string{fmt.Sprintf("field %s does not exist", fieldName)},
					},
				}
			}

			if valErr, ok := valErrs[fieldName]; ok {
				valErr.Messages = append(valErr.Messages, fmt.Sprintf("field %s does not exist", fieldName))
				valErrs[fieldName] = valErr
			} else {
				valErrs[fieldName] = ValidationError{
					Field:    fieldName,
					Messages: []string{fmt.Sprintf("field %s does not exist", fieldName)},
				}
			}
		}

		err := fv.ValidateFieldCtx(ctx, v, field.Interface())
		if err != nil {
			if mergedConfig[config.ReturnEarly] {
				return ValidationErrors{
					fieldName: ValidationError{
						Field:    fieldName,
						Messages: []string{fmt.Sprintf("%s", err)},
					},
				}
			}

			if arr, ok := valErrs[fieldName]; ok {
				arr.Messages = append(arr.Messages, fmt.Sprintf("%s", err))
				valErrs[fieldName] = arr
			} else {
				valErrs[fieldName] = ValidationError{
					Field:    fieldName,
					Messages: []string{fmt.Sprintf("%s", err)},
				}
			}
		}
	}

	if len(valErrs) != 0 {
		return valErrs
	}

	return nil
}

// ValidateAnyCtx Method to validate a struct from any type with custom validator instance
func (sv *StructValidator[T]) ValidateAnyCtx(ctx context.Context, v *pgValidator.Validate, structKind any, configs ...config.Config) (*T, ValidationErrors) {
	structVal, ok := structKind.(T)

	if !ok {
		return nil, ValidationErrors{
			"error": ValidationError{
				Field:    "error",
				Messages: []string{"argument value type is not valid"},
			},
		}
	}

	valErrs := sv.ValidateCtx(ctx, v, &structVal)

	if valErrs != nil {
		return nil, valErrs
	}

	return &structVal, nil
}

// ValidateIOReaderCtx Method to validate a struct from an io.Reader
// To prevent memory leak make sure to close the reader after calling this method
// e.g: defer reader.Close()
func (sv *StructValidator[T]) ValidateIOReaderCtx(ctx context.Context, v *pgValidator.Validate, reader io.Reader, configs ...config.Config) (*T, ValidationErrors) {
	var structVal T

	err := json.NewDecoder(reader).Decode(&structVal)
	if err != nil {
		return nil, ValidationErrors{
			"error": ValidationError{
				Field:    "error",
				Messages: []string{fmt.Sprintf("failed to decode struct from reader: %s", err)},
			},
		}
	}

	valErrs := sv.ValidateCtx(ctx, v, &structVal)
	if valErrs != nil {
		return nil, valErrs
	}

	return &structVal, nil
}
