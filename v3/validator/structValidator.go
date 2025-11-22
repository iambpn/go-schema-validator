package validator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/iambpn/go-schema-validator/v3/internal/config"
)

type ValidationError struct {
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

type StructValidator[T any] struct {
	rules map[string]*Validator
}

// Method to create a struct validation
func NewStruct[T any]() *StructValidator[T] {
	return &StructValidator[T]{
		rules: make(map[string]*Validator),
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
func (sv *StructValidator[T]) ValidateCtx(ctx context.Context, structVal *T, configs ...config.Config) []ValidationError {
	mergedConfig := config.MergeConfigs(configs...)

	val := reflect.ValueOf(structVal)

	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		valErrs := []ValidationError{}
		valErrs = append(valErrs, ValidationError{Messages: []string{"argument value must be a struct"}})
		return valErrs
	}

	valErrMap := make(map[string][]string)
	for fieldName, validator := range sv.rules {
		field := val.FieldByName(fieldName)

		// check if value is exist and is usable
		if !field.IsValid() {
			if mergedConfig[config.ReturnEarly] {
				return []ValidationError{{
					Field:    fieldName,
					Messages: []string{fmt.Sprintf("field %s does not exist", fieldName)},
				}}
			}

			if arr, ok := valErrMap[fieldName]; ok {
				arr = append(arr, fmt.Sprintf("field %s does not exist", fieldName))
				valErrMap[fieldName] = arr
			} else {
				valErrMap[fieldName] = []string{fmt.Sprintf("field %s does not exist", fieldName)}
			}
		}

		err := validator.ValidateCtx(ctx, field.Interface())
		if err != nil {
			if mergedConfig[config.ReturnEarly] {
				return []ValidationError{{
					Field:    fieldName,
					Messages: []string{fmt.Sprintf("%s", err)},
				}}
			}

			if arr, ok := valErrMap[fieldName]; ok {
				arr = append(arr, fmt.Sprintf("%s", err))
				valErrMap[fieldName] = arr
			} else {
				valErrMap[fieldName] = []string{fmt.Sprintf("%s", err)}
			}
		}
	}

	if len(valErrMap) > 0 {
		valErrs := []ValidationError{}
		for field, messages := range valErrMap {
			valErrs = append(valErrs, ValidationError{
				Field:    field,
				Messages: messages,
			})
		}
		return valErrs
	}

	return nil
}

// Method to validate a struct from any type
func (sv *StructValidator[T]) ValidateAnyCtx(ctx context.Context, structKind any) (*T, []ValidationError) {
	structVal, ok := structKind.(T)

	if !ok {
		return nil, []ValidationError{{
			Messages: []string{"argument value type is not valid"},
		}}
	}

	err := sv.ValidateCtx(ctx, &structVal)

	return &structVal, err
}

// Method to validate a struct from an io.Reader
// To prevent memory leak make sure to close the reader after calling this method
// e.g: defer reader.Close()
func (sv *StructValidator[T]) ValidateIOReaderCtx(ctx context.Context, reader io.Reader) (*T, []ValidationError) {
	var structVal T

	err := json.NewDecoder(reader).Decode(&structVal)
	if err != nil {
		return nil, []ValidationError{{
			Messages: []string{fmt.Sprintf("failed to decode struct from reader: %s", err)},
		}}
	}

	valErr := sv.ValidateCtx(ctx, &structVal)
	if valErr != nil {
		return nil, valErr
	}

	return &structVal, nil
}
