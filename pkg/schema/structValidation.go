package schema

import (
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
)

type structValidation struct {
	validator *validator.Validate
	rules     map[string]*Schema
}

// Method to add validation to struct property
func (sv *structValidation) Field(name string, schema *Schema) *structValidation {
	sv.rules[name] = schema
	return sv
}

// Method to validate a struct
func (sv *structValidation) Validate(structVal interface{}) error {
	val := reflect.ValueOf(structVal)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("argument value must be a struct")
	}

	for fieldName, schema := range sv.rules {
		field := val.FieldByName(fieldName)

		if !field.IsValid() {
			return fmt.Errorf("field %s does not exist", fieldName)
		}

		err := schema.Validate(field.Interface())
		if err != nil {
			return fmt.Errorf("%w", err)
		}
	}
	return nil
}
