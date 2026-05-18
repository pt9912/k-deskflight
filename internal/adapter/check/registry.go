/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

// Package check enthält die konkreten Check-Implementierungen
// (architecture.md AR-004 Adapter-Layer) sowie den Map-basierten
// CheckRegistry.
package check

import (
	"sort"
	"sync"

	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

// Registry ist eine threadsafe Map-Implementierung von
// `port.CheckRegistry` (architecture.md AR-013). In M3 ist
// `ListByProfile` profil-agnostisch (alle registrierten Checks gelten
// für alle Profile); M4 differenziert das, sobald mehrere Checks
// existieren und Profile unterschiedlich aktivieren.
type Registry struct {
	mu     sync.RWMutex
	checks map[string]domain.Check
}

// NewRegistry liefert eine leere, einsatzbereite Registry.
func NewRegistry() *Registry {
	return &Registry{checks: make(map[string]domain.Check)}
}

// Register fügt einen Check unter `c.Name()` ein. Doppel-Registrierung
// überschreibt — Wiring in cmd/operator/main.go ist heute single-shot,
// also kein Programmier-Risiko; ein expliziter Panic-on-dup wäre für
// M3 zu defensiv.
func (r *Registry) Register(c domain.Check) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.checks[c.Name()] = c
}

// Resolve liefert den Check unter dem gegebenen Namen.
func (r *Registry) Resolve(name string) (domain.Check, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.checks[name]
	return c, ok
}

// All liefert einen Snapshot aller registrierten Checks (slice-M5
// Review-Befund 5). Reihenfolge folgt der Go-Map-Iteration und ist
// damit non-deterministisch — Aufrufer sortieren bei Bedarf nach
// `Check.Name()`.
func (r *Registry) All() []domain.Check {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.Check, 0, len(r.checks))
	for _, c := range r.checks {
		out = append(out, c)
	}
	return out
}

// ListByProfile löst die Spec-Map zu aktivierbaren Checks auf. In M3
// gelten alle registrierten Checks für alle Profile — Issues entstehen
// nur bei unbekannten Spec-Namen. M4 ergänzt
// `CheckNotAllowedInProfile` mit echten Profil-Allow-Listen.
//
// Die Issue-Liste ist dedupliziert und alphabetisch nach `Name`
// sortiert (AR-013).
func (r *Registry) ListByProfile(
	_ string,
	spec map[string]domain.CheckSpec,
) ([]domain.Check, []port.CheckSelectionIssue) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	active := make([]domain.Check, 0, len(spec))
	issueSet := make(map[string]port.CheckSelectionIssue)

	for name := range spec {
		check, ok := r.checks[name]
		if !ok {
			issueSet[name] = port.CheckSelectionIssue{
				Name:   name,
				Reason: "UnknownCheck",
			}
			continue
		}
		active = append(active, check)
	}

	sort.SliceStable(active, func(i, j int) bool {
		return active[i].Name() < active[j].Name()
	})

	issues := make([]port.CheckSelectionIssue, 0, len(issueSet))
	for _, issue := range issueSet {
		issues = append(issues, issue)
	}
	sort.SliceStable(issues, func(i, j int) bool {
		return issues[i].Name < issues[j].Name
	})

	return active, issues
}
