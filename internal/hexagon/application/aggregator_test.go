/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package application_test

import (
	"testing"
	"time"

	preflightv1alpha1 "github.com/pt9912/k-deskflight/api/v1alpha1"
	"github.com/pt9912/k-deskflight/internal/hexagon/application"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

func fixedNow() time.Time {
	return time.Date(2026, time.May, 17, 12, 0, 0, 0, time.UTC)
}

func TestAggregateEmpty(t *testing.T) {
	t.Parallel()
	out := application.Aggregate(nil, fixedNow())

	if out.Phase != preflightv1alpha1.PhasePassed {
		t.Errorf("Phase: got %q, want %q (empty results = Passed)", out.Phase, preflightv1alpha1.PhasePassed)
	}
	if out.Summary.ChecksTotal != 0 {
		t.Errorf("Summary.ChecksTotal: got %d, want 0", out.Summary.ChecksTotal)
	}
	if len(out.Conditions) != 0 {
		t.Errorf("Conditions: got %d entries, want 0", len(out.Conditions))
	}
}

func TestAggregateAllPassed(t *testing.T) {
	t.Parallel()
	results := []domain.Result{
		{Name: "AReady", Status: domain.StatusTrue, Severity: domain.SeverityInfo, LastTransition: fixedNow()},
		{Name: "BReady", Status: domain.StatusTrue, Severity: domain.SeverityInfo, LastTransition: fixedNow()},
	}

	out := application.Aggregate(results, fixedNow())

	if out.Phase != preflightv1alpha1.PhasePassed {
		t.Errorf("Phase: got %q, want Passed", out.Phase)
	}
	if got, want := out.Summary.Passed, int32(2); got != want {
		t.Errorf("Summary.Passed: got %d, want %d", got, want)
	}
	if len(out.Conditions) != 2 {
		t.Errorf("Conditions count: got %d, want 2", len(out.Conditions))
	}
}

func TestAggregateOneCriticalFailed(t *testing.T) {
	t.Parallel()
	results := []domain.Result{
		{Name: "AReady", Status: domain.StatusTrue, Severity: domain.SeverityInfo, LastTransition: fixedNow()},
		{Name: "BReady", Status: domain.StatusFalse, Severity: domain.SeverityCritical, LastTransition: fixedNow()},
		{Name: "CReady", Status: domain.StatusUnknown, Severity: domain.SeverityWarning, LastTransition: fixedNow()},
	}

	out := application.Aggregate(results, fixedNow())

	if out.Phase != preflightv1alpha1.PhaseFailed {
		t.Errorf("Phase: got %q, want Failed (critical wins)", out.Phase)
	}
	if got, want := out.Summary.Failed, int32(1); got != want {
		t.Errorf("Summary.Failed: got %d, want %d", got, want)
	}
	if got, want := out.Summary.Unknown, int32(1); got != want {
		t.Errorf("Summary.Unknown: got %d, want %d", got, want)
	}
}

func TestAggregateWarningWithoutCritical(t *testing.T) {
	t.Parallel()
	results := []domain.Result{
		{Name: "BReady", Status: domain.StatusFalse, Severity: domain.SeverityWarning, LastTransition: fixedNow()},
	}

	out := application.Aggregate(results, fixedNow())

	if out.Phase != preflightv1alpha1.PhaseWarning {
		t.Errorf("Phase: got %q, want Warning", out.Phase)
	}
	if got, want := out.Summary.Warning, int32(1); got != want {
		t.Errorf("Summary.Warning: got %d, want %d", got, want)
	}
}

func TestAggregateOnlyUnknown(t *testing.T) {
	t.Parallel()
	results := []domain.Result{
		{Name: "BReady", Status: domain.StatusUnknown, Severity: domain.SeverityWarning, LastTransition: fixedNow()},
	}

	out := application.Aggregate(results, fixedNow())

	if out.Phase != preflightv1alpha1.PhaseUnknown {
		t.Errorf("Phase: got %q, want Unknown", out.Phase)
	}
}

func TestAggregateConditionsDedupeHighestSeverityWins(t *testing.T) {
	t.Parallel()
	results := []domain.Result{
		{Name: "Ready", Status: domain.StatusTrue, Severity: domain.SeverityInfo, LastTransition: fixedNow()},
		{Name: "Ready", Status: domain.StatusFalse, Severity: domain.SeverityCritical, LastTransition: fixedNow()},
	}

	out := application.Aggregate(results, fixedNow())

	if len(out.Conditions) != 1 {
		t.Fatalf("Conditions count: got %d, want 1 (deduped)", len(out.Conditions))
	}
	if out.Conditions[0].Severity != preflightv1alpha1.SeverityCritical {
		t.Errorf("Severity: got %q, want critical (dedupe must keep highest)", out.Conditions[0].Severity)
	}
}

func TestAggregateConditionsDedupeNewerTransitionWins(t *testing.T) {
	t.Parallel()
	earlier := fixedNow()
	later := earlier.Add(time.Hour)

	results := []domain.Result{
		{Name: "Ready", Status: domain.StatusTrue, Severity: domain.SeverityInfo, Reason: "Old", LastTransition: earlier},
		{Name: "Ready", Status: domain.StatusTrue, Severity: domain.SeverityInfo, Reason: "New", LastTransition: later},
	}

	out := application.Aggregate(results, fixedNow())

	if len(out.Conditions) != 1 {
		t.Fatalf("Conditions count: got %d, want 1", len(out.Conditions))
	}
	if out.Conditions[0].Reason != "New" {
		t.Errorf("Reason: got %q, want New (newer transition wins at equal severity)", out.Conditions[0].Reason)
	}
}

func TestAggregateConditionsSortedAlphabetically(t *testing.T) {
	t.Parallel()
	results := []domain.Result{
		{Name: "ZebraReady", Status: domain.StatusTrue, Severity: domain.SeverityInfo, LastTransition: fixedNow()},
		{Name: "AppleReady", Status: domain.StatusTrue, Severity: domain.SeverityInfo, LastTransition: fixedNow()},
		{Name: "MikeReady", Status: domain.StatusTrue, Severity: domain.SeverityInfo, LastTransition: fixedNow()},
	}

	out := application.Aggregate(results, fixedNow())

	if len(out.Conditions) != 3 {
		t.Fatalf("Conditions count: got %d, want 3", len(out.Conditions))
	}
	want := []string{"AppleReady", "MikeReady", "ZebraReady"}
	for i, w := range want {
		if out.Conditions[i].Type != w {
			t.Errorf("Conditions[%d].Type: got %q, want %q", i, out.Conditions[i].Type, w)
		}
	}
}

// TestAggregateEmptySeverityCountsAsWarning deckt zwei Pfade in einer
// Pflicht-Sicherheits-Klausel:
//   - aggregator.go switch-default für ungültige/leere Severity bei
//     Status=False (zählt als Warning).
//   - aggregator.go severityRank default für Empty-String (rank 0),
//     damit der Dedupe einen leeren Severity-Eintrag gegen einen
//     non-empty stabil rangiert.
func TestAggregateEmptySeverityCountsAsWarning(t *testing.T) {
	t.Parallel()
	results := []domain.Result{
		{Name: "WeirdReady", Status: domain.StatusFalse, Severity: "", LastTransition: fixedNow()},
	}
	out := application.Aggregate(results, fixedNow())

	if out.Phase != preflightv1alpha1.PhaseWarning {
		t.Errorf("Phase: got %q, want Warning (empty severity defensive fallback)", out.Phase)
	}
	if got, want := out.Summary.Warning, int32(1); got != want {
		t.Errorf("Summary.Warning: got %d, want %d", got, want)
	}
}

// TestAggregateDedupeEmptySeverityLoses fixiert die severityRank-
// Default-Klausel: zwei Results mit gleichem Name, einer mit empty
// Severity (rank 0), einer mit Info (rank 1) — der Info-Eintrag
// gewinnt im Dedupe.
func TestAggregateDedupeEmptySeverityLoses(t *testing.T) {
	t.Parallel()
	results := []domain.Result{
		{Name: "DupeReady", Status: domain.StatusTrue, Severity: "", LastTransition: fixedNow()},
		{Name: "DupeReady", Status: domain.StatusTrue, Severity: domain.SeverityInfo, Reason: "Info", LastTransition: fixedNow()},
	}
	out := application.Aggregate(results, fixedNow())

	if len(out.Conditions) != 1 {
		t.Fatalf("Conditions count: got %d, want 1 (dedupe by name)", len(out.Conditions))
	}
	if out.Conditions[0].Severity != preflightv1alpha1.SeverityInfo {
		t.Errorf("Severity: got %q, want info (higher rank wins)", out.Conditions[0].Severity)
	}
}
