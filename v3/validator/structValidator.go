package validator

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	pgValidator "github.com/go-playground/validator/v10"
)

type StructValidator[T any] struct {
	pgValidate *pgValidator.Validate
	rules      map[string]*Validator
}

// Method to create a struct validation
func NewStruct[T any]() *StructValidator[T] {
	return &StructValidator[T]{
		pgValidate: pgValidator.New(),
		rules:      make(map[string]*Validator),
	}
}

// Method to add validation rules to struct property
func (sv *StructValidator[T]) AddFieldRules(name string, addRules func(*Validator)) *StructValidator[T] {
	validator := New()

	addRules(validator)

	sv.rules[name] = validator

	return sv
}

// Method to validate a struct
func (sv *StructValidator[T]) Validate(structVal *T) error {
	val := reflect.ValueOf(structVal)

	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("argument value must be a struct")
	}

	for fieldName, validator := range sv.rules {
		field := val.FieldByName(fieldName)

		// check if value is exist and is usable
		if !field.IsValid() {
			return fmt.Errorf("field %s does not exist", fieldName)
		}

		err := validator.Validate(field.Interface())
		if err != nil {
			return fmt.Errorf("%w", err)
		}
	}
	return nil
}

// Method to validate a struct from any type
func (sv *StructValidator[T]) ValidateAny(structKind any) (*T, error) {
	structVal, ok := structKind.(T)

	if !ok {
		return nil, fmt.Errorf("value must be of type %T", structVal)
	}

	err := sv.Validate(&structVal)

	return &structVal, err
}

// Method to validate a struct from an io.Reader
// To prevent memory leak make sure to close the reader after calling this method
// e.g: defer reader.Close()
func (sv *StructValidator[T]) ValidateIOReader(reader io.Reader) (*T, error) {
	var structVal T

	err := json.NewDecoder(reader).Decode(&structVal)
	if err != nil {
		return nil, fmt.Errorf("failed to decode struct from reader: %w", err)
	}

	err = sv.Validate(&structVal)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return &structVal, nil
}
