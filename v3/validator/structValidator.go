package validator

import (
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"

	pgValidator "github.com/go-playground/validator/v10"
	"github.com/iambpn/go-schema-validator/v3/config"
)

// generalErrorKey is the ValidationErrors key for an error that does not belong to one field, such as bad JSON input.
const generalErrorKey = "error"

// textUnmarshalerType is the type of encoding.TextUnmarshaler. JSON stores values of such types, such as time.Time, as strings.
var textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()

// ValidationError holds the error messages for one struct field.
type ValidationError struct {
	Field    string   `json:"field"`
	Messages []string `json:"messages"`
}

// ValidationErrors maps a field name to its ValidationError.
// The key "error" holds an error that does not belong to one field, such as bad JSON input.
type ValidationErrors map[string]ValidationError

// ToErrorMap returns the messages of each field, keyed by field name.
func (ve ValidationErrors) ToErrorMap() map[string][]string {
	result := make(map[string][]string)
	for field, err := range ve {
		result[field] = err.Messages
	}
	return result
}

// newGeneralError returns ValidationErrors with one message under the "error" key.
func newGeneralError(message string) ValidationErrors {
	return ValidationErrors{
		generalErrorKey: ValidationError{
			Field:    generalErrorKey,
			Messages: []string{message},
		},
	}
}

// StructValidator holds the field rules for the struct type T.
// Create it with NewStruct, add rules with AddFieldRules, and check a struct with one of the Validate methods.
type StructValidator[T any] struct {
	rules map[string]*FieldValidator
	// fieldOrder keeps the field names in the order they were added, so fields are validated in a stable order.
	fieldOrder []string
	// configs holds the configs passed to ValidateStructCtx, so a CustomValidate hook that calls ValidateCtx still uses them.
	configs []config.Config
}

// NewStruct returns a StructValidator for the struct type T without rules.
func NewStruct[T any]() *StructValidator[T] {
	return &StructValidator[T]{
		rules: make(map[string]*FieldValidator),
	}
}

// AddFieldRules sets the rules for the struct field called name and returns sv for chaining.
//
// addRules receives an empty FieldValidator and adds the rules to it.
// If you call AddFieldRules again for the same field, the new rules replace the old ones, and the field keeps its first position.
// It panics if T is not a struct type, or if T has no exported field called name. These are bugs in the calling code.
func (sv *StructValidator[T]) AddFieldRules(name string, addRules func(*FieldValidator)) *StructValidator[T] {
	mustHaveExportedField[T](name)

	validator := New()

	addRules(validator)

	if _, exists := sv.rules[name]; !exists {
		sv.fieldOrder = append(sv.fieldOrder, name)
	}
	sv.rules[name] = validator

	return sv
}

// mustHaveExportedField panics if T is not a struct type, or if T has no exported field called name.
// Exported fields of embedded structs count, because Go promotes them to T.
func mustHaveExportedField[T any](name string) {
	structType := reflect.TypeFor[T]()

	if structType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("validator: %s is not a struct type", structType))
	}

	field, found := structType.FieldByName(name)
	if !found {
		panic(fmt.Sprintf("validator: struct %s has no field %s", structType, name))
	}

	if !field.IsExported() {
		panic(fmt.Sprintf("validator: field %s of struct %s is not exported", name, structType))
	}
}

// ValidateCtx checks the struct that structVal points to and returns nil if all fields pass.
//
// Fields are checked in the order they were added. By default, it stops at the first failed field.
// Pass config.SetReturnEarly(false) to get one error for each failed field.
func (sv *StructValidator[T]) ValidateCtx(ctx context.Context, v *pgValidator.Validate, structVal *T, configs ...config.Config) ValidationErrors {
	mergedConfig := config.MergeConfigs(slices.Concat(sv.configs, configs)...)

	val := reflect.ValueOf(structVal)

	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return newGeneralError("provided value is not a struct")
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

// validateField checks one field of the struct with the rules added for it.
// AddFieldRules already checked that the field exists and is exported.
// If the field sits in a nil embedded struct pointer, the rules check a nil value.
func (sv *StructValidator[T]) validateField(ctx context.Context, v *pgValidator.Validate, structVal reflect.Value, fieldName string) error {
	fieldType, _ := structVal.Type().FieldByName(fieldName)

	var fieldValue any
	field, err := structVal.FieldByIndexErr(fieldType.Index)
	if err == nil {
		fieldValue = field.Interface()
	}

	return sv.rules[fieldName].validateValue(ctx, v, fieldValue, fieldName)
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
// The error message is meant for end users and does not show Go type names.
func decodeJSON(reader io.Reader, target any) error {
	decoder := json.NewDecoder(reader)

	if err := decoder.Decode(target); err != nil {
		return jsonInputError(err)
	}

	// Any token after the first JSON value means the reader has extra data.
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("JSON input must contain only one value")
	}

	return nil
}

// jsonInputError turns an error from the JSON decoder into an error for end users.
func jsonInputError(err error) error {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		return jsonTypeError(typeErr)
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) || errors.Is(err, io.ErrUnexpectedEOF) {
		return errors.New("JSON input is not valid")
	}

	if errors.Is(err, io.EOF) {
		return errors.New("JSON input is empty")
	}

	// Errors from the reader itself and from custom decoders of a field type, such as time.Time, end here.
	return errors.New("JSON input could not be decoded")
}

// jsonTypeError returns the error for a JSON value with the wrong type, such as a number in a string field.
// An empty typeErr.Field means the whole JSON input has the wrong type.
func jsonTypeError(typeErr *json.UnmarshalTypeError) error {
	expected := jsonTypeName(typeErr.Type)

	// A number that does not fit the field, such as 300 for an int8 field, has its value after "number ".
	if expected == "a number" && strings.HasPrefix(typeErr.Value, "number ") {
		return fmt.Errorf("JSON field '%s' has a number that is not allowed", typeErr.Field)
	}

	if typeErr.Field == "" {
		return fmt.Errorf("JSON input must be %s", expected)
	}

	return fmt.Errorf("JSON field '%s' must be %s", typeErr.Field, expected)
}

// jsonTypeName returns the JSON type, with an article, that the Go type expects, such as "a string".
func jsonTypeName(goType reflect.Type) string {
	if goType == nil {
		return "a different type"
	}

	for goType.Kind() == reflect.Pointer {
		goType = goType.Elem()
	}

	pointerType := reflect.PointerTo(goType)
	if pointerType.Implements(textUnmarshalerType) {
		return "a string"
	}

	switch goType.Kind() {
	case reflect.String:
		return "a string"
	case reflect.Bool:
		return "a boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		return "a number"
	case reflect.Struct, reflect.Map:
		return "an object"
	case reflect.Slice:
		// JSON stores a []byte value as a base64 string.
		if goType.Elem().Kind() == reflect.Uint8 {
			return "a string"
		}
		return "an array"
	case reflect.Array:
		return "an array"
	}

	return "a different type"
}

// ValidateAnyCtx checks a struct that is either a T or a non-nil *T.
//
// It returns a pointer to the checked struct if all fields pass. For a *T input, it returns the same pointer.
// For any other input, it returns an "error" entry that says the type is not valid.
func (sv *StructValidator[T]) ValidateAnyCtx(ctx context.Context, v *pgValidator.Validate, structKind any, configs ...config.Config) (*T, ValidationErrors) {
	structVal, ok := toStructPointer[T](structKind)

	if !ok {
		return nil, newGeneralError("argument value type is not valid")
	}

	valErrs := sv.ValidateCtx(ctx, v, structVal, configs...)

	if valErrs != nil {
		return nil, valErrs
	}

	return structVal, nil
}

// ValidateIOReaderCtx decodes one JSON value from reader into a T and checks it.
//
// It returns a pointer to the decoded struct if all fields pass.
// If the JSON input is empty, not valid, has the wrong types, or has extra data after the value, it returns an "error" entry.
// The caller must close the reader, for example with defer reader.Close().
func (sv *StructValidator[T]) ValidateIOReaderCtx(ctx context.Context, v *pgValidator.Validate, reader io.Reader, configs ...config.Config) (*T, ValidationErrors) {
	var structVal T

	err := decodeJSON(reader, &structVal)
	if err != nil {
		return nil, newGeneralError(err.Error())
	}

	valErrs := sv.ValidateCtx(ctx, v, &structVal, configs...)
	if valErrs != nil {
		return nil, valErrs
	}

	return &structVal, nil
}
