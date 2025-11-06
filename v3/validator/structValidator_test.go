package validator

import (
	"reflect"
	"testing"
)

type User struct {
	Name string
	Age  int
}

func TestField(t *testing.T) {
	structVal := structValidator[User]{
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

func TestValidateStruct(t *testing.T) {
	structVal := structValidator[User]{
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

	err = structVal.ValidateStruct(&noUser)

	if err != nil {
		t.Errorf("Expected no error on valid struct, got %v", err)
	}

	noNameUser := User{
		Name: "",
		Age:  18,
	}

	err = structVal.ValidateStruct(&noNameUser)

	if err == nil {
		t.Errorf("Expected error on empty name, got %v", err)
	}

	if err.Error() != nameRequiredMessage {
		t.Errorf("Expected error message to be '%s', got '%v'", nameRequiredMessage, err)
	}

	noAgeUser := User{
		Name: "John",
		Age:  0,
	}

	err = structVal.ValidateStruct(&noAgeUser)

	if err == nil {
		t.Errorf("Expected error on zero age, got %v", err)
	}

	if err.Error() != ageGt18Message {
		t.Errorf("Expected error message to be '%s', got '%v'", ageGt18Message, err)
	}
}

func TestNewStruct(t *testing.T) {
	schema := NewStruct[User]()

	if schema == nil {
		t.Errorf("Expected struct to be created, got %v", schema)
	}

	if reflect.TypeOf(schema) != reflect.TypeOf(&structValidator[User]{}) {
		t.Errorf("Expected struct to be of type structValidator, got %v", reflect.TypeOf(schema))
	}
}

func TestValidateAnySuccess(t *testing.T) {
	structVal := structValidator[User]{
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

	if ret != noUser {
		t.Fatalf("Expected returned value to equal input, got %v", ret)
	}
}

func TestValidateStruct_NonStruct(t *testing.T) {
	sv := structValidator[any]{
		pgValidate: nil,
		rules:      make(map[string]*Validator),
	}

	var anyValue any = "not a struct"

	err := sv.ValidateStruct(&anyValue)

	if err == nil {
		t.Fatalf("Expected error when validating a non-struct, got nil")
	}

	expected := "argument value must be a struct"
	if err.Error() != expected {
		t.Fatalf("Expected error message '%s', got '%v'", expected, err)
	}
}

func TestValidateStruct_FieldDoesNotExist(t *testing.T) {
	sv := structValidator[User]{
		pgValidate: nil,
		rules:      make(map[string]*Validator),
	}

	// add a rule for a non-existing field
	sv.AddFieldRules("UnknownField", func(v *Validator) {
		v.AddRule("required", "UnknownField is required")
	})

	err := sv.ValidateStruct(&User{
		Name: "John",
		Age:  30,
	})

	if err == nil {
		t.Fatalf("Expected error for missing field, got nil")
	}

	expected := "field UnknownField does not exist"
	if err.Error() != expected {
		t.Fatalf("Expected error message '%s', got '%v'", expected, err)
	}
}

func TestValidateStruct_PointerInput(t *testing.T) {
	sv := structValidator[User]{
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
	err := sv.ValidateStruct(&user)

	if err != nil {
		t.Fatalf("Expected no error validating pointer to struct, got %v", err)
	}
}
