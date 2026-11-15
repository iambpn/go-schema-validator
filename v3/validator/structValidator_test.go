package validator

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	pgValidator "github.com/go-playground/validator/v10"
	"github.com/iambpn/go-schema-validator/v3/config"
)

// User is the struct that most tests validate.
type User struct {
	Name string
	Age  int
}

// TestField checks that AddFieldRules stores one rule set for each field.
func TestField(t *testing.T) {
	structVal := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}

	nameField := "Name"
	ageField := "Age"

	nameRequiredMessage := "Name is required"
	ageGt18Message := "Age must be at least 18"

	structVal.AddFieldRules(nameField, func(v *FieldValidator) {
		v.AddRule("required", nameRequiredMessage)
	})
	structVal.AddFieldRules(ageField, func(v *FieldValidator) {
		v.AddRule("min=18", ageGt18Message)
	})

	if len(structVal.rules) != 2 {
		t.Errorf("Expected 2 fields, got %v", len(structVal.rules))
	}

	for fieldName := range structVal.rules {
		if fieldName != nameField && fieldName != ageField {
			t.Errorf("Expected field name to be %s or %s, got %s", nameField, ageField, fieldName)
		}
	}
}

// assertFirstErrorMessage validates val with sv and fails the test unless field has the error message errMsg.
func assertFirstErrorMessage[T any](val *T, sv *StructValidator[T], v *pgValidator.Validate, field string, errMsg string, t *testing.T) {
	t.Helper()

	err := sv.ValidateCtx(t.Context(), v, val)

	if err == nil {
		t.Fatalf("Expected validation to fail, got nil")
	}

	if _, ok := err[field]; !ok {
		t.Fatalf("Expected error field to be '%s', but it was missing from the error map", field)
	}

	if err[field].Messages[0] != errMsg {
		t.Fatalf("Expected error message to be '%s', got '%s'", errMsg, err[field].Messages[0])
	}

	if err[field].Field != field {
		t.Fatalf("Expected error field to be '%s', got '%s'", field, err[field].Field)
	}
}

// TestStructValidatorValidateChainedRules checks that each field returns the message of its first failed rule.
func TestStructValidatorValidateChainedRules(t *testing.T) {
	structVal := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	nameField := "Name"
	nameMin2Message := "Name must be minimum of 2 characters"
	ageField := "Age"

	nameRequiredMessage := "Name is required"
	ageGt18Message := "Age must be at least 18"

	structVal.AddFieldRules(nameField, func(v *FieldValidator) {
		v.AddRule("required", nameRequiredMessage)
		v.AddRule("min=2", nameMin2Message)
	})
	structVal.AddFieldRules(ageField, func(v *FieldValidator) {
		v.AddRule("min=18", ageGt18Message)
	})

	assertFirstErrorMessage(&User{
		Name: "",
		Age:  18,
	}, &structVal, v, nameField, nameRequiredMessage, t)

	assertFirstErrorMessage(&User{
		Name: "a",
		Age:  18,
	}, &structVal, v, nameField, nameMin2Message, t)

	assertFirstErrorMessage(&User{
		Name: "aa",
		Age:  0,
	}, &structVal, v, ageField, ageGt18Message, t)
}

// TestStructValidatorValidate checks that ValidateCtx accepts a valid User and returns the right message for each failed field.
func TestStructValidatorValidate(t *testing.T) {
	structVal := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	nameField := "Name"
	ageField := "Age"

	nameRequiredMessage := "Name is required"
	ageGt18Message := "Age must be at least 18"

	structVal.AddFieldRules(nameField, func(v *FieldValidator) {
		v.AddRule("required", nameRequiredMessage)
	})
	structVal.AddFieldRules(ageField, func(v *FieldValidator) {
		v.AddRule("min=18", ageGt18Message)
	})

	_, err := structVal.ValidateAnyCtx(t.Context(), v, "")

	if err == nil {
		t.Errorf("Expected error on empty struct, got %v", err)
	}

	noUser := User{
		Name: "John",
		Age:  18,
	}

	err = structVal.ValidateCtx(t.Context(), v, &noUser)

	if err != nil {
		t.Errorf("Expected no error on valid struct, got %v", err)
	}

	noNameUser := User{
		Name: "",
		Age:  18,
	}
	assertFirstErrorMessage(&noNameUser, &structVal, v, nameField, nameRequiredMessage, t)

	noAgeUser := User{
		Name: "John",
		Age:  0,
	}
	assertFirstErrorMessage(&noAgeUser, &structVal, v, ageField, ageGt18Message, t)
}

// TestNewStruct checks that NewStruct returns a *StructValidator.
func TestNewStruct(t *testing.T) {
	schema := NewStruct[User]()

	if schema == nil {
		t.Errorf("Expected struct to be created, got %v", schema)
	}

	if reflect.TypeOf(schema) != reflect.TypeOf(&StructValidator[User]{}) {
		t.Errorf("Expected struct to be of type StructValidator, got %v", reflect.TypeOf(schema))
	}
}

// TestValidateAnySuccess checks that ValidateAnyCtx accepts a valid struct value and returns an equal struct.
func TestValidateAnySuccess(t *testing.T) {
	structVal := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	noUser := User{
		Name: "Alice",
		Age:  25,
	}

	ret, err := structVal.ValidateAnyCtx(t.Context(), v, noUser)

	if err != nil {
		t.Fatalf("Expected no error on ValidateAny with correct type, got %v", err)
	}

	if *ret != noUser {
		t.Fatalf("Expected returned value to equal input, got %v", ret)
	}
}

// TestValidateStruct_NonStruct checks that ValidateCtx returns an "error" entry when T is not a struct type.
func TestValidateStruct_NonStruct(t *testing.T) {
	sv := StructValidator[any]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	var anyValue any = "not a struct"

	expected := "provided value is not a struct"
	assertFirstErrorMessage(&anyValue, &sv, v, "error", expected, t)
}

// TestAddFieldRules_PanicsForUnknownField checks that AddFieldRules panics for a field name that User does not have.
func TestAddFieldRules_PanicsForUnknownField(t *testing.T) {
	sv := NewStruct[User]()

	assertPanics(t, func() {
		sv.AddFieldRules("UnknownField", func(v *FieldValidator) {
			v.AddRule("required", "UnknownField is required")
		})
	})
}

// TestAddFieldRules_PanicsForNonStructType checks that AddFieldRules panics when T is not a struct type.
func TestAddFieldRules_PanicsForNonStructType(t *testing.T) {
	sv := NewStruct[string]()

	assertPanics(t, func() {
		sv.AddFieldRules("Name", func(v *FieldValidator) {
			v.AddRule("required")
		})
	})
}

// TestValidateStruct_PointerInput checks that ValidateCtx accepts a pointer to a valid struct.
func TestValidateStruct_PointerInput(t *testing.T) {
	sv := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	sv.AddFieldRules("Name", func(v *FieldValidator) {
		v.AddRule("required", "Name is required")
	})

	user := User{
		Name: "PointerUser",
		Age:  40,
	}

	// pass pointer to struct
	valErrs := sv.ValidateCtx(t.Context(), v, &user)

	if valErrs != nil {
		t.Fatalf("Expected no error validating pointer to struct, got %v", valErrs.ToErrorMap())
	}
}

// TestValidateStruct_IoReaderInput checks that ValidateIOReaderCtx decodes valid JSON and returns an equal struct.
func TestValidateStruct_IoReaderInput(t *testing.T) {
	sv := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	sv.AddFieldRules("Name", func(v *FieldValidator) {
		v.AddRule("required", "Name is required")
	})

	// convert struct to io.Reader (which is not a struct)
	originalUser := User{
		Name: "ReaderUser",
		Age:  50,
	}
	bytes, err := json.Marshal(originalUser)

	if err != nil {
		t.Fatalf("Failed to marshal struct to JSON: %v", err)
	}

	reader := strings.NewReader(string(bytes))

	user, valErr := sv.ValidateIOReaderCtx(t.Context(), v, reader)

	if valErr != nil {
		t.Fatalf("Expected nil when validating io.Reader, got %v", valErr.ToErrorMap())
	}

	if (*user) != originalUser {
		t.Fatalf("Expected returned user to be same as Original User, got %v", user)
	}
}

// TestValidateStructMultipleErrors checks that ReturnEarly set to false returns one error for each failed field.
func TestValidateStructMultipleErrors(t *testing.T) {
	sv := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	sv.AddFieldRules("Name", func(v *FieldValidator) {
		v.AddRule("required", "Name is required")
		v.AddRule("min=3", "Name must be at least 3 characters")
	})
	sv.AddFieldRules("Age", func(v *FieldValidator) {
		v.AddRule("min=18", "Age must be at least 18")
	})

	user := User{
		Name: "Al",
		Age:  15,
	}

	err := sv.ValidateCtx(t.Context(), v, &user, config.SetReturnEarly(false))

	if err == nil {
		t.Fatalf("Expected validation to fail, got nil")
	}

	if len(err) != 2 {
		t.Fatalf("Expected 2 validation errors, got %d", len(err))
	}

	for _, valErr := range err {
		switch valErr.Field {
		case "Name":
			if len(valErr.Messages) != 1 || valErr.Messages[0] != "Name must be at least 3 characters" {
				t.Fatalf("Unexpected error messages for Name field: %v", valErr.Messages)
			}
		case "Age":
			if len(valErr.Messages) != 1 || valErr.Messages[0] != "Age must be at least 18" {
				t.Fatalf("Unexpected error messages for Age field: %v", valErr.Messages)
			}
		default:
			t.Fatalf("Unexpected field in validation errors: %s", valErr.Field)
		}
	}
}

// TestValidateValidationErrorsToMap checks that ToErrorMap returns the messages of each failed field.
func TestValidateValidationErrorsToMap(t *testing.T) {
	sv := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	sv.AddFieldRules("Name", func(v *FieldValidator) {
		v.AddRule("required", "Name is required")
		v.AddRule("min=3", "Name must be at least 3 characters")
	})
	sv.AddFieldRules("Age", func(v *FieldValidator) {
		v.AddRule("min=18", "Age must be at least 18")
	})

	user := User{
		Name: "Al",
		Age:  15,
	}

	err := sv.ValidateCtx(t.Context(), v, &user, config.SetReturnEarly(false))

	if err == nil {
		t.Fatalf("Expected validation to fail, got nil")
	}

	mapErr := err.ToErrorMap()

	if len(mapErr) != 2 {
		t.Fatalf("Expected 2 fields in error map, got %d", len(mapErr))
	}

	if len(mapErr["Name"]) != 1 || mapErr["Name"][0] != "Name must be at least 3 characters" {
		t.Fatalf("Unexpected error messages for Name field: %v", mapErr["Name"])
	}

	if len(mapErr["Age"]) != 1 || mapErr["Age"][0] != "Age must be at least 18" {
		t.Fatalf("Unexpected error messages for Age field: %v", mapErr["Age"])
	}
}

// TestValidateStruct_ReturnEarlyOnValidationError checks that ReturnEarly set to true returns only the first failed field.
func TestValidateStruct_ReturnEarlyOnValidationError(t *testing.T) {
	sv := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	// add multiple field rules
	sv.AddFieldRules("Name", func(v *FieldValidator) {
		v.AddRule("required", "Name is required")
		v.AddRule("min=3", "Name must be at least 3 characters")
	})
	sv.AddFieldRules("Age", func(v *FieldValidator) {
		v.AddRule("min=18", "Age must be at least 18")
	})

	user := User{
		Name: "Al",
		Age:  15,
	}

	// with ReturnEarly=true, should return on first validation error
	err := sv.ValidateCtx(t.Context(), v, &user, config.SetReturnEarly(true))

	if err == nil {
		t.Fatalf("Expected validation to fail, got nil")
	}

	if len(err) != 1 {
		t.Fatalf("Expected 1 error with ReturnEarly=true, got %d errors", len(err))
	}

	// should return the first error encountered
	if err["Name"].Field != "Name" {
		t.Fatalf("Expected first error to be for Name field, got '%s'", err["Name"].Field)
	}
}

// TestValidateStruct_ReturnEarlyFalseCollectsAllErrors checks that ReturnEarly set to false returns errors for all failed fields.
func TestValidateStruct_ReturnEarlyFalseCollectsAllErrors(t *testing.T) {
	sv := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	sv.AddFieldRules("Name", func(v *FieldValidator) {
		v.AddRule("required", "Name is required")
	})
	sv.AddFieldRules("Age", func(v *FieldValidator) {
		v.AddRule("min=18", "Age must be at least 18")
	})

	user := User{
		Name: "",
		Age:  10,
	}

	// with ReturnEarly=false (default), should collect all errors
	err := sv.ValidateCtx(t.Context(), v, &user, config.SetReturnEarly(false))

	if err == nil {
		t.Fatalf("Expected validation to fail, got nil")
	}

	if len(err) != 2 {
		t.Fatalf("Expected 2 errors with ReturnEarly=false, got %d errors", len(err))
	}

	// verify both errors are present
	errorMap := make(map[string]bool)
	for _, e := range err {
		errorMap[e.Field] = true
	}

	if !errorMap["Name"] || !errorMap["Age"] {
		t.Fatalf("Expected errors for both Name and Age fields")
	}
}

// TestValidateIOReaderCtx_WithValidationError checks that ValidateIOReaderCtx returns field errors and no struct for valid JSON with bad values.
func TestValidateIOReaderCtx_WithValidationError(t *testing.T) {
	sv := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	sv.AddFieldRules("Name", func(v *FieldValidator) {
		v.AddRule("required", "Name is required")
	})
	sv.AddFieldRules("Age", func(v *FieldValidator) {
		v.AddRule("min=18", "Age must be at least 18")
	})

	// valid JSON that decodes successfully but fails validation
	invalidJSON := `{"Name": "", "Age": 15}`
	reader := strings.NewReader(invalidJSON)

	user, err := sv.ValidateIOReaderCtx(t.Context(), v, reader)

	if err == nil {
		t.Fatalf("Expected validation to fail when struct is invalid, got nil")
	}

	if user != nil {
		t.Fatalf("Expected user to be nil when validation fails, got %v", user)
	}

	// verify errors are present
	if len(err) == 0 {
		t.Fatalf("Expected at least 1 validation error, got 0")
	}
}

// TestValidateIOReaderCtx_WithValidJSON checks that ValidateIOReaderCtx returns the decoded struct for valid JSON with good values.
func TestValidateIOReaderCtx_WithValidJSON(t *testing.T) {
	sv := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	sv.AddFieldRules("Name", func(v *FieldValidator) {
		v.AddRule("required", "Name is required")
	})
	sv.AddFieldRules("Age", func(v *FieldValidator) {
		v.AddRule("min=18", "Age must be at least 18")
	})

	// valid JSON with valid data
	validJSON := `{"Name": "John", "Age": 25}`
	reader := strings.NewReader(validJSON)

	user, err := sv.ValidateIOReaderCtx(t.Context(), v, reader)

	if err != nil {
		t.Fatalf("Expected validation to pass, got error: %v", err)
	}

	if user == nil {
		t.Fatalf("Expected user to not be nil when validation passes")
	}

	if user.Name != "John" || user.Age != 25 {
		t.Fatalf("Expected user to have Name='John' and Age=25, got %v", user)
	}
}

// TestValidateIOReaderCtx_WithInvalidJSON checks that ValidateIOReaderCtx returns "JSON input is not valid" for a syntax error.
func TestValidateIOReaderCtx_WithInvalidJSON(t *testing.T) {
	sv := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	sv.AddFieldRules("Name", func(v *FieldValidator) {
		v.AddRule("required", "Name is required")
	})

	// invalid JSON that fails to decode
	invalidJSON := `{"Name": "John", invalid json}`
	reader := strings.NewReader(invalidJSON)

	user, err := sv.ValidateIOReaderCtx(t.Context(), v, reader)

	if err == nil {
		t.Fatalf("Expected validation to fail for invalid JSON, got nil")
	}

	if user != nil {
		t.Fatalf("Expected user to be nil when JSON decoding fails, got %v", user)
	}

	if len(err) == 0 {
		t.Fatalf("Expected at least 1 error, got 0")
	}

	if err["error"].Messages[0] != "JSON input is not valid" {
		t.Fatalf("Expected 'JSON input is not valid', got '%s'", err["error"].Messages[0])
	}
}

// TestValidateAnyCtx_WithInvalidStruct checks that ValidateAnyCtx returns field errors and no struct for a struct with bad values.
func TestValidateAnyCtx_WithInvalidStruct(t *testing.T) {
	sv := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	sv.AddFieldRules("Name", func(v *FieldValidator) {
		v.AddRule("required", "Name is required")
	})
	sv.AddFieldRules("Age", func(v *FieldValidator) {
		v.AddRule("min=18", "Age must be at least 18")
	})

	invalidUser := User{
		Name: "",
		Age:  10,
	}

	user, err := sv.ValidateAnyCtx(t.Context(), v, invalidUser)

	if err == nil {
		t.Fatalf("Expected validation to fail for invalid struct, got nil")
	}

	if user != nil {
		t.Fatalf("Expected user to be nil when validation fails, got %v", user)
	}

	if len(err) == 0 {
		t.Fatalf("Expected validation errors, got none")
	}
}

// TestValidateStruct_ReturnEarlyWithFieldValidationError checks that ReturnEarly set to true returns one error when one field fails.
func TestValidateStruct_ReturnEarlyWithFieldValidationError(t *testing.T) {
	sv := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	// multiple rules on same field
	sv.AddFieldRules("Name", func(v *FieldValidator) {
		v.AddRule("required", "Name is required")
		v.AddRule("min=3", "Name must be at least 3 characters")
	})

	user := User{
		Name: "Al", // fails both min=3 and any other rules after
		Age:  20,
	}

	// with ReturnEarly=true, should return on first validation error
	err := sv.ValidateCtx(t.Context(), v, &user, config.SetReturnEarly(true))

	if err == nil {
		t.Fatalf("Expected validation to fail, got nil")
	}

	if len(err) != 1 {
		t.Fatalf("Expected 1 error with ReturnEarly=true, got %d", len(err))
	}
}

// userWithUnexportedField is a struct with an unexported field. Tests use it to check that rules for that field are refused.
type userWithUnexportedField struct {
	Name string
	age  int
}

// TestAddFieldRules_PanicsForUnexportedField checks that AddFieldRules panics for an unexported field.
func TestAddFieldRules_PanicsForUnexportedField(t *testing.T) {
	sv := NewStruct[userWithUnexportedField]()

	assertPanics(t, func() {
		sv.AddFieldRules("age", func(v *FieldValidator) {
			v.AddRule("min=18", "age must be at least 18")
		})
	})
}

// TestValidateStruct_ReturnEarlyFollowsFieldOrder checks that ReturnEarly always reports the first added field, not a random one.
func TestValidateStruct_ReturnEarlyFollowsFieldOrder(t *testing.T) {
	v := pgValidator.New()

	// run many times because map iteration order is random
	for range 50 {
		sv := NewStruct[User]().
			AddFieldRules("Name", func(v *FieldValidator) {
				v.AddRule("required", "Name is required")
			}).
			AddFieldRules("Age", func(v *FieldValidator) {
				v.AddRule("min=18", "Age must be at least 18")
			})

		valErrs := sv.ValidateCtx(t.Context(), v, &User{})

		if len(valErrs) != 1 {
			t.Fatalf("Expected 1 error with ReturnEarly=true, got %d", len(valErrs))
		}

		if _, ok := valErrs["Name"]; !ok {
			t.Fatalf("Expected error for the first added field 'Name', got %v", valErrs.ToErrorMap())
		}
	}
}

// TestAddFieldRules_SameFieldTwiceReplacesRules checks that a second AddFieldRules call for a field replaces its rules and keeps one position.
func TestAddFieldRules_SameFieldTwiceReplacesRules(t *testing.T) {
	v := pgValidator.New()

	sv := NewStruct[User]().
		AddFieldRules("Name", func(v *FieldValidator) {
			v.AddRule("required", "first message")
		}).
		AddFieldRules("Name", func(v *FieldValidator) {
			v.AddRule("required", "second message")
		})

	if len(sv.fieldOrder) != 1 {
		t.Fatalf("Expected field to be listed once, got %v", sv.fieldOrder)
	}

	assertFirstErrorMessage(&User{}, sv, v, "Name", "second message", t)
}

// TestValidateAnyCtx_UsesConfig checks that ValidateAnyCtx passes its configs to ValidateCtx.
func TestValidateAnyCtx_UsesConfig(t *testing.T) {
	v := pgValidator.New()

	sv := NewStruct[User]().
		AddFieldRules("Name", func(v *FieldValidator) {
			v.AddRule("required", "Name is required")
		}).
		AddFieldRules("Age", func(v *FieldValidator) {
			v.AddRule("min=18", "Age must be at least 18")
		})

	_, valErrs := sv.ValidateAnyCtx(t.Context(), v, User{}, config.SetReturnEarly(false))

	if len(valErrs) != 2 {
		t.Fatalf("Expected 2 errors with ReturnEarly=false, got %v", valErrs.ToErrorMap())
	}
}

// TestValidateIOReaderCtx_UsesConfig checks that ValidateIOReaderCtx passes its configs to ValidateCtx.
func TestValidateIOReaderCtx_UsesConfig(t *testing.T) {
	v := pgValidator.New()

	sv := NewStruct[User]().
		AddFieldRules("Name", func(v *FieldValidator) {
			v.AddRule("required", "Name is required")
		}).
		AddFieldRules("Age", func(v *FieldValidator) {
			v.AddRule("min=18", "Age must be at least 18")
		})

	_, valErrs := sv.ValidateIOReaderCtx(t.Context(), v, strings.NewReader(`{}`), config.SetReturnEarly(false))

	if len(valErrs) != 2 {
		t.Fatalf("Expected 2 errors with ReturnEarly=false, got %v", valErrs.ToErrorMap())
	}
}

// TestValidateAnyCtx_WithPointer checks that ValidateAnyCtx returns the same pointer for a *User input and rejects a nil pointer.
func TestValidateAnyCtx_WithPointer(t *testing.T) {
	v := pgValidator.New()

	sv := NewStruct[User]().
		AddFieldRules("Name", func(v *FieldValidator) {
			v.AddRule("required", "Name is required")
		})

	user := &User{Name: "John", Age: 30}

	validatedUser, valErrs := sv.ValidateAnyCtx(t.Context(), v, user)

	if valErrs != nil {
		t.Fatalf("Expected no error validating pointer to struct, got %v", valErrs.ToErrorMap())
	}

	if validatedUser != user {
		t.Fatalf("Expected the same pointer to be returned")
	}

	var nilUser *User
	_, valErrs = sv.ValidateAnyCtx(t.Context(), v, nilUser)

	if valErrs == nil {
		t.Fatalf("Expected validation to fail for nil pointer, got nil")
	}
}

// TestValidateIOReaderCtx_WithTrailingData checks that extra data after the JSON value is rejected, and that trailing whitespace is allowed.
func TestValidateIOReaderCtx_WithTrailingData(t *testing.T) {
	v := pgValidator.New()

	sv := NewStruct[User]().
		AddFieldRules("Name", func(v *FieldValidator) {
			v.AddRule("required", "Name is required")
		})

	_, valErrs := sv.ValidateIOReaderCtx(t.Context(), v, strings.NewReader(`{"Name":"John"} garbage`))

	if valErrs == nil {
		t.Fatalf("Expected validation to fail for data after the JSON value, got nil")
	}

	if valErrs["error"].Messages[0] != "JSON input must contain only one value" {
		t.Fatalf("Expected 'JSON input must contain only one value', got: %v", valErrs["error"].Messages[0])
	}

	// whitespace after the JSON value is allowed
	_, valErrs = sv.ValidateIOReaderCtx(t.Context(), v, strings.NewReader("{\"Name\":\"John\"}\n"))

	if valErrs != nil {
		t.Fatalf("Expected no error for trailing whitespace, got %v", valErrs.ToErrorMap())
	}
}

// TestValidateStruct_DefaultMessageNamesField checks that the default message of a struct field names the field.
func TestValidateStruct_DefaultMessageNamesField(t *testing.T) {
	v := pgValidator.New()

	sv := NewStruct[User]().
		AddFieldRules("Name", func(v *FieldValidator) {
			v.Required()
		})

	assertFirstErrorMessage(&User{}, sv, v, "Name", "field 'Name' failed 'required' validation", t)
}

// Address is a struct that tests embed in other structs.
type Address struct {
	City string
}

// userWithAddress is a struct with an embedded *Address, so its City field is promoted.
type userWithAddress struct {
	Name string
	*Address
}

// TestValidateStruct_PromotedFieldOfEmbeddedStruct checks that rules work on a promoted field, and that a nil embedded pointer gives a validation error instead of a panic.
func TestValidateStruct_PromotedFieldOfEmbeddedStruct(t *testing.T) {
	v := pgValidator.New()

	sv := NewStruct[userWithAddress]().
		AddFieldRules("City", func(v *FieldValidator) {
			v.Required("City is required")
		})

	valErrs := sv.ValidateCtx(t.Context(), v, &userWithAddress{Address: &Address{City: "Paris"}})

	if valErrs != nil {
		t.Fatalf("Expected no error, got %v", valErrs.ToErrorMap())
	}

	// a nil embedded pointer gives the rules a nil value instead of a panic
	assertFirstErrorMessage(&userWithAddress{}, sv, v, "City", "City is required", t)
}

// failingReader is an io.Reader that always fails, like a broken network connection.
type failingReader struct{}

// Read always returns an error.
func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("connection reset")
}

// userWithDetails is a struct with several field types, so tests can trigger each kind of JSON error.
type userWithDetails struct {
	Name      string
	Small     int8
	CreatedAt time.Time
	Address   Address
}

// TestValidateIOReaderCtx_JSONErrorMessages checks the plain message for each kind of bad JSON input.
func TestValidateIOReaderCtx_JSONErrorMessages(t *testing.T) {
	v := pgValidator.New()
	sv := NewStruct[userWithDetails]()

	tests := []struct {
		name     string
		reader   io.Reader
		expected string
	}{
		{"wrong type", strings.NewReader(`{"Name": 5}`), "JSON field 'Name' must be a string"},
		{"wrong nested type", strings.NewReader(`{"Address": {"City": 5}}`), "JSON field 'Address.City' must be a string"},
		{"wrong type for whole input", strings.NewReader(`[1]`), "JSON input must be an object"},
		{"number out of range", strings.NewReader(`{"Small": 300}`), "JSON field 'Small' has a number that is not allowed"},
		{"bad time value", strings.NewReader(`{"CreatedAt": "yesterday"}`), "JSON input could not be decoded"},
		{"empty input", strings.NewReader(``), "JSON input is empty"},
		{"cut-off input", strings.NewReader(`{"Name":`), "JSON input is not valid"},
		{"reader error", failingReader{}, "JSON input could not be decoded"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, valErrs := sv.ValidateIOReaderCtx(t.Context(), v, test.reader)

			if valErrs == nil {
				t.Fatalf("Expected an error, got nil")
			}

			if valErrs["error"].Messages[0] != test.expected {
				t.Fatalf("Expected '%s', got '%s'", test.expected, valErrs["error"].Messages[0])
			}
		})
	}
}

// TestJSONTypeName checks the JSON type name for each kind of Go type.
func TestJSONTypeName(t *testing.T) {
	tests := []struct {
		goType   reflect.Type
		expected string
	}{
		{nil, "a different type"},
		{reflect.TypeFor[*string](), "a string"},
		{reflect.TypeFor[time.Time](), "a string"},
		{reflect.TypeFor[[]byte](), "a string"},
		{reflect.TypeFor[bool](), "a boolean"},
		{reflect.TypeFor[float64](), "a number"},
		{reflect.TypeFor[map[string]int](), "an object"},
		{reflect.TypeFor[[]int](), "an array"},
		{reflect.TypeFor[[2]int](), "an array"},
		{reflect.TypeFor[func()](), "a different type"},
	}

	for _, test := range tests {
		if got := jsonTypeName(test.goType); got != test.expected {
			t.Errorf("Expected '%s' for %v, got '%s'", test.expected, test.goType, got)
		}
	}
}
