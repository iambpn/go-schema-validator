# Go Schema Validator v3

Lightweight, context-aware, chainable schema validation utilities built on top of `go-playground/validator` for simple values and generic structs.

## Introduction

Go Schema Validator v3 wraps `github.com/go-playground/validator/v10` with a fluent API and Go generics so you can describe validation logic once and reuse it across values, structs, and even streamed input. It ships with helper shortcuts for common rules, structured validation builders, context support, configurable error handling, and interfaces for plugging in custom logic.

## Features

- Fluent rule builder with human-friendly error messages
- Context-aware validation with `context.Context` support
- Type-safe struct validation using `StructValidator[T]`
- Structured error reporting with field-level validation errors
- Configurable validation behavior (e.g., return early on first error)
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

import (
	"context"

	"github.com/iambpn/go-schema-validator/v3/validator"
)

func main() {
	ctx := context.Background()

	err := validator.New().
		Required("Name is required").
		Min(3, "Name must have at least 3 characters").
		ValidateCtx(ctx, "Jo")

	if err != nil {
		panic(err)
	}
}
```

### Validate a struct with field-specific rules

```go
package main

import (
	"context"
	"fmt"

	"github.com/iambpn/go-schema-validator/v3/validator"
)

type User struct {
	Name  string
	Email string
	Age   int
}

func main() {
	ctx := context.Background()

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

	if valErrs := userValidator.ValidateCtx(ctx, &candidate); valErrs != nil {
		for _, valErr := range valErrs {
			fmt.Printf("Field '%s': %v\n", valErr.Field, valErr.Messages)
		}
	}
}
```

### Use interfaces for reusable validation logic

```go
package main

import (
	"context"

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
func (p *Payload) CustomValidate(ctx context.Context, data *Payload, sv *validator.StructValidator[Payload]) []validator.ValidationError {
	return sv.ValidateCtx(ctx, data)
}

func main() {
	ctx := context.Background()

	validated, valErrs := validator.ValidateStructCtx[Payload](ctx, Payload{Name: "Admin", Age: 20})
	if valErrs != nil {
		panic(valErrs[0].Messages[0])
	}

	_ = validated // ready-to-use struct
}
```

### Validate streamed JSON

```go
package main

import (
	"context"
	"strings"

	"github.com/iambpn/go-schema-validator/v3/validator"
)

type User struct {
	Name string
	Age  int
}

func main() {
	ctx := context.Background()

	reader := strings.NewReader(`{"name":"Streamed","age":25}`)
	validated, valErrs := validator.NewStruct[User]().
		AddFieldRules("Name", func(v *validator.Validator) { v.Required() }).
		AddFieldRules("Age", func(v *validator.Validator) { v.Min(18) }).
		ValidateIOReaderCtx(ctx, reader)

	if valErrs != nil {
		panic(valErrs[0].Messages[0])
	}

	_ = validated
}
```

### Configuration options

```go
package main

import (
	"context"
	"fmt"

	"github.com/iambpn/go-schema-validator/v3/internal/config"
	"github.com/iambpn/go-schema-validator/v3/validator"
)

type User struct {
	Name  string
	Email string
}

func main() {
	ctx := context.Background()

	userValidator := validator.NewStruct[User]().
		AddFieldRules("Name", func(v *validator.Validator) {
			v.Required("Name is required")
		}).
		AddFieldRules("Email", func(v *validator.Validator) {
			v.Required("Email is required").Email("Invalid email")
		})

	user := User{Name: "", Email: "invalid"}

	// Return early on first error (default behavior)
	valErrs := userValidator.ValidateCtx(ctx, &user, config.SetReturnEarly(true))
	if valErrs != nil {
		fmt.Printf("First error only: %v\n", valErrs[0].Messages[0])
	}

	// Collect all validation errors
	valErrs = userValidator.ValidateCtx(ctx, &user, config.SetReturnEarly(false))
	if valErrs != nil {
		for _, err := range valErrs {
			fmt.Printf("Field '%s': %v\n", err.Field, err.Messages)
		}
	}
}
```

## Exposed APIs

### Validator (for single values)

- `validator.New() *validator.Validator`: create a fluent rule builder for scalar values
- `(*validator.Validator).AddRule(tag string, message ...string) *Validator`: attach go-playground tags with optional messages
- `(*validator.Validator).Required/Email/Min/Max/Int(message ...string) *Validator`: helper methods wrapping common rules
- `(*validator.Validator).ValidateCtx(ctx context.Context, value any) error`: validate a value against the configured rule set

### StructValidator (for struct types)

- `validator.NewStruct[T any]() *validator.StructValidator[T]`: build validations for struct types
- `(*validator.StructValidator[T]).AddFieldRules(name string, fn func(*validator.Validator)) *StructValidator[T]`: register rules for struct fields
- `(*validator.StructValidator[T]).ValidateCtx(ctx context.Context, structPtr *T, configs ...config.Config) []validator.ValidationError`: validate an instance with optional configuration
- `(*validator.StructValidator[T]).ValidateAnyCtx(ctx context.Context, value any) (*T, []validator.ValidationError)`: validate a generic value castable to `T`
- `(*validator.StructValidator[T]).ValidateIOReaderCtx(ctx context.Context, reader io.Reader) (*T, []validator.ValidationError)`: decode JSON from a reader and validate it

### High-level helpers

- `validator.ValidateStructCtx[S any](ctx context.Context, data any, configs ...config.Config) (*S, []validator.ValidationError)`: high-level helper that leverages the interfaces below

### Interfaces

- `validator.ValidationRules[T]`: interface for declaring struct rules
  - `ValidationRules(sv *StructValidator[T])`
- `validator.CustomValidate[T]`: interface for supplying custom or cross-field validation logic
  - `CustomValidate(ctx context.Context, data *T, sv *StructValidator[T]) []ValidationError`

### Types

- `validator.ValidationError`: structured error type
  ```go
  type ValidationError struct {
      Field    string   `json:"field"`
      Messages []string `json:"messages"`
  }
  ```

### Configuration

- `config.SetReturnEarly(val bool) config.Config`: configure whether validation should stop at the first error (default: `true`) or collect all errors
- `config.GetDefaultConfig() config.Config`: get default configuration (returns early on first error)
- `config.MergeConfigs(configs ...config.Config) config.Config`: merge multiple configurations

## Contributing

- Fork the repository and create a feature branch
- Run `make test` or `go test ./... -v -cover` before submitting a pull request
- Use `make run` or `go run ./cmd/main.go` to experiment with the examples
- Run `make html-coverage` to generate a coverage report
- Open a PR describing the change, tests, and any new validation helpers

## License

This project is licensed under the MIT License. See `LICENSE` for details.
