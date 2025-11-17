package main

import (
	"fmt"

	"github.com/iambpn/go-schema-validator/v3/validator"
)

func main() {
	// Simple field validation
	customSchema := validator.New().
		AddRule("min=3", "Must be at least 3 characters").
		AddRule("max=10", "Must be at most 10 characters").
		AddRule("required", "This field is required")

	err := customSchema.Validate("he")
	if err != nil {
		fmt.Println("Validation error:", err)
	}

	// Using helper methods
	emailSchema := validator.New().
		Email("Must be a valid email").
		Required("Email is required")

	err = emailSchema.Validate("not-an-email")
	if err != nil {
		fmt.Println("Email validation error:", err)
	}

	// Struct validation
	type User struct {
		Name  string
		Email string
		Age   int
	}

	userSchema := validator.NewStruct[User]().
		AddFieldRules("Name", func(v *validator.Validator) {
			v.
				AddRule("min=2", "Name must be at least 2 characters").
				AddRule("max=50", "Name must be at most 50 characters")
		}).
		AddFieldRules("Email", func(v *validator.Validator) {
			v.
				Email("Must be a valid email")
		}).
		AddFieldRules("Age", func(v *validator.Validator) {
			v.Int("Age must be an integer").
				AddRule("min=18", "Must be at least 18 years old").
				AddRule("max=120", "Must be at most 120 years old")
		})

	user := User{
		Name:  "J",
		Email: "not-an-email",
		Age:   15,
	}

	err = userSchema.Validate(&user)
	if err != nil {
		fmt.Println("User validation error:", err)
	}
}
