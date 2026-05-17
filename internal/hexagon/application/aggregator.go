/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package application

import (
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	preflightv1alpha1 "github.com/pt9912/k-deskflight/api/v1alpha1"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

// AggregationOutput bündelt das Schreibe-Tripel für den CR-Status
// (architecture.md AR-014).
type AggregationOutput struct {
	Phase      preflightv1alpha1.Phase
	Summary    preflightv1alpha1.Summary
	Conditions []preflightv1alpha1.Condition
}

// Aggregate mappt eine Liste von Check-Results auf die Status-Schreibe-
// Ebene gemäß AR-014 / LH-F-031.
//
// Zähl-Logik:
//   - Status=True            → Summary.Passed++
//   - Status=False/critical  → Summary.Failed++
//   - Status=False/warning   → Summary.Warning++
//   - Status=False/info      → Summary.Warning++ (informeller Fail)
//   - Status=Unknown         → Summary.Unknown++
//
// Phasen-Mapping (höchste Priorität gewinnt):
//   - Mindestens ein Failed-critical → Failed
//   - Sonst mindestens ein Warning   → Warning
//   - Sonst mindestens ein Unknown   → Unknown
//   - Sonst                          → Passed
//
// Conditions-Pfad: Results werden nach Name dedupliziert (höchste
// Severity gewinnt, dann neueste LastTransition), alphabetisch nach
// Name sortiert, und in api/v1alpha1.Condition konvertiert.
func Aggregate(results []domain.Result, now time.Time) AggregationOutput {
	summary := preflightv1alpha1.Summary{
		ChecksTotal: int32(len(results)),
		LastChecked: ptrMetaTime(metav1.NewTime(now.UTC())),
	}

	type bucket struct {
		hasFailedCritical bool
		hasFailedNonCrit  bool
		hasUnknown        bool
	}
	var b bucket

	for _, r := range results {
		switch r.Status {
		case domain.StatusTrue:
			summary.Passed++
		case domain.StatusFalse:
			switch r.Severity {
			case domain.SeverityCritical:
				summary.Failed++
				b.hasFailedCritical = true
			case domain.SeverityWarning, domain.SeverityInfo:
				summary.Warning++
				b.hasFailedNonCrit = true
			default:
				// Architekturverstoß per AR-014: leere/ungültige Severity.
				// Sicherheitsfangnetz: als Warning zählen.
				summary.Warning++
				b.hasFailedNonCrit = true
			}
		case domain.StatusUnknown:
			summary.Unknown++
			b.hasUnknown = true
		}
	}

	phase := derivePhase(b.hasFailedCritical, b.hasFailedNonCrit, b.hasUnknown, len(results))

	conditions := dedupeAndSortConditions(results)

	return AggregationOutput{
		Phase:      phase,
		Summary:    summary,
		Conditions: conditions,
	}
}

func derivePhase(hasFailedCritical, hasFailedNonCrit, hasUnknown bool, total int) preflightv1alpha1.Phase {
	switch {
	case hasFailedCritical:
		return preflightv1alpha1.PhaseFailed
	case hasFailedNonCrit:
		return preflightv1alpha1.PhaseWarning
	case hasUnknown:
		return preflightv1alpha1.PhaseUnknown
	case total == 0:
		// Reconcile ohne Checks (M2-Fall, M3 nur bei leerer ChecksSpec):
		// Passed mit leeren Counters.
		return preflightv1alpha1.PhasePassed
	default:
		return preflightv1alpha1.PhasePassed
	}
}

// severityRank macht die AR-014-Dedupe deterministisch:
// critical > warning > info > (unbekannt).
func severityRank(s domain.Severity) int {
	switch s {
	case domain.SeverityCritical:
		return 3
	case domain.SeverityWarning:
		return 2
	case domain.SeverityInfo:
		return 1
	default:
		return 0
	}
}

func dedupeAndSortConditions(results []domain.Result) []preflightv1alpha1.Condition {
	byName := make(map[string]domain.Result, len(results))
	for _, r := range results {
		existing, present := byName[r.Name]
		if !present {
			byName[r.Name] = r
			continue
		}
		// AR-014: höhere Severity gewinnt; bei Gleichstand jüngeres
		// LastTransition; bei Gleichstand bleibt der erste Eintrag
		// (stable-Verhalten).
		switch {
		case severityRank(r.Severity) > severityRank(existing.Severity):
			byName[r.Name] = r
		case severityRank(r.Severity) == severityRank(existing.Severity) &&
			r.LastTransition.After(existing.LastTransition):
			byName[r.Name] = r
		}
	}

	out := make([]preflightv1alpha1.Condition, 0, len(byName))
	for _, r := range byName {
		out = append(out, preflightv1alpha1.Condition{
			Type:               r.Name,
			Status:             metav1.ConditionStatus(r.Status),
			Reason:             r.Reason,
			Message:            r.Message,
			LastTransitionTime: metav1.NewTime(r.LastTransition),
			Severity:           preflightv1alpha1.Severity(r.Severity),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Type < out[j].Type
	})
	return out
}

func ptrMetaTime(t metav1.Time) *metav1.Time { return &t }
