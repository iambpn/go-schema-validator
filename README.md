# go-schema-validator

golang-schema-validator is a zod like API used to validation user input. It allows user to define a validation schema and add a custom message while creating validation schema and returns the custom message on validation failed.

One of the big pain point of using go-playground/validator is that it doesn't allow user to define custom message for validation error. This package provides a simple API to create a validation schema and add custom message for each validation rule.

This package is the wrapper around [go-playground/validator](https://github.com/go-playground/validator) package.

## Install Package

```bash
go get github.com/iambpn/go-schema-validator/schema
```

## Usage

```go
package main

import (
	"fmt"
	"go-schema-validator/schema"
)

func main() {
	// Simple field validation
	customSchema := schema.New().
		AddValidation("min=3", "Must be at least 3 characters").
		AddValidation("max=10", "Must be at most 10 characters").
		AddValidation("required", "This field is required")

	err := customSchema.Validate("hello")
	if err != nil {
		fmt.Println("Validation error:", err)
	}

	// Using helper methods
	emailSchema := schema.New().
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

	userSchema := schema.New().Struct().
		Field("Name", schema.New().
			AddValidation("min=2", "Name must be at least 2 characters").
			AddValidation("max=50", "Name must be at most 50 characters")).
		Field("Email", schema.New().
			Email("Must be a valid email")).
		Field("Age", schema.New().
			Int("Age must be an integer").
			AddValidation("min=18", "Must be at least 18 years old").
			AddValidation("max=120", "Must be at most 120 years old"))

	user := User{
		Name:  "J",
		Email: "not-an-email",
		Age:   15,
	}

	err = userSchema.Validate(user)
	if err != nil {
		fmt.Println("User validation error:", err)
	}
}
```

## Resources

- [Go validator](https://github.com/go-playground/validator)
- [Go publish package](https://go.dev/doc/modules/publishing)
- [Go pkg site](https://pkg.go.dev/about)
