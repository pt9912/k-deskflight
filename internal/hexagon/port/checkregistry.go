/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package port

import "github.com/pt9912/k-deskflight/internal/hexagon/domain"

// CheckSelectionIssue beschreibt einen einzelnen, nicht auflösbaren
// Check-Spec-Eintrag (architecture.md AR-013).
type CheckSelectionIssue struct {
	// Name ist der Check-Identifier, der nicht aufgelöst werden konnte.
	Name string

	// Reason ist ein stabiler Bezeichner: aktuell `UnknownCheck` oder
	// `CheckNotAllowedInProfile`.
	Reason string
}

// CheckRegistry verwaltet die Liste der dem Operator bekannten Checks
// und liefert profil-aufgelöste Auswahlen für den Reconciler
// (architecture.md AR-013).
type CheckRegistry interface {
	// Register fügt einen Check zur Registry hinzu. Doppel-Registrierung
	// unter demselben Namen ist ein Programmierfehler; M3 erlaubt
	// Überschreiben (Adapter-Implementierung dokumentiert das).
	Register(c domain.Check)

	// Resolve liefert den Check unter dem gegebenen Namen, oder
	// `(_, false)`, wenn er nicht registriert ist. Für explizite
	// Direktanfragen.
	Resolve(name string) (domain.Check, bool)

	// ListByProfile liefert die aktivierbaren Checks für ein Profil
	// inkl. Issues. Erste Rückgabe sind die aufgelösten Checks; zweite
	// Rückgabe enthält Tupel `(Name, Reason)` für nicht auflösbare
	// Check-Namen. Issues sind dedupliziert und alphabetisch sortiert.
	ListByProfile(
		profile string,
		spec map[string]domain.CheckSpec,
	) ([]domain.Check, []CheckSelectionIssue)
}
