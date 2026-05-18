/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package check_test

import (
	"context"
	"testing"

	"github.com/pt9912/k-deskflight/internal/adapter/check"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

// stubCheck ist eine minimale domain.Check-Implementierung für die
// Registry-Tests — kein API-Zugriff, kein Verhalten.
type stubCheck struct {
	name string
}

func (s stubCheck) Name() string                                       { return s.name }
func (s stubCheck) SpecKind() string                                   { return s.name }
func (s stubCheck) RequiredPermissions() []domain.PermissionRequest    { return nil }
func (s stubCheck) Run(_ context.Context, _ domain.CheckSpec) domain.Result {
	return domain.Result{Name: s.name, Status: domain.StatusTrue}
}

func TestRegistryRegisterAndResolve(t *testing.T) {
	r := check.NewRegistry()
	r.Register(stubCheck{name: "alpha"})

	got, ok := r.Resolve("alpha")
	if !ok {
		t.Fatal("Resolve(alpha): ok=false, want true")
	}
	if got.Name() != "alpha" {
		t.Errorf("Resolve(alpha).Name(): got %q, want %q", got.Name(), "alpha")
	}

	if _, ok := r.Resolve("missing"); ok {
		t.Error("Resolve(missing): ok=true, want false")
	}
}

func TestRegistryListByProfileEmptySpec(t *testing.T) {
	r := check.NewRegistry()
	r.Register(stubCheck{name: "alpha"})

	active, issues := r.ListByProfile("production", map[string]domain.CheckSpec{})

	if len(active) != 0 {
		t.Errorf("active: got %d entries, want 0 (empty spec)", len(active))
	}
	if len(issues) != 0 {
		t.Errorf("issues: got %d entries, want 0 (empty spec)", len(issues))
	}
}

func TestRegistryListByProfileResolves(t *testing.T) {
	r := check.NewRegistry()
	r.Register(stubCheck{name: "alpha"})
	r.Register(stubCheck{name: "beta"})

	spec := map[string]domain.CheckSpec{
		"alpha": domain.KubernetesVersionSpec{Min: "1.0"},
		"beta":  domain.KubernetesVersionSpec{Min: "2.0"},
	}

	active, issues := r.ListByProfile("production", spec)

	if len(active) != 2 {
		t.Fatalf("active: got %d entries, want 2", len(active))
	}
	if active[0].Name() != "alpha" || active[1].Name() != "beta" {
		t.Errorf("active order: got [%q, %q], want [alpha, beta]", active[0].Name(), active[1].Name())
	}
	if len(issues) != 0 {
		t.Errorf("issues: got %d entries, want 0 (all resolved)", len(issues))
	}
}

func TestRegistryListByProfileUnknownCheck(t *testing.T) {
	r := check.NewRegistry()
	r.Register(stubCheck{name: "alpha"})

	spec := map[string]domain.CheckSpec{
		"alpha":   domain.KubernetesVersionSpec{Min: "1.0"},
		"unknown": domain.KubernetesVersionSpec{Min: "1.0"},
		"gamma":   domain.KubernetesVersionSpec{Min: "1.0"},
	}

	active, issues := r.ListByProfile("production", spec)

	if len(active) != 1 || active[0].Name() != "alpha" {
		t.Errorf("active: want only [alpha], got %v", active)
	}
	if len(issues) != 2 {
		t.Fatalf("issues: got %d entries, want 2 (unknown + gamma)", len(issues))
	}
	// alphabetic sort
	if issues[0].Name != "gamma" || issues[1].Name != "unknown" {
		t.Errorf("issues order: got [%q, %q], want [gamma, unknown]", issues[0].Name, issues[1].Name)
	}
	for _, issue := range issues {
		if issue.Reason != "UnknownCheck" {
			t.Errorf("issue %q reason: got %q, want %q", issue.Name, issue.Reason, "UnknownCheck")
		}
	}
}
