/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

// Whitebox-Test (package application, nicht application_test) für
// `mergeIntervalWarning`. Der Tie-Break-Pfad (Round-2-Befund 5 aus dem
// M6-Step-1-Review) ist heute nicht über das Public-API erreichbar,
// weil kein Check `ConfigurationInvalid` als Result.Name produziert —
// die v0.2-Worker-Pool-Erweiterung soll genau das tun, dann fängt
// dieser Test eine versehentliche Rückwärts-Tie-Break-Regression ab.
package application

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	preflightv1alpha1 "github.com/pt9912/k-deskflight/api/v1alpha1"
)

func mergeRigBaseline(severity preflightv1alpha1.Severity, message string) AggregationOutput {
	now := time.Date(2026, time.May, 19, 12, 0, 0, 0, time.UTC)
	return AggregationOutput{
		Phase: preflightv1alpha1.PhasePassed,
		Summary: preflightv1alpha1.Summary{
			ChecksTotal: 1,
			Passed:      1,
			LastChecked: ptrMetaTime(metav1.NewTime(now)),
		},
		Conditions: []preflightv1alpha1.Condition{{
			Type:               ConditionTypeConfigurationInvalid,
			Status:             metav1.ConditionTrue,
			Reason:             "PreExistingReason",
			Message:            message,
			LastTransitionTime: metav1.NewTime(now),
			Severity:           severity,
		}},
	}
}

func warningRig(severity preflightv1alpha1.Severity, message string) *ConditionWarning {
	return &ConditionWarning{
		Type:     ConditionTypeConfigurationInvalid,
		Reason:   ReasonIntervalUnparseable,
		Severity: severity,
		Message:  message,
	}
}

func findConfigurationInvalid(t *testing.T, out AggregationOutput) preflightv1alpha1.Condition {
	t.Helper()
	for _, c := range out.Conditions {
		if c.Type == ConditionTypeConfigurationInvalid {
			return c
		}
	}
	t.Fatalf("expected ConfigurationInvalid condition, got %+v", out.Conditions)
	return preflightv1alpha1.Condition{}
}

// TestMergeIntervalWarningTieBreakLowerSeverityKept: existierende
// Aggregator-Condition ist `critical`, Interval-Warning ist `warning`
// → existierende bleibt (höhere Severity gewinnt).
func TestMergeIntervalWarningTieBreakLowerSeverityKept(t *testing.T) {
	base := mergeRigBaseline(preflightv1alpha1.SeverityCritical, "existing critical")
	warn := warningRig(preflightv1alpha1.SeverityWarning, "incoming warning")
	now := time.Date(2026, time.May, 19, 12, 0, 1, 0, time.UTC)

	out := mergeIntervalWarning(base, warn, now)

	got := findConfigurationInvalid(t, out)
	if got.Severity != preflightv1alpha1.SeverityCritical {
		t.Errorf("Severity: got %q, want critical (higher pre-existing must win)", got.Severity)
	}
	if got.Reason != "PreExistingReason" {
		t.Errorf("Reason: got %q, want PreExistingReason (existing must be kept)", got.Reason)
	}
	if got.Message != "existing critical" {
		t.Errorf("Message: got %q, want %q", got.Message, "existing critical")
	}
	// Phase darf nicht unter Warning fallen — base ist Passed, escalate
	// hebt auf Warning.
	if out.Phase != preflightv1alpha1.PhaseWarning {
		t.Errorf("Phase: got %q, want Warning", out.Phase)
	}
}

// TestMergeIntervalWarningTieBreakHigherSeverityWins: existierende ist
// `info`, Warning ist `warning` → Warning ersetzt die existierende.
func TestMergeIntervalWarningTieBreakHigherSeverityWins(t *testing.T) {
	base := mergeRigBaseline(preflightv1alpha1.SeverityInfo, "existing info")
	warn := warningRig(preflightv1alpha1.SeverityWarning, "incoming warning")
	now := time.Date(2026, time.May, 19, 12, 0, 1, 0, time.UTC)

	out := mergeIntervalWarning(base, warn, now)

	got := findConfigurationInvalid(t, out)
	if got.Severity != preflightv1alpha1.SeverityWarning {
		t.Errorf("Severity: got %q, want warning (higher incoming must win)", got.Severity)
	}
	if got.Reason != ReasonIntervalUnparseable {
		t.Errorf("Reason: got %q, want %q (incoming must replace)",
			got.Reason, ReasonIntervalUnparseable)
	}
	if got.Message != "incoming warning" {
		t.Errorf("Message: got %q, want %q", got.Message, "incoming warning")
	}
}

// TestMergeIntervalWarningTieBreakEqualSeverityKeepsExisting:
// Severity-Gleichstand → existierende bleibt (stable-Verhalten).
func TestMergeIntervalWarningTieBreakEqualSeverityKeepsExisting(t *testing.T) {
	base := mergeRigBaseline(preflightv1alpha1.SeverityWarning, "existing warning")
	warn := warningRig(preflightv1alpha1.SeverityWarning, "incoming warning")
	now := time.Date(2026, time.May, 19, 12, 0, 1, 0, time.UTC)

	out := mergeIntervalWarning(base, warn, now)

	got := findConfigurationInvalid(t, out)
	if got.Reason != "PreExistingReason" {
		t.Errorf("Reason: got %q, want PreExistingReason (equal severity → keep existing)",
			got.Reason)
	}
	if got.Message != "existing warning" {
		t.Errorf("Message: got %q, want %q (no replacement on tie)",
			got.Message, "existing warning")
	}
}

// TestMergeIntervalWarningDedupOutputStaysSorted: synthetisch
// unsortierte Conditions-Slice + Dedup-Hit → Output ist trotzdem
// alphabetisch sortiert (Round-2-Befund 4).
func TestMergeIntervalWarningDedupOutputStaysSorted(t *testing.T) {
	now := time.Date(2026, time.May, 19, 12, 0, 0, 0, time.UTC)
	base := AggregationOutput{
		Phase: preflightv1alpha1.PhasePassed,
		Summary: preflightv1alpha1.Summary{
			ChecksTotal: 2,
			LastChecked: ptrMetaTime(metav1.NewTime(now)),
		},
		// Bewusst unsortiert: ZetaCheck > ConfigurationInvalid > AlphaCheck
		Conditions: []preflightv1alpha1.Condition{
			{Type: "ZetaCheck", LastTransitionTime: metav1.NewTime(now)},
			{
				Type:               ConditionTypeConfigurationInvalid,
				Severity:           preflightv1alpha1.SeverityInfo,
				LastTransitionTime: metav1.NewTime(now),
			},
			{Type: "AlphaCheck", LastTransitionTime: metav1.NewTime(now)},
		},
	}
	warn := warningRig(preflightv1alpha1.SeverityWarning, "incoming")

	out := mergeIntervalWarning(base, warn, now.Add(time.Second))

	want := []string{"AlphaCheck", ConditionTypeConfigurationInvalid, "ZetaCheck"}
	if len(out.Conditions) != len(want) {
		t.Fatalf("Conditions length: got %d, want %d", len(out.Conditions), len(want))
	}
	for i, exp := range want {
		if out.Conditions[i].Type != exp {
			t.Errorf("Conditions[%d].Type: got %q, want %q (output must be sorted)",
				i, out.Conditions[i].Type, exp)
		}
	}
}
