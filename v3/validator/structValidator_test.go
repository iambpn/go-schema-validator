package validator

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/iambpn/go-schema-validator/v3/internal/config"
)

type User struct {
	Name string
	Age  int
}

func TestField(t *testing.T) {
	structVal := StructValidator[User]{
		pgValidate: nil,
		rules:      make(map[string]*Validator),
	}

	nameField := "Name"
	ageField := "Age"

	nameRequiredMessage := "Name is required"
	ageGt18Message := "Age must be at least 18"

	structVal.AddFieldRules(nameField, func(v *Validator) {
		v.AddRule("required", nameRequiredMessage)
	})
	structVal.AddFieldRules(ageField, func(v *Validator) {
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

func assertFirstErrorMessage[T any](val *T, sv *StructValidator[T], field string, errMsg string, t *testing.T) {
	t.Helper()

	err := sv.Validate(val)

	if err == nil {
		t.Fatalf("Expected validation to fail, got nil")
	}

	if err[0].Messages[0] != errMsg {
		t.Fatalf("Expected error message to be '%s', got '%s'", errMsg, err[0].Messages[0])
	}

	if err[0].Field != field {
		t.Fatalf("Expected error field to be '%s', got '%s'", field, err[0].Field)
	}
}

func TestStructValidatorValidateChainedRules(t *testing.T) {
	structVal := StructValidator[User]{
		pgValidate: nil,
		rules:      make(map[string]*Validator),
	}

	nameField := "Name"
	nameMin2Message := "Name must be minimum of 2 characters"
	ageField := "Age"

	nameRequiredMessage := "Name is required"
	ageGt18Message := "Age must be at least 18"

	structVal.AddFieldRules(nameField, func(v *Validator) {
		v.AddRule("required", nameRequiredMessage)
		v.AddRule("min=2", nameMin2Message)
	})
	structVal.AddFieldRules(ageField, func(v *Validator) {
		v.AddRule("min=18", ageGt18Message)
	})

	assertFirstErrorMessage(&User{
		Name: "",
		Age:  18,
	}, &structVal, nameField, nameRequiredMessage, t)

	assertFirstErrorMessage(&User{
		Name: "a",
		Age:  18,
	}, &structVal, nameField, nameMin2Message, t)

	assertFirstErrorMessage(&User{
		Name: "aa",
		Age:  0,
	}, &structVal, ageField, ageGt18Message, t)
}

func TestStructValidatorValidate(t *testing.T) {
	structVal := StructValidator[User]{
		pgValidate: nil,
		rules:      make(map[string]*Validator),
	}

	nameField := "Name"
	ageField := "Age"

	nameRequiredMessage := "Name is required"
	ageGt18Message := "Age must be at least 18"

	structVal.AddFieldRules(nameField, func(v *Validator) {
		v.AddRule("required", nameRequiredMessage)
	})
	structVal.AddFieldRules(ageField, func(v *Validator) {
		v.AddRule("min=18", ageGt18Message)
	})

	_, err := structVal.ValidateAny("")

	if err == nil {
		t.Errorf("Expected error on empty struct, got %v", err)
	}

	noUser := User{
		Name: "John",
		Age:  18,
	}

	err = structVal.Validate(&noUser)

	if err != nil {
		t.Errorf("Expected no error on valid struct, got %v", err)
	}

	noNameUser := User{
		Name: "",
		Age:  18,
	}
	assertFirstErrorMessage(&noNameUser, &structVal, nameField, nameRequiredMessage, t)

	noAgeUser := User{
		Name: "John",
		Age:  0,
	}
	assertFirstErrorMessage(&noAgeUser, &structVal, ageField, ageGt18Message, t)
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
		pgValidate: nil,
		rules:      make(map[string]*Validator),
	}

	noUser := User{
		Name: "Alice",
		Age:  25,
	}

	ret, err := structVal.ValidateAny(noUser)

	if err != nil {
		t.Fatalf("Expected no error on ValidateAny with correct type, got %v", err)
	}

	if *ret != noUser {
		t.Fatalf("Expected returned value to equal input, got %v", ret)
	}
}

func TestValidateStruct_NonStruct(t *testing.T) {
	sv := StructValidator[any]{
		pgValidate: nil,
		rules:      make(map[string]*Validator),
	}

	var anyValue any = "not a struct"

	expected := "argument value must be a struct"
	assertFirstErrorMessage(&anyValue, &sv, "", expected, t)
}

func TestValidateStruct_FieldDoesNotExist(t *testing.T) {
	sv := StructValidator[User]{
		pgValidate: nil,
		rules:      make(map[string]*Validator),
	}

	// add a rule for a non-existing field
	sv.AddFieldRules("UnknownField", func(v *Validator) {
		v.AddRule("required", "UnknownField is required")
	})

	expected := "field UnknownField does not exist"
	usr := &User{
		Name: "John",
		Age:  30,
	}

	assertFirstErrorMessage(usr, &sv, "UnknownField", expected, t)
}

func TestValidateStruct_PointerInput(t *testing.T) {
	sv := StructValidator[User]{
		pgValidate: nil,
		rules:      make(map[string]*Validator),
	}

	sv.AddFieldRules("Name", func(v *Validator) {
		v.AddRule("required", "Name is required")
	})

	user := User{
		Name: "PointerUser",
		Age:  40,
	}

	// pass pointer to struct
	err := sv.Validate(&user)

	if err != nil {
		t.Fatalf("Expected no error validating pointer to struct, got %v", err)
	}
}

func TestValidateStruct_IoReaderInput(t *testing.T) {
	sv := StructValidator[User]{
		pgValidate: nil,
		rules:      make(map[string]*Validator),
	}

	sv.AddFieldRules("Name", func(v *Validator) {
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

	user, valErr := sv.ValidateIOReader(reader)

	if valErr != nil {
		t.Fatalf("Expected nil when validating io.Reader, got %v", valErr[0].Messages[0])
	}

	if (*user) != originalUser {
		t.Fatalf("Expected returned user to be same as Original User, got %v", user)
	}
}

func TestValidateStructMultipleErrors(t *testing.T) {
	sv := StructValidator[User]{
		pgValidate: nil,
		rules:      make(map[string]*Validator),
	}

	sv.AddFieldRules("Name", func(v *Validator) {
		v.AddRule("required", "Name is required")
		v.AddRule("min=3", "Name must be at least 3 characters")
	})
	sv.AddFieldRules("Age", func(v *Validator) {
		v.AddRule("min=18", "Age must be at least 18")
	})

	user := User{
		Name: "Al",
		Age:  15,
	}

	err := sv.Validate(&user, config.SetReturnEarly(false))

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
