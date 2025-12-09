# go-schema-validator v3

Lightweight, context-aware schema validation for Go built on top of `github.com/go-playground/validator/v10`. Define rules fluently, attach human-friendly messages, and reuse them across values, structs, and streamed JSON with Go generics.

## Introduction

This library wraps go-playground/validator with a small, fluent API. You describe validation once and apply it to single values, structs, or `io.Reader` inputs. Rules support custom messages, execution is context-aware, and behavior is configurable (return early or collect all errors). Interfaces are available when you prefer declarative, reusable validation logic.

## Features

- Fluent, chainable rule builder with custom messages
- Context-aware validation for values, structs, and JSON streams
- Go generics for type-safe struct validation (`StructValidator[T]`)
- Helper methods for common rules (`Required`, `Email`, `Min`, `Max`, `Length`, `Optional`, `URL`, `UUID`, `IsNumber`, `IsBoolean`)
- Configurable error aggregation (stop on first error or collect all)
- Structured error reporting with `ValidationError`/`ValidationErrors`
- Interfaces for declarative rules and custom validation hooks
- Panic-safe single-value validation (recovers and surfaces errors)

## Installation

Prerequisites: Go 1.25 or newer.

```bash
go get github.com/iambpn/go-schema-validator/v3
```

The package expects an instance of `*validator.Validate` from `github.com/go-playground/validator/v10` (aliased as `pgValidator` in examples below); create and reuse it in your application.

## Usage

### 1) Validate a single value

```go
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

	err := validator.New().
		Required("Name is required").
		Min(3, "Name must have at least 3 characters").
		ValidateFieldCtx(ctx, v, "Jo")

	if err != nil {
		fmt.Println("validation failed:", err)
	}
}
```

### 2) Validate a struct with field-specific rules

```go
package main

import (
	"context"
	"fmt"

	pgValidator "github.com/go-playground/validator/v10"
	"github.com/iambpn/go-schema-validator/v3/internal/config"
	"github.com/iambpn/go-schema-validator/v3/validator"
)

type User struct {
	Name  string
	Email string
	Age   int
}

func main() {
	ctx := context.Background()
	v := pgValidator.New()

	userValidator := validator.NewStruct[User]().
		AddFieldRules("Name", func(fv *validator.FieldValidator) {
			fv.Required("Name is required").Min(2, "Name must be at least 2 characters")
		}).
		AddFieldRules("Email", func(fv *validator.FieldValidator) {
			fv.Required("Email is required").Email("Invalid email address")
		}).
		AddFieldRules("Age", func(fv *validator.FieldValidator) {
			fv.IsNumber("Age must be numeric").Min(18, "Age must be 18 or older")
		})

	candidate := User{Name: "Al", Email: "bad", Age: 15}

	// Collect all errors instead of returning after the first
	if errs := userValidator.ValidateCtx(ctx, v, &candidate, config.SetReturnEarly(false)); errs != nil {
		for field, err := range errs.ToErrorMap() {
			fmt.Printf("%s: %v\n", field, err)
		}
	}
}
```

### 3) Declarative rules via interfaces

Implement `ValidationRules[T]` (and optionally `CustomValidate[T]`) to keep validation next to your structs.

```go
package main

import (
	"context"

	pgValidator "github.com/go-playground/validator/v10"
	"github.com/iambpn/go-schema-validator/v3/validator"
)

type Payload struct {
	Name string
	Age  int
}

func (p *Payload) ValidationRules(sv *validator.StructValidator[Payload]) {
	sv.AddFieldRules("Name", func(fv *validator.FieldValidator) {
		fv.Required("Name is required")
	})
	sv.AddFieldRules("Age", func(fv *validator.FieldValidator) {
		fv.Min(18, "Age must be 18 or older")
	})
}

// Optional custom hook for cross-field checks or alternative logic
func (p *Payload) CustomValidate(ctx context.Context, v *pgValidator.Validate, data *Payload, sv *validator.StructValidator[Payload]) validator.ValidationErrors {
	return sv.ValidateCtx(ctx, v, data) // reuse the declared rules
}

func main() {
	ctx := context.Background()
	v := pgValidator.New()

	validated, errs := validator.ValidateStructCtx[Payload](ctx, v, Payload{Name: "Ada", Age: 17})
	_ = validated

	if errs != nil {
		// errs is a map[string]ValidationError
		panic(errs.ToErrorMap())
	}
}
```

### 4) Validate streamed JSON from an `io.Reader`

```go
package main

import (
	"context"
	"strings"

	pgValidator "github.com/go-playground/validator/v10"
	"github.com/iambpn/go-schema-validator/v3/validator"
)

type User struct {
	Name string
	Age  int
}

func main() {
	ctx := context.Background()
	v := pgValidator.New()

	reader := strings.NewReader(`{"Name":"Streamed","Age":21}`)

	validated, errs := validator.NewStruct[User]().
		AddFieldRules("Name", func(fv *validator.FieldValidator) { fv.Required() }).
		AddFieldRules("Age", func(fv *validator.FieldValidator) { fv.Min(18) }).
		ValidateIOReaderCtx(ctx, v, reader)

	_ = validated
	if errs != nil {
		panic(errs.ToErrorMap())
	}
}
```

### 5) Configuration knobs

`config.SetReturnEarly(bool)` controls whether validation stops at the first error (default: `true`). Pass it to any `Validate*Ctx` call:

```go
errs := userValidator.ValidateCtx(ctx, v, &candidate, config.SetReturnEarly(false))
```

## Exposed APIs

In the signatures below, `pgValidator` refers to `github.com/go-playground/validator/v10`.

### Field validation

- `validator.New() *FieldValidator` — create a rule builder for scalar values
- `(*FieldValidator).AddRule(tag string, message ...string) *FieldValidator` — attach a go-playground tag with an optional message
- Helper shortcuts: `Required`, `Email`, `Min`, `Max`, `Length`, `Optional` (adds `omitempty`), `URL`, `UUID`, `IsNumber`, `IsBoolean`
- `(*FieldValidator).ValidateFieldCtx(ctx context.Context, v *pgValidator.Validate, value any) error` — run rules against a single value (panic-safe)

### Struct validation

- `validator.NewStruct[T any]() *StructValidator[T]` — build validations for struct types
- `(*StructValidator[T]).AddFieldRules(name string, fn func(*FieldValidator)) *StructValidator[T]` — register rules for a struct field
- `(*StructValidator[T]).ValidateCtx(ctx context.Context, v *pgValidator.Validate, structPtr *T, configs ...config.Config) ValidationErrors` — validate a struct instance
- `(*StructValidator[T]).ValidateAnyCtx(ctx context.Context, v *pgValidator.Validate, value any, configs ...config.Config) (*T, ValidationErrors)` — validate a value type-assertable to `T`
- `(*StructValidator[T]).ValidateIOReaderCtx(ctx context.Context, v *pgValidator.Validate, reader io.Reader, configs ...config.Config) (*T, ValidationErrors)` — decode JSON from a reader and validate it

### High-level helper

- `validator.ValidateStructCtx[S any](ctx context.Context, v *pgValidator.Validate, data any, configs ...config.Config) (*S, ValidationErrors)` — validate structs (or readers) implementing `ValidationRules`/`CustomValidate`

### Interfaces

- `validator.ValidationRules[T]` — declare field rules via `ValidationRules(sv *StructValidator[T])`
- `validator.CustomValidate[T]` — supply custom or cross-field logic via `CustomValidate(ctx, v, data, sv)`

### Types & utilities

- `type ValidationError struct { Field string; Messages []string }`
- `type ValidationErrors map[string]ValidationError` with `ToErrorMap()` for `map[string][]string`

### Configuration

- `config.SetReturnEarly(val bool) config.Config`
- `config.GetDefaultConfig() config.Config`
- `config.MergeConfigs(configs ...config.Config) config.Config`

## Contributing

1. Fork the repo and create a feature branch.
2. Keep changes formatted (`gofmt` is fine) and add tests where relevant.
3. Run the suite before opening a PR: `make test` (or `go test ./... -v -cover`).
4. Try the example app with `make run` and generate coverage via `make html-coverage` if helpful.
5. Open a PR describing what changed, why, and how it was tested.

## License

MIT License – see `LICENSE` for details.
