package validator

import (
	"fmt"
	"reflect"

	pgValidator "github.com/go-playground/validator/v10"
)

type structValidator[T any] struct {
	pgValidate *pgValidator.Validate
	rules      map[string]*Validator
}

// Method to create a struct validation
func NewStruct[T any]() *structValidator[T] {
	return &structValidator[T]{
		pgValidate: pgValidator.New(),
		rules:      make(map[string]*Validator),
	}
}

// Method to add validation rules to struct property
func (sv *structValidator[T]) AddFieldRules(name string, addRules func(*Validator)) *structValidator[T] {
	validator := New()

	addRules(validator)

	sv.rules[name] = validator

	return sv
}

// Method to validate a struct
func (sv *structValidator[T]) ValidateStruct(structVal *T) error {
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
func (sv *structValidator[T]) ValidateAny(structKind any) (T, error) {
	structVal, ok := structKind.(T)

	if !ok {
		return structVal, fmt.Errorf("value must be of type %T", structVal)
	}

	err := sv.ValidateStruct(&structVal)

	return structVal, err
}
