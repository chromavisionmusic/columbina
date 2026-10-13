package main

import (
	"testing"
)

func TestResolveEnvironment(t *testing.T) {
	t.Run("args provided takes precedence", func(t *testing.T) {
		t.Setenv("ENV", "production")
		got := resolveEnvironment([]string{"cmd", "staging"})
		if got != "staging" {
			t.Errorf("got %q, want 'staging'", got)
		}
	})

	t.Run("ENV environment variable used when no args", func(t *testing.T) {
		t.Setenv("ENV", "dev_custom")
		got := resolveEnvironment([]string{"cmd"})
		if got != "dev_custom" {
			t.Errorf("got %q, want 'dev_custom'", got)
		}
	})

	t.Run("defaults to production or development based on dotfiles", func(t *testing.T) {
		t.Setenv("ENV", "")
		got := resolveEnvironment([]string{"cmd"})
		if got != "production" && got != "development" {
			t.Errorf("expected 'production' or 'development', got %q", got)
		}
	})
}
