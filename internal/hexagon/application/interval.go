/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package application

import (
	"fmt"
	"time"

	preflightv1alpha1 "github.com/pt9912/k-deskflight/api/v1alpha1"
)

// Interval-Bounds gemäß `architecture.md` AR-010 + slice-M6 §2.3.
const (
	// DefaultInterval ist der Fallback, wenn `Spec.Interval` leer
	// gelassen wird oder vom Normalisierer als nicht-interpretierbar
	// eingestuft wird (Parse-Fehler).
	DefaultInterval = 5 * time.Minute

	// MinInterval ist die untere Grenze des erlaubten Intervalls;
	// kleinere Werte werden auf diesen Wert geklemmt.
	MinInterval = 30 * time.Second

	// MaxInterval ist die obere Grenze; größere Werte werden auf
	// diesen Wert geklemmt.
	MaxInterval = 24 * time.Hour
)

// CR-Spec-Scope-Konfigurations-Diagnose-Konstanten (AR-010.1).
//
// **Naming-Konvention für Folge-Slices** (slice-M6 §4 Step-1-Review-
// Fixup Befund 7 + Round-2-Befund 3): künftige `ConfigurationInvalid`-
// Reasons (z. B. v0.2 Worker-Pool-Felder, M7 Leader-Election-
// Konflikte) folgen dem Muster `Reason<FeatureName><State>`. „State"
// ist der konkrete Fehlerfall (Unparseable/ClampedMin/ClampedMax/
// Defaulted) — **nicht** ein Sammel-Reason wie „Normalized", damit
// Anwender per `kubectl get … -o jsonpath` und Prometheus-Alerts den
// Fall trennscharf filtern können. Alle Reasons leben unter demselben
// `ConditionTypeConfigurationInvalid`-Type, mit Severity-Tie-Break in
// `mergeIntervalWarning` (siehe `reconciler.go`).
const (
	// ConditionTypeConfigurationInvalid ist der Condition-Type für
	// CR-Spec-Scope-Konfigurationsfehler (AR-010.1). Aktuell nur von
	// `NormalizeInterval` ausgelöst; v0.2 ergänzt andere CR-Spec-
	// Konfigurations-Probleme (Worker-Pool-Felder etc.) unter dem
	// gleichen Type.
	ConditionTypeConfigurationInvalid = "ConfigurationInvalid"

	// ReasonIntervalUnparseable: `time.ParseDuration` schlägt fehl;
	// der Reconciler fällt auf `DefaultInterval` zurück.
	ReasonIntervalUnparseable = "IntervalUnparseable"

	// ReasonIntervalClampedMin: parsed Wert `< MinInterval`; auf
	// `MinInterval` geklemmt.
	ReasonIntervalClampedMin = "IntervalClampedMin"

	// ReasonIntervalClampedMax: parsed Wert `> MaxInterval`; auf
	// `MaxInterval` geklemmt.
	ReasonIntervalClampedMax = "IntervalClampedMax"
)

// ConditionWarning trägt eine vom Reconciler erzeugte
// `ConfigurationInvalid`-Warning (oder vergleichbare
// CR-Spec-Scope-Diagnose). Der Reconciler integriert sie nach
// `Aggregate` in das `AggregationOutput` (slice-M6 §2.3). Ein
// `nil`-Pointer bedeutet „keine Warning, alles in Ordnung".
type ConditionWarning struct {
	Type     string
	Reason   string
	Severity preflightv1alpha1.Severity
	Message  string
}

// NormalizeInterval übersetzt den `Spec.Interval`-Wert in eine
// `time.Duration` plus optionalem `ConditionWarning` gemäß
// slice-M6 §2.3.1-Klassifikations-Regel:
//
//   - `raw == ""` → Default `5m`, keine Warning (Anwender hat das
//     Feld nicht gesetzt, das ist die normale Default-Aktivierung —
//     kein Konfigurationsfehler).
//   - `time.ParseDuration` schlägt fehl → Default `5m` + Warning.
//   - Parsed Duration `< MinInterval` → clamp `MinInterval` + Warning.
//   - Parsed Duration `> MaxInterval` → clamp `MaxInterval` + Warning.
//   - Sonst → unverändert übernehmen, keine Warning.
//
// Die Funktion ist `application`-Layer-konform: sie hängt nur an
// `time` aus stdlib und am `api/v1alpha1`-Severity-Enum, importiert
// **keine** k8s-Pakete (depguard-Regel `domain-isolation` aus
// `.golangci.yml`).
//
// `raw` ist plain `string` (nicht `*string`) — Pointer wäre
// semantisch wertlos gewesen (slice-M6 Step-1-Review-Fixup Befund 5):
// `""` und „nicht gesetzt" werden identisch behandelt.
func NormalizeInterval(raw string) (time.Duration, *ConditionWarning) {
	if raw == "" {
		return DefaultInterval, nil
	}

	parsed, err := time.ParseDuration(raw)
	if err != nil {
		return DefaultInterval, &ConditionWarning{
			Type:     ConditionTypeConfigurationInvalid,
			Reason:   ReasonIntervalUnparseable,
			Severity: preflightv1alpha1.SeverityWarning,
			Message: fmt.Sprintf(
				"spec.interval %q is not a valid Go duration (%s); falling back to default %s",
				raw, err.Error(), DefaultInterval,
			),
		}
	}

	if parsed < MinInterval {
		return MinInterval, &ConditionWarning{
			Type:     ConditionTypeConfigurationInvalid,
			Reason:   ReasonIntervalClampedMin,
			Severity: preflightv1alpha1.SeverityWarning,
			Message: fmt.Sprintf(
				"spec.interval %s is below minimum %s; clamped to %s",
				parsed, MinInterval, MinInterval,
			),
		}
	}

	if parsed > MaxInterval {
		return MaxInterval, &ConditionWarning{
			Type:     ConditionTypeConfigurationInvalid,
			Reason:   ReasonIntervalClampedMax,
			Severity: preflightv1alpha1.SeverityWarning,
			Message: fmt.Sprintf(
				"spec.interval %s exceeds maximum %s; clamped to %s",
				parsed, MaxInterval, MaxInterval,
			),
		}
	}

	return parsed, nil
}
