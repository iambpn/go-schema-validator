package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	pgValidator "github.com/go-playground/validator/v10"
	"github.com/iambpn/go-schema-validator/v3/internal/config"
)

// implement Validate interface for User struct
func (u *User) ValidationRules(sv *StructValidator[User]) {
	sv.AddFieldRules("Name", func(v *FieldValidator) {
		v.AddRule("required", "Name is required")
	})

	sv.AddFieldRules("Age", func(v *FieldValidator) {
		v.AddRule("min=18", "Age must be at least 18")
	})
}

func TestValidateStruct_withReader(t *testing.T) {
	user := User{
		Name: "name",
		Age:  18,
	}
	v := pgValidator.New()

	// convert to reader
	b, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal struct to JSON: %v", err)
	}

	reader := bytes.NewReader(b)

	validatedUser, valErr := ValidateStructCtx[User](t.Context(), v, reader)

	if valErr != nil {
		t.Fatalf("Expected validation to pass, got error: %v", valErr.ToErrorMap())
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
	v := pgValidator.New()

	// convert to reader
	b, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal struct to JSON: %v", err)
	}

	reader := bytes.NewReader(b)

	_, valErr := ValidateStructCtx[User](t.Context(), v, reader)

	if valErr == nil {
		t.Fatalf("Expected validation to fail, got error: nil")
	}
}

func TestValidateStruct_withStructInput(t *testing.T) {
	user := User{
		Name: "ValidUser",
		Age:  30,
	}
	v := pgValidator.New()

	validatedUser, err := ValidateStructCtx[User](t.Context(), v, user)

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
	v := pgValidator.New()

	_, err := ValidateStructCtx[User](t.Context(), v, user)

	if err == nil {
		t.Fatalf("Expected validation to fail, got error: %v", err)
	}
}

func TestValidateStruct_withInvalidTypeInReader(t *testing.T) {
	invalidJSON := `{"Name": "name", "Age": "not-an-integer"}`
	v := pgValidator.New()

	reader := strings.NewReader(invalidJSON)

	_, err := ValidateStructCtx[User](t.Context(), v, reader)

	if err == nil {
		t.Fatalf("Expected validation to fail for invalid type in JSON, got nil error")
	}
}

func TestValidateStruct_withInvalidJSON(t *testing.T) {
	invalidJSON := `{"Name": "name", "Age": "not-an-integer"}`
	v := pgValidator.New()

	reader := strings.NewReader(invalidJSON)

	_, err := ValidateStructCtx[User](t.Context(), v, reader)

	if err == nil {
		t.Fatalf("Expected validation to fail for invalid JSON, got nil error")
	}
}

func TestValidateStruct_withInvalidType(t *testing.T) {
	v := pgValidator.New()

	_, err := ValidateStructCtx[User](t.Context(), v, 12345)

	if err == nil {
		t.Fatalf("Expected validation to fail for invalid type, got nil error")
	}
}

func TestValidateStruct_withNilReader(t *testing.T) {
	v := pgValidator.New()

	_, err := ValidateStructCtx[User](t.Context(), v, nil)

	if err == nil {
		t.Fatalf("Expected validation to fail for nil reader, got nil error")
	}
}

func TestValidateStruct_withNilStruct(t *testing.T) {
	var user *User = nil
	v := pgValidator.New()

	_, err := ValidateStructCtx[User](t.Context(), v, user)

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
	v := pgValidator.New()

	validatedData, err := ValidateStructCtx[NoValidation](t.Context(), v, data)

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
func (nv *ValidateOnly) CustomValidate(ctx context.Context, v *pgValidator.Validate, data *ValidateOnly, sv *StructValidator[ValidateOnly]) ValidationErrors {
	return ValidationErrors{
		"error": ValidationError{Field: "error", Messages: []string{"no validation rules defined"}},
	}
}

func TestValidationStruct_OnlyValidateInterface(t *testing.T) {
	data := ValidateOnly{
		Field1: "test",
		Field2: 10,
	}
	v := pgValidator.New()

	validatedData, valErr := ValidateStructCtx[ValidateOnly](t.Context(), v, data)

	if valErr == nil {
		t.Fatalf("Expected error for struct with only Validate interface, got nil")
	}

	if valErr["error"].Messages[0] != "no validation rules defined" {
		t.Fatalf("Expected error message 'no validation rules defined', got '%v'", strings.Join(valErr["error"].Messages, ", "))
	}

	if valErr["error"].Field != "error" {
		t.Fatalf("Expected error field to be 'error', got '%s'", valErr["error"].Field)
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
	sv.AddFieldRules("Field1", func(v *FieldValidator) {
		v.AddRule("required", "Field1 is required")
	})
}

func (cv *CustomValidationStruct) CustomValidate(ctx context.Context, v *pgValidator.Validate, data *CustomValidationStruct, sv *StructValidator[CustomValidationStruct]) ValidationErrors {
	if data.Field2 < 0 {
		return ValidationErrors{
			"Field2": ValidationError{Field: "Field2", Messages: []string{"Field2 must be non-negative"}},
		}
	}
	return sv.ValidateCtx(ctx, v, data)
}

func TestValidationStruct_ValidateAndValidateInterface(t *testing.T) {
	data := CustomValidationStruct{
		Field1: "test",
		Field2: -5,
	}
	v := pgValidator.New()

	_, valErr := ValidateStructCtx[CustomValidationStruct](t.Context(), v, data)

	if valErr == nil {
		t.Fatalf("Expected error for custom validation failure, got nil")
	}

	expectedErrMsg := "Field2 must be non-negative"
	if valErr["Field2"].Messages[0] != expectedErrMsg {
		t.Fatalf("Expected error message '%s', got '%v'", expectedErrMsg, valErr["Field2"].Messages[0])
	}

	if valErr["Field2"].Field != "Field2" {
		t.Fatalf("Expected error field 'Field2', got '%s'", valErr["Field2"].Field)
	}
}

func TestMultiFieldValidationErrors(t *testing.T) {
	user := User{
		Name: "",
		Age:  15,
	}
	v := pgValidator.New()

	_, valErr := ValidateStructCtx[User](t.Context(), v, user, config.SetReturnEarly(false))

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

type CustomValidateOnlyStruct struct {
	Field1 string
	Field2 int
}

// implement only CustomValidate interface (no ValidationRules)
func (cv *CustomValidateOnlyStruct) CustomValidate(ctx context.Context, v *pgValidator.Validate, data *CustomValidateOnlyStruct, sv *StructValidator[CustomValidateOnlyStruct]) ValidationErrors {
	// return validation error without validation rules
	if data.Field2 < 0 {
		return ValidationErrors{
			"Field2": ValidationError{
				Field:    "Field2",
				Messages: []string{"Field2 must be non-negative"},
			},
		}
	}
	return nil
}

func TestValidateStruct_CustomValidateOnlyWithError(t *testing.T) {
	data := CustomValidateOnlyStruct{
		Field1: "test",
		Field2: -5,
	}
	v := pgValidator.New()

	validatedData, valErr := ValidateStructCtx[CustomValidateOnlyStruct](t.Context(), v, data)

	if valErr == nil {
		t.Fatalf("Expected validation to fail with CustomValidate-only interface, got nil")
	}

	if len(valErr) == 0 {
		t.Fatalf("Expected at least 1 validation error, got 0")
	}

	if valErr["Field2"].Field != "Field2" {
		t.Fatalf("Expected error field 'Field2', got '%s'", valErr["Field2"].Field)
	}

	if valErr["Field2"].Messages[0] != "Field2 must be non-negative" {
		t.Fatalf("Expected error message 'Field2 must be non-negative', got '%s'", valErr["Field2"].Messages[0])
	}

	if validatedData != nil {
		t.Fatalf("Expected validated data to be nil when validation fails, got %v", *validatedData)
	}
}

func TestValidateStruct_CustomValidateOnlyWithSuccess(t *testing.T) {
	data := CustomValidateOnlyStruct{
		Field1: "test",
		Field2: 5,
	}
	v := pgValidator.New()

	validatedData, valErr := ValidateStructCtx[CustomValidateOnlyStruct](t.Context(), v, data)

	if valErr != nil {
		t.Fatalf("Expected validation to pass, got error: %v", valErr["Field2"].Messages[0])
	}

	if validatedData == nil {
		t.Fatalf("Expected validated data to not be nil when validation passes")
	}

	if validatedData.Field1 != "test" || validatedData.Field2 != 5 {
		t.Fatalf("Expected validated data to match input, got %v", *validatedData)
	}
}
