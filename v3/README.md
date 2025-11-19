# Go Schema Validator v3
Lightweight, chainable schema validation utilities built on top of `go-playground/validator` for simple values and generic structs.

## Introduction
Go Schema Validator v3 wraps `github.com/go-playground/validator/v10` with a fluent API and Go generics so you can describe validation logic once and reuse it across values, structs, and even streamed input. It ships with helper shortcuts for common rules, structured validation builders, and interfaces for plugging in custom logic.

## Features
- Fluent rule builder with human friendly error messages
- Type-safe struct validation using `StructValidator[T]`
- Works with native values, structs, and JSON `io.Reader` inputs
- Simple helpers for common rules like `Required`, `Email`, `Min`, `Max`, and `Int`
- Optional interfaces for declarative and custom validation workflows

## Installation
```bash
go get github.com/iambpn/go-schema-validator/v3
```

The module targets Go 1.25 or newer as declared in `go.mod`.

## Usage

### Validate a single value
```go
package main

import "github.com/iambpn/go-schema-validator/v3/validator"

func main() {
	err := validator.New().
		Required("Name is required").
		Min(3, "Name must have at least 3 characters").
		Validate("Jo")

	if err != nil {
		panic(err)
	}
}
```

### Validate a struct with field-specific rules
```go
package main

import (
	"fmt"

	"github.com/iambpn/go-schema-validator/v3/validator"
)

type User struct {
	Name  string
	Email string
	Age   int
}

func main() {
	userValidator := validator.NewStruct[User]().
		AddFieldRules("Name", func(v *validator.Validator) {
			v.Required("Name is required").
				Min(2, "Name must be at least 2 characters")
		}).
		AddFieldRules("Email", func(v *validator.Validator) {
			v.Email("Email must be valid")
		}).
		AddFieldRules("Age", func(v *validator.Validator) {
			v.Int("Age must be a number").
				Min(18, "Age must be 18 or older")
		})

	candidate := User{Name: "Jane", Email: "jane@example.com", Age: 32}

	if err := userValidator.Validate(&candidate); err != nil {
		fmt.Println("validation error:", err)
	}
}
```

### Use interfaces for reusable validation logic
```go
package main

import (
	"github.com/iambpn/go-schema-validator/v3/validator"
)

type Payload struct {
	Name string
	Age  int
}

func (p *Payload) ValidationRules(sv *validator.StructValidator[Payload]) {
	sv.AddFieldRules("Name", func(v *validator.Validator) {
		v.Required("Name is required")
	})
	sv.AddFieldRules("Age", func(v *validator.Validator) {
		v.Min(18, "Age must be at least 18")
	})
}

// Optional: add cross-field or complex checks
func (p *Payload) CustomValidate(data *Payload, sv *validator.StructValidator[Payload]) error {
	return sv.Validate(data)
}

func main() {
	validated, err := validator.ValidateStruct[Payload](Payload{Name: "Admin", Age: 20})
	if err != nil {
		panic(err)
	}

	_ = validated // ready-to-use struct
}
```

### Validate streamed JSON
```go
package main

import (
	"strings"

	"github.com/iambpn/go-schema-validator/v3/validator"
)

type User struct {
	Name string
	Age  int
}

func main() {
	reader := strings.NewReader(`{"name":"Streamed","age":25}`)
	validated, err := validator.NewStruct[User]().
		AddFieldRules("Name", func(v *validator.Validator) { v.Required() }).
		AddFieldRules("Age", func(v *validator.Validator) { v.Min(18) }).
		ValidateIOReader(reader)

	if err != nil {
		panic(err)
	}

	_ = validated
}
```

## Exposed APIs
- `validator.New() *validator.Validator`: create a fluent rule builder for scalar values
- `(*validator.Validator).AddRule(tag string, message ...string)`: attach go-playground tags with optional messages
- `(*validator.Validator).Required/Email/Min/Max/Int`: helper methods wrapping common rules
- `(*validator.Validator).Validate(value any) error`: validate a value against the configured rule set
- `validator.NewStruct[T any]() *validator.StructValidator[T]`: build validations for struct types
- `(*validator.StructValidator[T]).AddFieldRules(name string, fn func(*validator.Validator))`: register rules for struct fields
- `(*validator.StructValidator[T]).Validate(structPtr *T) error`: validate an instance
- `(*validator.StructValidator[T]).ValidateAny(value any) (*T, error)`: validate a generic value castable to `T`
- `(*validator.StructValidator[T]).ValidateIOReader(reader io.Reader) (*T, error)`: decode JSON from a reader and validate it
- `validator.ValidateStruct[S any](data any) (*S, error)`: high-level helper that leverages the interfaces below
- `validator.ValidationRules[T]`: interface for declaring struct rules
- `validator.CustomValidate[T]`: interface for supplying custom or cross-field validation logic

## Contributing
- Fork the repository and create a feature branch
- Run `go test ./... -v` before submitting a pull request
- Use `go run ./cmd/main.go` to experiment with the examples
- Open a PR describing the change, tests, and any new validation helpers

## License
This project is licensed under the MIT License. See `LICENSE` for details.
