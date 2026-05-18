/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

// Package markers extrahiert `+kubebuilder:rbac:`-Marker aus Go-Quelldateien
// (architecture.md AR-007, slice-M5 §2.2 / §2.7). Wird ausschließlich von
// Test-/Audit-Pfaden genutzt — der Parser läuft via `go/parser` über
// Comments, ohne die Quelle zu laden oder zu kompilieren.
//
// Zwei Audit-Tests teilen sich dieses Mini-Package:
//
//   - `internal/hexagon/application/destructive_audit_test.go` —
//     verifiziert `LH-SEC-005` (keine destruktiven Verben auf fremde
//     Ressourcen).
//   - `internal/adapter/check/rbac_consistency_test.go` — verifiziert
//     `Check.RequiredPermissions()` vs. Reconciler-Marker.
//
// Die Aufteilung in ein eigenes Package umgeht die `depguard`-
// Layer-Trennung: `application-no-adapter` und `adapter-no-application`
// verbieten den Tests, sich gegenseitig zu importieren; `markers` ist
// neutraler Audit-Code und damit für beide Tests zugänglich.
package markers

import (
	"fmt"
	"go/parser"
	"go/token"
	"strings"
)

// Marker repräsentiert einen einzelnen `+kubebuilder:rbac:`-Eintrag aus
// einer Quelldatei. Resources können `resource/subresource`-Form
// tragen (z. B. `opendeskpreflightchecks/status`).
type Marker struct {
	Groups    []string
	Resources []string
	Verbs     []string
	Position  string // file:line
}

// ParseRBACFile liest die Quelldatei unter `path`, parst alle Comments
// und liefert pro `+kubebuilder:rbac:`-Marker einen `Marker`-Eintrag.
// Nicht-Marker-Comments werden ignoriert.
func ParseRBACFile(path string) ([]Marker, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var markers []Marker
	for _, group := range f.Comments {
		for _, c := range group.List {
			text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			if !strings.HasPrefix(text, "+kubebuilder:rbac:") {
				continue
			}
			payload := strings.TrimPrefix(text, "+kubebuilder:rbac:")
			m := parseRBACLine(payload)
			m.Position = fset.Position(c.Pos()).String()
			markers = append(markers, m)
		}
	}
	return markers, nil
}

// parseRBACLine teilt eine Marker-Payload wie
// `groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch`
// in die drei Felder auf. `;` trennt Multi-Values, `""` umschließt
// einen leeren Group-String (für die Core-API).
func parseRBACLine(payload string) Marker {
	var m Marker
	for _, kv := range strings.Split(payload, ",") {
		idx := strings.IndexByte(kv, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(kv[:idx])
		val := strings.Trim(strings.TrimSpace(kv[idx+1:]), `"`)
		switch key {
		case "groups":
			m.Groups = splitSemicolon(val)
		case "resources":
			m.Resources = splitSemicolon(val)
		case "verbs":
			m.Verbs = splitSemicolon(val)
		}
	}
	return m
}

func splitSemicolon(s string) []string {
	if s == "" {
		return []string{""}
	}
	return strings.Split(s, ";")
}

// Triple ist die kleinste verbalisierte Permission-Einheit für
// Konsistenz-Tests: (Group, Resource[/Subresource], Verb). Die String-
// Repräsentation ist stabil, damit sie als Map-Key dient.
type Triple struct {
	Group    string
	Resource string // kann "resource/subresource" enthalten
	Verb     string
}

// String liefert eine kanonische Darstellung für Log/Set-Vergleiche.
func (t Triple) String() string {
	g := t.Group
	if g == "" {
		g = "core"
	}
	return fmt.Sprintf("%s %s/%s", t.Verb, g, t.Resource)
}

// Expand entfaltet einen Marker in die Cartesian-Triples
// `(Group × Resource × Verb)`.
func (m Marker) Expand() []Triple {
	out := make([]Triple, 0, len(m.Groups)*len(m.Resources)*len(m.Verbs))
	for _, g := range m.Groups {
		for _, r := range m.Resources {
			for _, v := range m.Verbs {
				out = append(out, Triple{Group: g, Resource: r, Verb: v})
			}
		}
	}
	return out
}
