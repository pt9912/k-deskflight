/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

// Package domain hält die reine Fachdomäne der Preflight-Checks
// (architecture.md AR-004 Hexagon-Innerstes). Hier gibt es bewusst
// keine Kubernetes-/controller-runtime-Imports — depguard-Regel
// `domain-isolation` (AR-005) erzwingt das.
package domain

import (
	"context"
	"time"
)

// Severity beschreibt den operativen Schweregrad eines Check-Ergebnisses
// (architecture.md AR-006/AR-014).
type Severity string

const (
	// SeverityInfo ist der Default — rein informativ, keine Aktion nötig.
	SeverityInfo Severity = "info"

	// SeverityWarning markiert einen behebbaren oder beratenden Befund.
	SeverityWarning Severity = "warning"

	// SeverityCritical markiert einen Befund, der vor Installation behoben
	// sein muss.
	SeverityCritical Severity = "critical"
)

// ConditionStatus ist die Kubernetes-konforme True/False/Unknown-Trinität.
type ConditionStatus string

const (
	// StatusTrue zeigt einen positiv abgeschlossenen Check.
	StatusTrue ConditionStatus = "True"

	// StatusFalse zeigt einen negativ abgeschlossenen Check.
	StatusFalse ConditionStatus = "False"

	// StatusUnknown wird gesetzt, wenn der Check sein Ergebnis nicht
	// ermitteln konnte (Timeout, fehlende Berechtigung, Cluster
	// nicht erreichbar).
	StatusUnknown ConditionStatus = "Unknown"
)

// Result ist die normalisierte Ausgabe eines Check-Laufs
// (architecture.md AR-012). Aggregator (AR-014) konsumiert eine
// Liste davon und mappt sie auf Phase/Conditions/Summary der CR.
type Result struct {
	// Name ist der Condition-Type, der im CR-Status erscheint
	// (z. B. "KubernetesVersionReady").
	Name string

	// Status ist True/False/Unknown.
	Status ConditionStatus

	// Reason ist ein stabiler CamelCase-Identifier für den Aggregator
	// und für Watches.
	Reason string

	// Severity ist Pflichtfeld (AR-014); Default `info`, wenn der
	// Producer nichts setzt, ist Architekturverstoß — Producer-Pipeline
	// muss normalisieren.
	Severity Severity

	// Message ist die menschenlesbare Erklärung.
	Message string

	// LastTransition ist der Zeitpunkt, an dem dieser Status-Wert
	// erstmals erreicht wurde.
	LastTransition time.Time
}

// CheckSpec ist die typsichere Parameter-Box, die der Reconciler einem
// Check beim Run übergibt (AR-012). Die `Kind`-Methode dient zur
// Identitäts-Prüfung gegen `Check.SpecKind()` vor dem Run-Aufruf.
type CheckSpec interface {
	// Kind liefert den stabilen Spec-Typ-Token (z. B. "kubernetesVersion").
	Kind() string

	// Validate prüft Syntax-/Format-Constraints, die das CRD-OpenAPI-
	// Schema nicht ausdrücken kann. Fehler hier führen vor dem Run zu
	// `Status: Failed` + `Condition: SpecInvalid` im Reconciler.
	Validate(ctx context.Context) error
}

// Check ist die Schnittstelle, die jeder konkrete Preflight-Check
// implementiert (AR-012). Implementierungen liegen in
// `internal/adapter/check/`.
//
// Kontrakt:
//   - `Run` muss bei Kontextabbruch (`ctx.Done()`) deterministisch
//     terminieren.
//   - `Run` darf nicht panicken. Bei interner Inkonsistenz liefert die
//     Implementierung `Status: Unknown` + `Reason: InternalError`.
//   - Spec-Typ-Mismatch (`spec.Kind() != Check.SpecKind()`) wird vom
//     Aufrufer (Reconciler/Registry) abgefangen, nicht vom Check selbst.
type Check interface {
	// Name liefert den stabilen Check-Identifier
	// (z. B. "kubernetesVersion").
	Name() string

	// SpecKind liefert den Spec-Typ-Token, den dieser Check erwartet
	// (in der Regel identisch zu Name).
	SpecKind() string

	// ConditionType liefert den stabilen Condition-Type-String, den der
	// Check in seinem `Run`-Result als `Name`-Feld setzt
	// (z. B. `KubernetesVersionReady`, slice-M5 §2.2). Der Runner nutzt
	// das für synthetische Results (RBAC/Panic/Timeout/Cancel), damit
	// die Condition-Type-Identität über alle Reconcile-Pfade konsistent
	// bleibt und Anwender keine doppelten Conditions sehen.
	ConditionType() string

	// RequiredPermissions deklariert die Cluster-Rechte, die der Check
	// für seinen `Run` benötigt (slice-M5 §2.2, AR-018). Der Reconciler
	// ruft `port.AccessReviewer.CanI` für jede zurückgegebene Permission
	// vor dem `Run` auf; fehlende Rechte führen zu `Status: Unknown` +
	// `Reason: RBACInsufficient`. Discovery-basierte Checks
	// (z. B. ServerVersion, ServerGroups) brauchen keine zusätzlichen
	// Rechte und liefern eine leere Slice.
	RequiredPermissions() []PermissionRequest

	// Run führt den Check aus und liefert ein normalisiertes Result.
	Run(ctx context.Context, spec CheckSpec) Result
}
