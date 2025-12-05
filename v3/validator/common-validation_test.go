package validator

import (
	"testing"

	pgValidator "github.com/go-playground/validator/v10"
)

func TestIsNumber(t *testing.T) {
	s := New().IsNumber("Must be a number")
	v := pgValidator.New()

	if s.compileRules() != "number" {
		t.Errorf("Expected number, got %s", s.compileRules())
	}

	ctx := t.Context()
	err := s.ValidateFieldCtx(ctx, v, "1")

	if err != nil {
		t.Errorf("Expected no error on numeric string, got %v", err)
	}

	err = s.ValidateFieldCtx(ctx, v, 11)

	if err != nil {
		t.Errorf("Expected no error on integer, got %v", err)
	}
}

func TestMin(t *testing.T) {
	s := New().Min(10, "Must be at least 10 characters")
	v := pgValidator.New()

	ctx := t.Context()
	err := s.ValidateFieldCtx(ctx, v, "hello")

	if err == nil {
		t.Errorf("Expected error on string, got %v", err)
	}

	err = s.ValidateFieldCtx(ctx, v, "hello world!")

	if err != nil {
		t.Errorf("Expected no error on string, got %v", err)
	}
}

func TestMax(t *testing.T) {
	s := New().Max(10, "Must be at most 10 characters")
	v := pgValidator.New()

	ctx := t.Context()
	err := s.ValidateFieldCtx(ctx, v, "hello")

	if err != nil {
		t.Errorf("Expected error on string, got %v", err)
	}

	err = s.ValidateFieldCtx(ctx, v, "hello world")

	if err == nil {
		t.Errorf("Expected error on string, got %v", err)
	}
}

func TestRequired(t *testing.T) {
	s := New().Required("This field is required")
	v := pgValidator.New()

	ctx := t.Context()
	err := s.ValidateFieldCtx(ctx, v, "")

	if err == nil {
		t.Errorf("Expected error on empty string, got %v", err)
	}

	err = s.ValidateFieldCtx(ctx, v, "hello")

	if err != nil {
		t.Errorf("Expected no error on string, got %v", err)
	}
}

func TestEmail(t *testing.T) {
	s := New().Email("Must be a valid email")
	v := pgValidator.New()

	ctx := t.Context()
	err := s.ValidateFieldCtx(ctx, v, "not-an-email")

	if err == nil {
		t.Errorf("Expected error on invalid email, got %v", err)
	}

	err = s.ValidateFieldCtx(ctx, v, "test@example.com")

	if err != nil {
		t.Errorf("Expected no error on valid email, got %v", err)
	}
}

func TestIsBoolean(t *testing.T) {
	s := New().IsBoolean("Must be a boolean")
	v := pgValidator.New()

	ctx := t.Context()
	err := s.ValidateFieldCtx(ctx, v, "not-a-bool")

	if err == nil {
		t.Errorf("Expected error on non-boolean value, got %v", err)
	}

	err = s.ValidateFieldCtx(ctx, v, true)

	if err != nil {
		t.Errorf("Expected no error on boolean true, got %v", err)
	}

	err = s.ValidateFieldCtx(ctx, v, false)

	if err != nil {
		t.Errorf("Expected no error on boolean false, got %v", err)
	}
}

func TestOptional(t *testing.T) {
	s := New().Optional()
	v := pgValidator.New()

	ctx := t.Context()

	// omitempty allows empty/zero values
	err := s.ValidateFieldCtx(ctx, v, "")

	if err != nil {
		t.Errorf("Expected no error on empty string with Optional, got %v", err)
	}

	err = s.ValidateFieldCtx(ctx, v, "some value")

	if err != nil {
		t.Errorf("Expected no error on non-empty string with Optional, got %v", err)
	}

	err = s.ValidateFieldCtx(ctx, v, 0)

	if err != nil {
		t.Errorf("Expected no error on zero value with Optional, got %v", err)
	}
}

func TestURL(t *testing.T) {
	s := New().URL("Must be a valid URL")
	v := pgValidator.New()

	ctx := t.Context()
	err := s.ValidateFieldCtx(ctx, v, "not a url")

	if err == nil {
		t.Errorf("Expected error on invalid URL, got %v", err)
	}

	err = s.ValidateFieldCtx(ctx, v, "https://example.com")

	if err != nil {
		t.Errorf("Expected no error on valid URL, got %v", err)
	}

	err = s.ValidateFieldCtx(ctx, v, "http://localhost:8080/path?query=value")

	if err != nil {
		t.Errorf("Expected no error on valid complex URL, got %v", err)
	}
}

func TestUUID(t *testing.T) {
	s := New().UUID("Must be a valid UUID")
	v := pgValidator.New()

	ctx := t.Context()
	err := s.ValidateFieldCtx(ctx, v, "not-a-uuid")

	if err == nil {
		t.Errorf("Expected error on invalid UUID, got %v", err)
	}

	// valid UUID v4 format
	validUUID := "550e8400-e29b-41d4-a716-446655440000"
	err = s.ValidateFieldCtx(ctx, v, validUUID)

	if err != nil {
		t.Errorf("Expected no error on valid UUID, got %v", err)
	}
}

func TestLength(t *testing.T) {
	s := New().Length(5, "Must be exactly 5 characters")
	v := pgValidator.New()

	ctx := t.Context()
	err := s.ValidateFieldCtx(ctx, v, "ab")

	if err == nil {
		t.Errorf("Expected error on string shorter than 5, got %v", err)
	}

	err = s.ValidateFieldCtx(ctx, v, "hello")

	if err != nil {
		t.Errorf("Expected no error on string of exactly 5 characters, got %v", err)
	}

	err = s.ValidateFieldCtx(ctx, v, "toolong")

	if err == nil {
		t.Errorf("Expected error on string longer than 5, got %v", err)
	}
}
