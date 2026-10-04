package validator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"

	pgValidator "github.com/go-playground/validator/v10"
	"github.com/iambpn/go-schema-validator/v3/config"
)

// ValidationError represents a validation error for a specific field
type ValidationError struct {
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

// ValidationErrors maps a field name to its ValidationError.
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
	// fieldOrder keeps the field names in the order they were added, so fields are validated in a stable order.
	fieldOrder []string
	// configs holds the configs passed to ValidateStructCtx, so a CustomValidate hook that calls ValidateCtx still uses them.
	configs []config.Config
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

	if _, exists := sv.rules[name]; !exists {
		sv.fieldOrder = append(sv.fieldOrder, name)
	}
	sv.rules[name] = validator

	return sv
}

// ValidateCtx Method to validate a struct with custom validator instance
func (sv *StructValidator[T]) ValidateCtx(ctx context.Context, v *pgValidator.Validate, structVal *T, configs ...config.Config) ValidationErrors {
	mergedConfig := config.MergeConfigs(slices.Concat(sv.configs, configs)...)

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
	for _, fieldName := range sv.fieldOrder {
		err := sv.validateField(ctx, v, val, fieldName)
		if err == nil {
			continue
		}

		valErrs[fieldName] = ValidationError{
			Field:    fieldName,
			Messages: []string{err.Error()},
		}

		if mergedConfig[config.ReturnEarly] {
			return valErrs
		}
	}

	if len(valErrs) != 0 {
		return valErrs
	}

	return nil
}

// validateField validates one field of the struct with the rules added for it.
func (sv *StructValidator[T]) validateField(ctx context.Context, v *pgValidator.Validate, structVal reflect.Value, fieldName string) error {
	field := structVal.FieldByName(fieldName)

	// check if value is exist and is usable
	if !field.IsValid() {
		return fmt.Errorf("field %s does not exist", fieldName)
	}

	// Reflection cannot read the value of an unexported field.
	if !field.CanInterface() {
		return fmt.Errorf("field %s is not exported", fieldName)
	}

	return sv.rules[fieldName].ValidateFieldCtx(ctx, v, field.Interface())
}

// toStructPointer returns a pointer to the struct when the value is either T or a non-nil *T.
func toStructPointer[T any](value any) (*T, bool) {
	switch typed := value.(type) {
	case T:
		return &typed, true
	case *T:
		return typed, typed != nil
	}

	return nil, false
}

// decodeJSON decodes exactly one JSON value from the reader into target.
func decodeJSON(reader io.Reader, target any) error {
	decoder := json.NewDecoder(reader)

	if err := decoder.Decode(target); err != nil {
		return err
	}

	// Any token after the first JSON value means the reader has extra data.
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("unexpected data after the JSON value")
	}

	return nil
}

// ValidateAnyCtx Method to validate a struct from any type with custom validator instance
// structKind can be either T or a pointer to T.
func (sv *StructValidator[T]) ValidateAnyCtx(ctx context.Context, v *pgValidator.Validate, structKind any, configs ...config.Config) (*T, ValidationErrors) {
	structVal, ok := toStructPointer[T](structKind)

	if !ok {
		return nil, ValidationErrors{
			"error": ValidationError{
				Field:    "error",
				Messages: []string{"argument value type is not valid"},
			},
		}
	}

	valErrs := sv.ValidateCtx(ctx, v, structVal, configs...)

	if valErrs != nil {
		return nil, valErrs
	}

	return structVal, nil
}

// ValidateIOReaderCtx Method to validate a struct from an io.Reader
// To prevent memory leak make sure to close the reader after calling this method
// e.g: defer reader.Close()
func (sv *StructValidator[T]) ValidateIOReaderCtx(ctx context.Context, v *pgValidator.Validate, reader io.Reader, configs ...config.Config) (*T, ValidationErrors) {
	var structVal T

	err := decodeJSON(reader, &structVal)
	if err != nil {
		return nil, ValidationErrors{
			"error": ValidationError{
				Field:    "error",
				Messages: []string{fmt.Sprintf("failed to decode struct from reader: %s", err)},
			},
		}
	}

	valErrs := sv.ValidateCtx(ctx, v, &structVal, configs...)
	if valErrs != nil {
		return nil, valErrs
	}

	return &structVal, nil
}
