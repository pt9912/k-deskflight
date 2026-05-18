/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package application_test

import (
	"testing"

	"github.com/pt9912/k-deskflight/internal/audit/markers"
)

// TestNoDestructiveActionsOnForeignResources verifiziert `LH-SEC-005`:
// Der Reconciler darf keine destruktiven Verben (delete, deletecollection,
// patch, update, create) gegen fremde produktive Ressourcen führen.
// Whitelist deckt die wenigen legitimen Eigen- und Operative-Use-Cases.
//
// Bricht der Test:
//
//  1. Wurde der Marker bewusst hinzugefügt? Dann allowedDestructive
//     erweitern und im Kommentar Lastenheft-/ADR-Verweis ergänzen.
//  2. War es versehentlich? Marker streichen und RBAC neu generieren.
func TestNoDestructiveActionsOnForeignResources(t *testing.T) {
	// destructiveVerbs: was `LH-SEC-005` als „destruktive Aktion" markiert.
	// `update`/`create` sind dabei, weil sie auf fremde Ressourcen ebenso
	// gefährlich wären — `allowedDestructive` (unten) fängt die legitimen
	// Eigenpfade explizit ab.
	destructiveVerbs := map[string]bool{
		"delete":           true,
		"deletecollection": true,
		"patch":            true,
		"update":           true,
		"create":           true,
	}

	// allowedDestructive whitelisted Resource × Verb-Kombinationen.
	// **Erweiterungs-Pflicht:** kommt ein neuer Marker mit destruktivem
	// Verb hinzu, MUSS er hier mit Lastenheft-/ADR-Verweis aufgenommen
	// werden — sonst bricht der Test.
	allowedDestructive := map[string]map[string]bool{
		// Eigene CR — Anwender-Verwaltung (LH-F-001..LH-F-004).
		"opendeskpreflightchecks": {
			"create":           true,
			"update":           true,
			"patch":            true,
			"delete":           true,
			"deletecollection": true,
		},
		// Eigener Status — Reconciler-Pfad (LH-F-004, AR-009 §6).
		"opendeskpreflightchecks/status": {
			"update": true,
			"patch":  true,
		},
		// SAR-Subsystem (LH-F-024, AR-018).
		"selfsubjectaccessreviews": {"create": true},
		"selfsubjectrulesreviews":  {"create": true},
		// Leader-Election-Leases (AR-026, M2-Pre-Grant für M7-Aktivierung).
		"leases": {
			"create": true,
			"update": true,
			"patch":  true,
		},
	}

	parsed, err := markers.ParseRBACFile("reconciler.go")
	if err != nil {
		t.Fatalf("ParseRBACFile: %v", err)
	}
	if len(parsed) == 0 {
		t.Fatalf("no +kubebuilder:rbac: markers found — parser ran on the wrong file?")
	}

	var failures []string
	for _, m := range parsed {
		for _, tr := range m.Expand() {
			if !destructiveVerbs[tr.Verb] {
				continue
			}
			if allowedDestructive[tr.Resource][tr.Verb] {
				continue
			}
			failures = append(failures,
				m.Position+": destructive verb "+tr.Verb+
					" on resource "+tr.Resource+
					" is not in allowedDestructive (LH-SEC-005)")
		}
	}

	for _, msg := range failures {
		t.Error(msg)
	}
}
