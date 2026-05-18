/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package check_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/pt9912/k-deskflight/internal/adapter/check"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

type fakeStorageClassDiscovery struct {
	classes []port.StorageClassInfo
	err     error
}

func (f fakeStorageClassDiscovery) ListStorageClasses(_ context.Context) ([]port.StorageClassInfo, error) {
	return f.classes, f.err
}

func TestStorageClassPassedNamesOnly(t *testing.T) {
	t.Parallel()
	disc := fakeStorageClassDiscovery{classes: []port.StorageClassInfo{
		{Name: "standard", IsDefault: true},
		{Name: "fast"},
	}}
	chk := check.NewStorageClass(disc, fixedClock())

	res := chk.Run(context.Background(), domain.StorageClassSpec{Names: []string{"standard", "fast"}})

	if res.Status != domain.StatusTrue {
		t.Errorf("Status: got %q, want %q (message=%q)", res.Status, domain.StatusTrue, res.Message)
	}
	if res.Reason != "StorageClassReady" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "StorageClassReady")
	}
	if res.Severity != domain.SeverityInfo {
		t.Errorf("Severity: got %q, want %q", res.Severity, domain.SeverityInfo)
	}
}

func TestStorageClassPassedRequireDefault(t *testing.T) {
	t.Parallel()
	disc := fakeStorageClassDiscovery{classes: []port.StorageClassInfo{
		{Name: "standard", IsDefault: true},
	}}
	chk := check.NewStorageClass(disc, fixedClock())

	res := chk.Run(context.Background(), domain.StorageClassSpec{Names: []string{"standard"}, RequireDefault: true})

	if res.Status != domain.StatusTrue {
		t.Errorf("Status: got %q, want %q (message=%q)", res.Status, domain.StatusTrue, res.Message)
	}
}

func TestStorageClassFailedNameMissing(t *testing.T) {
	t.Parallel()
	disc := fakeStorageClassDiscovery{classes: []port.StorageClassInfo{
		{Name: "standard", IsDefault: true},
	}}
	chk := check.NewStorageClass(disc, fixedClock())

	res := chk.Run(context.Background(), domain.StorageClassSpec{Names: []string{"standard", "fast"}})

	if res.Status != domain.StatusFalse {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusFalse)
	}
	if res.Reason != "StorageClassMissing" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "StorageClassMissing")
	}
	if res.Severity != domain.SeverityCritical {
		t.Errorf("Severity: got %q, want %q", res.Severity, domain.SeverityCritical)
	}
	if !strings.Contains(res.Message, "fast") {
		t.Errorf("Message should name the missing class: got %q", res.Message)
	}
}

func TestStorageClassFailedDefaultMissing(t *testing.T) {
	t.Parallel()
	disc := fakeStorageClassDiscovery{classes: []port.StorageClassInfo{
		{Name: "standard", IsDefault: false},
	}}
	chk := check.NewStorageClass(disc, fixedClock())

	res := chk.Run(context.Background(), domain.StorageClassSpec{Names: []string{"standard"}, RequireDefault: true})

	if res.Status != domain.StatusFalse {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusFalse)
	}
	if res.Reason != "DefaultStorageClassMissing" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "DefaultStorageClassMissing")
	}
}

func TestStorageClassFailedNameAndDefaultMissing(t *testing.T) {
	t.Parallel()
	disc := fakeStorageClassDiscovery{classes: []port.StorageClassInfo{
		{Name: "standard", IsDefault: false},
	}}
	chk := check.NewStorageClass(disc, fixedClock())

	res := chk.Run(context.Background(), domain.StorageClassSpec{Names: []string{"missing"}, RequireDefault: true})

	if res.Status != domain.StatusFalse {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusFalse)
	}
	if res.Reason != "StorageClassMissing" {
		t.Errorf("Reason: got %q, want %q (names-missing takes precedence over default-missing)", res.Reason, "StorageClassMissing")
	}
	if !strings.Contains(res.Message, "missing") {
		t.Errorf("Message should diagnose both: got %q", res.Message)
	}
}

func TestStorageClassLookupFailed(t *testing.T) {
	t.Parallel()
	disc := fakeStorageClassDiscovery{err: errors.New("forbidden")}
	chk := check.NewStorageClass(disc, fixedClock())

	res := chk.Run(context.Background(), domain.StorageClassSpec{Names: []string{"standard"}})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "StorageClassLookupFailed" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "StorageClassLookupFailed")
	}
}

func TestStorageClassInvalidSpec(t *testing.T) {
	t.Parallel()
	disc := fakeStorageClassDiscovery{}
	chk := check.NewStorageClass(disc, fixedClock())

	res := chk.Run(context.Background(), otherSpec{})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "InvalidSpec" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "InvalidSpec")
	}
}

// LastTransition wird auf fixedClock() gesetzt, damit der Test deterministisch ist.
func TestStorageClassLastTransitionUsesClock(t *testing.T) {
	t.Parallel()
	disc := fakeStorageClassDiscovery{classes: []port.StorageClassInfo{
		{Name: "standard", IsDefault: true},
	}}
	chk := check.NewStorageClass(disc, fixedClock())

	res := chk.Run(context.Background(), domain.StorageClassSpec{Names: []string{"standard"}})

	want := time.Date(2026, time.May, 17, 12, 0, 0, 0, time.UTC)
	if !res.LastTransition.Equal(want) {
		t.Errorf("LastTransition: got %v, want %v", res.LastTransition, want)
	}
}
