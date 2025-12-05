package main

import (
	"context"
	"fmt"

	pgValidator "github.com/go-playground/validator/v10"
	"github.com/iambpn/go-schema-validator/v3/validator"
)

func main() {
	ctx := context.Background()
	v := pgValidator.New()

	// Simple field validation
	customSchema := validator.New().
		AddRule("min=3", "Must be at least 3 characters").
		AddRule("max=10", "Must be at most 10 characters").
		AddRule("required", "This field is required")

	err := customSchema.ValidateFieldCtx(ctx, v, "he")
	if err != nil {
		fmt.Println("Validation error:", err)
	}

	// Using helper methods
	emailSchema := validator.New().
		Email("Must be a valid email").
		Required("Email is required")

	err = emailSchema.ValidateFieldCtx(ctx, v, "not-an-email")
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
		AddFieldRules("Name", func(v *validator.FieldValidator) {
			v.
				AddRule("min=2", "Name must be at least 2 characters").
				AddRule("max=50", "Name must be at most 50 characters")
		}).
		AddFieldRules("Email", func(v *validator.FieldValidator) {
			v.
				Email("Must be a valid email")
		}).
		AddFieldRules("Age", func(v *validator.FieldValidator) {
			v.IsNumber("Age must be an integer").
				AddRule("min=18", "Must be at least 18 years old").
				AddRule("max=120", "Must be at most 120 years old")
		})

	user := User{
		Name:  "J",
		Email: "not-an-email",
		Age:   15,
	}

	valErr := userSchema.ValidateCtx(ctx, v, &user)
	if valErr != nil {
		fmt.Println("User validation error:", valErr["Name"].Messages[0])
	}
}
