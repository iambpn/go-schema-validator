package validator

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	pgValidator "github.com/go-playground/validator/v10"
	"github.com/iambpn/go-schema-validator/v3/internal/config"
)

type User struct {
	Name string
	Age  int
}

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

func TestNewStruct(t *testing.T) {
	schema := NewStruct[User]()

	if schema == nil {
		t.Errorf("Expected struct to be created, got %v", schema)
	}

	if reflect.TypeOf(schema) != reflect.TypeOf(&StructValidator[User]{}) {
		t.Errorf("Expected struct to be of type StructValidator, got %v", reflect.TypeOf(schema))
	}
}

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

func TestValidateStruct_NonStruct(t *testing.T) {
	sv := StructValidator[any]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	var anyValue any = "not a struct"

	expected := "provided value is not a struct"
	assertFirstErrorMessage(&anyValue, &sv, v, "error", expected, t)
}

func TestValidateStruct_FieldDoesNotExist(t *testing.T) {
	sv := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	// add a rule for a non-existing field
	sv.AddFieldRules("UnknownField", func(v *FieldValidator) {
		v.AddRule("required", "UnknownField is required")
	})

	expected := "field UnknownField does not exist"
	usr := &User{
		Name: "John",
		Age:  30,
	}

	assertFirstErrorMessage(usr, &sv, v, "UnknownField", expected, t)
}

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

func TestValidateStruct_ReturnEarlyOnFieldMissing(t *testing.T) {
	sv := StructValidator[User]{
		rules: make(map[string]*FieldValidator),
	}
	v := pgValidator.New()

	// add rule for a non-existing field
	sv.AddFieldRules("NonExistentField", func(v *FieldValidator) {
		v.AddRule("required", "NonExistentField is required")
	})

	user := User{
		Name: "John",
		Age:  30,
	}

	// with ReturnEarly=true, should stop at first error (non-existent field)
	valErrs := sv.ValidateCtx(t.Context(), v, &user, config.SetReturnEarly(true))

	if valErrs == nil {
		t.Fatalf("Expected validation to fail with non-existent field, got nil")
	}

	if len(valErrs) != 1 {
		t.Fatalf("Expected 1 error with ReturnEarly=true, got %d", len(valErrs))
	}

	if valErrs["NonExistentField"].Field != "NonExistentField" {
		t.Fatalf("Expected error field 'NonExistentField', got '%s'", valErrs["NonExistentField"].Field)
	}
}

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

	if !strings.Contains(err["error"].Messages[0], "failed to decode") {
		t.Fatalf("Expected error to mention decoding failure, got '%s'", err["error"].Messages[0])
	}
}

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
