package validator

import (
	"testing"
)

func TestInt(t *testing.T) {
	s := New().Int("Must be a number")

	if s.compileRules() != "number" {
		t.Errorf("Expected number, got %s", s.compileRules())
	}

	ctx := t.Context()
	err := s.ValidateCtx(ctx, "1")

	if err != nil {
		t.Errorf("Expected no error on numeric string, got %v", err)
	}

	err = s.ValidateCtx(ctx, 11)

	if err != nil {
		t.Errorf("Expected no error on integer, got %v", err)
	}
}

func TestMin(t *testing.T) {
	s := New().Min(10, "Must be at least 10 characters")

	ctx := t.Context()
	err := s.ValidateCtx(ctx, "hello")

	if err == nil {
		t.Errorf("Expected error on string, got %v", err)
	}

	err = s.ValidateCtx(ctx, "hello world!")

	if err != nil {
		t.Errorf("Expected no error on string, got %v", err)
	}
}

func TestMax(t *testing.T) {
	s := New().Max(10, "Must be at most 10 characters")

	ctx := t.Context()
	err := s.ValidateCtx(ctx, "hello")

	if err != nil {
		t.Errorf("Expected error on string, got %v", err)
	}

	err = s.ValidateCtx(ctx, "hello world")

	if err == nil {
		t.Errorf("Expected error on string, got %v", err)
	}
}

func TestRequired(t *testing.T) {
	s := New().Required("This field is required")

	ctx := t.Context()
	err := s.ValidateCtx(ctx, "")

	if err == nil {
		t.Errorf("Expected error on empty string, got %v", err)
	}

	err = s.ValidateCtx(ctx, "hello")

	if err != nil {
		t.Errorf("Expected no error on string, got %v", err)
	}
}

func TestEmail(t *testing.T) {
	s := New().Email("Must be a valid email")

	ctx := t.Context()
	err := s.ValidateCtx(ctx, "not-an-email")

	if err == nil {
		t.Errorf("Expected error on invalid email, got %v", err)
	}

	err = s.ValidateCtx(ctx, "test@example.com")

	if err != nil {
		t.Errorf("Expected no error on valid email, got %v", err)
	}
}
