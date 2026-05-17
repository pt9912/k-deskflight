/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package check_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/pt9912/k-deskflight/internal/adapter/check"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

// fakeKubernetesAPI ist ein test-doubles für port.KubernetesAPI.
type fakeKubernetesAPI struct {
	version string
	err     error
}

func (f fakeKubernetesAPI) ServerVersion(_ context.Context) (string, error) {
	return f.version, f.err
}

// otherSpec ist ein CheckSpec mit anderer Kind, um den
// Spec-Mismatch-Pfad zu testen.
type otherSpec struct{}

func (otherSpec) Kind() string                          { return "other" }
func (otherSpec) Validate(_ context.Context) error      { return nil }

func fixedClock() func() time.Time {
	return func() time.Time {
		return time.Date(2026, time.May, 17, 12, 0, 0, 0, time.UTC)
	}
}

func TestKubernetesVersionPassed(t *testing.T) {
	t.Parallel()
	api := fakeKubernetesAPI{version: "v1.34.2"}
	chk := check.NewKubernetesVersion(api, fixedClock())

	res := chk.Run(context.Background(), domain.KubernetesVersionSpec{Min: "1.34"})

	if res.Status != domain.StatusTrue {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusTrue)
	}
	if res.Reason != "KubernetesVersionReady" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "KubernetesVersionReady")
	}
	if res.Severity != domain.SeverityInfo {
		t.Errorf("Severity: got %q, want %q", res.Severity, domain.SeverityInfo)
	}
	if res.Name != "KubernetesVersionReady" {
		t.Errorf("Name: got %q, want %q", res.Name, "KubernetesVersionReady")
	}
}

func TestKubernetesVersionFailed(t *testing.T) {
	t.Parallel()
	api := fakeKubernetesAPI{version: "v1.34.2"}
	chk := check.NewKubernetesVersion(api, fixedClock())

	res := chk.Run(context.Background(), domain.KubernetesVersionSpec{Min: "99.99"})

	if res.Status != domain.StatusFalse {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusFalse)
	}
	if res.Reason != "KubernetesVersionTooOld" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "KubernetesVersionTooOld")
	}
	if res.Severity != domain.SeverityCritical {
		t.Errorf("Severity: got %q, want %q", res.Severity, domain.SeverityCritical)
	}
}

func TestKubernetesVersionServerLookupFailed(t *testing.T) {
	t.Parallel()
	api := fakeKubernetesAPI{err: errors.New("connection refused")}
	chk := check.NewKubernetesVersion(api, fixedClock())

	res := chk.Run(context.Background(), domain.KubernetesVersionSpec{Min: "1.34"})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "ServerVersionLookupFailed" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "ServerVersionLookupFailed")
	}
}

func TestKubernetesVersionServerParseFailed(t *testing.T) {
	t.Parallel()
	api := fakeKubernetesAPI{version: "not-a-version"}
	chk := check.NewKubernetesVersion(api, fixedClock())

	res := chk.Run(context.Background(), domain.KubernetesVersionSpec{Min: "1.34"})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "ServerVersionParseFailed" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "ServerVersionParseFailed")
	}
}

func TestKubernetesVersionMinParseFailed(t *testing.T) {
	t.Parallel()
	api := fakeKubernetesAPI{version: "v1.34.2"}
	chk := check.NewKubernetesVersion(api, fixedClock())

	res := chk.Run(context.Background(), domain.KubernetesVersionSpec{Min: "garbage"})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "MinVersionParseFailed" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "MinVersionParseFailed")
	}
}

func TestKubernetesVersionInvalidSpec(t *testing.T) {
	t.Parallel()
	api := fakeKubernetesAPI{version: "v1.34.2"}
	chk := check.NewKubernetesVersion(api, fixedClock())

	res := chk.Run(context.Background(), otherSpec{})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "InvalidSpec" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "InvalidSpec")
	}
}

func TestKubernetesVersionStripsBuildSuffix(t *testing.T) {
	t.Parallel()
	api := fakeKubernetesAPI{version: "v1.34.2+abc"}
	chk := check.NewKubernetesVersion(api, fixedClock())

	res := chk.Run(context.Background(), domain.KubernetesVersionSpec{Min: "1.34"})

	if res.Status != domain.StatusTrue {
		t.Errorf("Status: got %q, want %q (build suffix should not break compare)", res.Status, domain.StatusTrue)
	}
}
