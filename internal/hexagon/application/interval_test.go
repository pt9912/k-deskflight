/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package application_test

import (
	"strings"
	"testing"
	"time"

	preflightv1alpha1 "github.com/pt9912/k-deskflight/api/v1alpha1"
	"github.com/pt9912/k-deskflight/internal/hexagon/application"
)

// intervalCaseExpectation bündelt die erwartete Auswertung pro
// Tabellen-Zeile in `TestNormalizeInterval`. `reasonSub` ist optional
// — nur gesetzt, wenn die Warning-Message einen bestimmten Sub-String
// enthalten muss.
type intervalCaseExpectation struct {
	duration  time.Duration
	warning   bool
	reasonSub string
}

type intervalCase struct {
	name  string
	input *string
	want  intervalCaseExpectation
}

// normalizeIntervalCases hält die §2.3.1-Klassifikations-Tabelle als
// separate Helper-Funktion, damit der Test-Body unter dem funlen-
// Threshold bleibt.
func normalizeIntervalCases() []intervalCase {
	stringPtr := func(s string) *string { return &s }
	return []intervalCase{
		{name: "nil pointer falls back to default", input: nil,
			want: intervalCaseExpectation{duration: application.DefaultInterval}},
		{name: "empty string falls back to default", input: stringPtr(""),
			want: intervalCaseExpectation{duration: application.DefaultInterval}},
		{name: "zero duration clamps to min", input: stringPtr("0s"),
			want: intervalCaseExpectation{duration: application.MinInterval, warning: true, reasonSub: "below minimum"}},
		{name: "below-min duration clamps to min", input: stringPtr("15s"),
			want: intervalCaseExpectation{duration: application.MinInterval, warning: true, reasonSub: "below minimum"}},
		{name: "min boundary passes through", input: stringPtr("30s"),
			want: intervalCaseExpectation{duration: application.MinInterval}},
		{name: "default boundary passes through", input: stringPtr("5m"),
			want: intervalCaseExpectation{duration: 5 * time.Minute}},
		{name: "composite duration passes through", input: stringPtr("1h30m"),
			want: intervalCaseExpectation{duration: 90 * time.Minute}},
		{name: "max boundary passes through", input: stringPtr("24h"),
			want: intervalCaseExpectation{duration: application.MaxInterval}},
		{name: "above-max duration clamps to max", input: stringPtr("25h"),
			want: intervalCaseExpectation{duration: application.MaxInterval, warning: true, reasonSub: "exceeds maximum"}},
		{name: "parse-fail falls back to default", input: stringPtr("abc"),
			want: intervalCaseExpectation{duration: application.DefaultInterval, warning: true, reasonSub: "not a valid Go duration"}},
		{name: "negative duration clamps to min", input: stringPtr("-5m"),
			want: intervalCaseExpectation{duration: application.MinInterval, warning: true, reasonSub: "below minimum"}},
	}
}

// assertIntervalWarning prüft die ConditionWarning-Felder gegen die
// Erwartung. Ausgelagert, damit der TestNormalizeInterval-Body knapp
// bleibt (funlen).
func assertIntervalWarning(t *testing.T, got *application.ConditionWarning, want intervalCaseExpectation) {
	t.Helper()
	if !want.warning {
		if got != nil {
			t.Errorf("expected no warning, got %+v", got)
		}
		return
	}
	if got == nil {
		t.Fatalf("expected ConditionWarning, got nil")
	}
	if got.Type != application.ConditionTypeConfigurationInvalid {
		t.Errorf("warning.Type: got %q, want %q", got.Type, application.ConditionTypeConfigurationInvalid)
	}
	if got.Reason != application.ReasonIntervalNormalized {
		t.Errorf("warning.Reason: got %q, want %q", got.Reason, application.ReasonIntervalNormalized)
	}
	if got.Severity != preflightv1alpha1.SeverityWarning {
		t.Errorf("warning.Severity: got %q, want %q", got.Severity, preflightv1alpha1.SeverityWarning)
	}
	if want.reasonSub != "" && !strings.Contains(got.Message, want.reasonSub) {
		t.Errorf("warning.Message %q does not contain %q", got.Message, want.reasonSub)
	}
}

// TestNormalizeInterval deckt die §2.3.1-Klassifikations-Tabelle des
// Slice-M6-Plans ab. Jede Zeile der Tabelle ist ein eigener Case.
func TestNormalizeInterval(t *testing.T) {
	for _, tc := range normalizeIntervalCases() {
		t.Run(tc.name, func(t *testing.T) {
			got, warning := application.NormalizeInterval(tc.input)
			if got != tc.want.duration {
				t.Errorf("duration: got %s, want %s", got, tc.want.duration)
			}
			assertIntervalWarning(t, warning, tc.want)
		})
	}
}

// TestNormalizeIntervalBoundsAreOrdered ist ein kleiner Sanity-Check
// gegen Off-by-one-Brüche in den Konstanten (slice-M6 §2.3.1).
func TestNormalizeIntervalBoundsAreOrdered(t *testing.T) {
	if application.MinInterval >= application.DefaultInterval {
		t.Errorf("MinInterval (%s) must be < DefaultInterval (%s)",
			application.MinInterval, application.DefaultInterval)
	}
	if application.DefaultInterval >= application.MaxInterval {
		t.Errorf("DefaultInterval (%s) must be < MaxInterval (%s)",
			application.DefaultInterval, application.MaxInterval)
	}
}
