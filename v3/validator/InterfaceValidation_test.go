package validator

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/iambpn/go-schema-validator/v3/internal/config"
)

// implement Validate interface for User struct
func (u *User) ValidationRules(sv *StructValidator[User]) {
	sv.AddFieldRules("Name", func(v *Validator) {
		v.AddRule("required", "Name is required")
	})

	sv.AddFieldRules("Age", func(v *Validator) {
		v.AddRule("min=18", "Age must be at least 18")
	})
}

func TestValidateStruct_withReader(t *testing.T) {
	user := User{
		Name: "name",
		Age:  18,
	}

	// convert to reader
	b, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal struct to JSON: %v", err)
	}

	reader := bytes.NewReader(b)

	validatedUser, valErr := ValidateStruct[User](reader)

	if valErr != nil {
		t.Fatalf("Expected validation to pass, got error: %v", valErr[0].Messages[0])
	}

	if *validatedUser != user {
		t.Fatalf("Expected validated user to be same as original user, got %v", *validatedUser)
	}
}

type InvalidUser struct {
	Name string
	DOB  string
}

func TestValidateStruct_withReaderAndError(t *testing.T) {
	user := InvalidUser{
		Name: "name",
		DOB:  "2025/01/01",
	}

	// convert to reader
	b, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal struct to JSON: %v", err)
	}

	reader := bytes.NewReader(b)

	_, valErr := ValidateStruct[User](reader)

	if valErr == nil {
		t.Fatalf("Expected validation to fail, got error: nil")
	}
}

func TestValidateStruct_withStructInput(t *testing.T) {
	user := User{
		Name: "ValidUser",
		Age:  30,
	}

	validatedUser, err := ValidateStruct[User](user)

	if err != nil {
		t.Fatalf("Expected validation to pass, got error: %v", err)
	}

	if *validatedUser != user {
		t.Fatalf("Expected validated user to be same as original user, got %v", *validatedUser)
	}
}

func TestValidateStruct_withStructInputAndError(t *testing.T) {
	user := InvalidUser{
		Name: "InvalidUser",
		DOB:  "2030/12/12",
	}

	_, err := ValidateStruct[User](user)

	if err == nil {
		t.Fatalf("Expected validation to fail, got error: %v", err)
	}
}

func TestValidateStruct_withInvalidTypeInReader(t *testing.T) {
	invalidJSON := `{"Name": "name", "Age": "not-an-integer"}`

	reader := strings.NewReader(invalidJSON)

	_, err := ValidateStruct[User](reader)

	if err == nil {
		t.Fatalf("Expected validation to fail for invalid type in JSON, got nil error")
	}
}

func TestValidateStruct_withInvalidJSON(t *testing.T) {
	invalidJSON := `{"Name": "name", "Age": "not-an-integer"}`

	reader := strings.NewReader(invalidJSON)

	_, err := ValidateStruct[User](reader)

	if err == nil {
		t.Fatalf("Expected validation to fail for invalid JSON, got nil error")
	}
}

func TestValidateStruct_withInvalidType(t *testing.T) {
	_, err := ValidateStruct[User](12345)

	if err == nil {
		t.Fatalf("Expected validation to fail for invalid type, got nil error")
	}
}

func TestValidateStruct_withNilReader(t *testing.T) {
	_, err := ValidateStruct[User](nil)

	if err == nil {
		t.Fatalf("Expected validation to fail for nil reader, got nil error")
	}
}

func TestValidateStruct_withNilStruct(t *testing.T) {
	var user *User = nil

	_, err := ValidateStruct[User](user)

	if err == nil {
		t.Fatalf("Expected validation to fail for nil struct, got nil error")
	}
}

func TestValidationStruct_NoValidationRuleAndValidate(t *testing.T) {
	type NoValidation struct {
		Field1 string
		Field2 int
	}

	data := NoValidation{
		Field1: "test",
		Field2: 10,
	}

	validatedData, err := ValidateStruct[NoValidation](data)

	if err == nil {
		t.Fatalf("Expected  error for struct without ValidationRule, got %v", err)
	}

	if validatedData != nil {
		t.Fatalf("Expected validated data to be nil, got %v", *validatedData)
	}
}

type ValidateOnly struct {
	Field1 string
	Field2 int
}

// implement only CustomValidate interface
func (nv *ValidateOnly) CustomValidate(data *ValidateOnly, sv *StructValidator[ValidateOnly]) []ValidationError {
	return []ValidationError{{Messages: []string{"no validation rules defined"}}}
}

func TestValidationStruct_OnlyValidateInterface(t *testing.T) {
	data := ValidateOnly{
		Field1: "test",
		Field2: 10,
	}

	validatedData, valErr := ValidateStruct[ValidateOnly](data)

	if valErr == nil {
		t.Fatalf("Expected error for struct with only Validate interface, got nil")
	}

	if valErr[0].Messages[0] != "no validation rules defined" {
		t.Fatalf("Expected error message 'no validation rules defined', got '%v'", strings.Join(valErr[0].Messages, ", "))
	}

	if valErr[0].Field != "" {
		t.Fatalf("Expected error field to be empty, got '%s'", valErr[0].Field)
	}

	if validatedData != nil {
		t.Fatalf("Expected validated data to be nil, got %v", *validatedData)
	}
}

type CustomValidationStruct struct {
	Field1 string
	Field2 int
}

// implement both ValidationRules and Validate interfaces
func (cv *CustomValidationStruct) ValidationRules(sv *StructValidator[CustomValidationStruct]) {
	sv.AddFieldRules("Field1", func(v *Validator) {
		v.AddRule("required", "Field1 is required")
	})
}

func (cv *CustomValidationStruct) CustomValidate(data *CustomValidationStruct, sv *StructValidator[CustomValidationStruct]) []ValidationError {
	if data.Field2 < 0 {
		return []ValidationError{{Messages: []string{"Field2 must be non-negative"}, Field: "Field2"}}
	}
	return sv.Validate(data)
}

func TestValidationStruct_ValidateAndValidateInterface(t *testing.T) {
	data := CustomValidationStruct{
		Field1: "test",
		Field2: -5,
	}

	_, valErr := ValidateStruct[CustomValidationStruct](data)

	if valErr == nil {
		t.Fatalf("Expected error for custom validation failure, got nil")
	}

	expectedErrMsg := "Field2 must be non-negative"
	if valErr[0].Messages[0] != expectedErrMsg {
		t.Fatalf("Expected error message '%s', got '%v'", expectedErrMsg, valErr[0].Messages[0])
	}

	if valErr[0].Field != "Field2" {
		t.Fatalf("Expected error field 'Field2', got '%s'", valErr[0].Field)
	}
}

func TestMultiFieldValidationErrors(t *testing.T) {
	user := User{
		Name: "",
		Age:  15,
	}

	_, valErr := ValidateStruct[User](user, config.SetReturnEarly(false))

	if valErr == nil {
		t.Fatalf("Expected validation to fail, got nil")
	}

	if len(valErr) != 2 {
		t.Fatalf("Expected 2 validation errors, got %d", len(valErr))
	}

	expectedErrors := map[string]string{
		"Name": "Name is required",
		"Age":  "Age must be at least 18",
	}

	for _, err := range valErr {
		expectedMsg, exists := expectedErrors[err.Field]
		if !exists {
			t.Fatalf("Unexpected validation error for field %s", err.Field)
		}
		if err.Messages[0] != expectedMsg {
			t.Fatalf("Expected error message '%s' for field %s, got '%s'", expectedMsg, err.Field, err.Messages[0])
		}
	}
}
