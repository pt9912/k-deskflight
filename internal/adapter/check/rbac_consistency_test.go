/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package check_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/pt9912/k-deskflight/internal/adapter/check"
	"github.com/pt9912/k-deskflight/internal/audit/markers"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

// allMVPChecks ist die Pflicht-Wartungs-Tabelle der MVP-Checks
// (slice-M5 §2.2). Wenn ein neuer Check hinzukommt, MUSS er hier
// eingetragen werden; sonst verifiziert der Konsistenz-Test seine
// `RequiredPermissions()` nicht.
func allMVPChecks() []domain.Check {
	return []domain.Check{
		check.NewKubernetesVersion(nil, nil),
		check.NewStorageClass(nil, nil),
		check.NewIngressClass(nil, nil),
		check.NewCertManager(nil, nil),
		check.NewClusterResources(nil, nil),
	}
}

// operativeMarkersWhitelist liefert die Marker-Triples, die nicht von
// einer `Check.RequiredPermissions()` beansprucht werden, sondern den
// laufenden Operator-Pfad versorgen. Begründung pro Eintrag.
//
// **Erweiterungs-Pflicht:** Wenn ein neuer Marker hinzukommt, der von
// keiner Check-Permission gedeckt wird, MUSS er hier eingetragen
// werden — sonst bricht der Konsistenz-Test.
func operativeMarkersWhitelist() map[string]bool {
	return map[string]bool{
		// controller-runtime liest/watcht den eigenen CR (LH-F-003).
		"get k-deskflight.geo-terrain.net/opendeskpreflightchecks":   true,
		"list k-deskflight.geo-terrain.net/opendeskpreflightchecks":  true,
		"watch k-deskflight.geo-terrain.net/opendeskpreflightchecks": true,

		// Reconciler schreibt Status (LH-F-004, AR-009 §6).
		"get k-deskflight.geo-terrain.net/opendeskpreflightchecks/status":    true,
		"update k-deskflight.geo-terrain.net/opendeskpreflightchecks/status": true,
		"patch k-deskflight.geo-terrain.net/opendeskpreflightchecks/status":  true,

		// SAR-Adapter ruft selfsubjectaccessreviews create (AR-018).
		"create authorization.k8s.io/selfsubjectaccessreviews": true,
		// Pre-Grant für SelfSubjectRulesReview (v0.2 LH-F-024-Erweiterung).
		"create authorization.k8s.io/selfsubjectrulesreviews": true,

		// Leader-Election-Leases (AR-026, M7-Aktivierung; M2-Pre-Grant).
		"get coordination.k8s.io/leases":    true,
		"list coordination.k8s.io/leases":   true,
		"watch coordination.k8s.io/leases":  true,
		"create coordination.k8s.io/leases": true,
		"update coordination.k8s.io/leases": true,
		"patch coordination.k8s.io/leases":  true,

		// CRD-Read als M2-Pre-Grant (slice-M4 §3.5). Aktuell ungenutzt —
		// cert-manager-Check läuft über discovery.ServerGroups, nicht
		// über direkten CRD-Read. Whitelist als „future-use".
		"get apiextensions.k8s.io/customresourcedefinitions":  true,
		"list apiextensions.k8s.io/customresourcedefinitions": true,

		// Pre-Grant für Storage/Ingress/Nodes-Watch (controller-runtime
		// erwartet watch-Verben, sobald wir Watches auf diese Resources
		// einrichten). list-Verben sind durch RequiredPermissions
		// abgedeckt.
		"get core/nodes":                         true,
		"watch core/nodes":                       true,
		"get networking.k8s.io/ingressclasses":   true,
		"watch networking.k8s.io/ingressclasses": true,
		"get storage.k8s.io/storageclasses":      true,
		"watch storage.k8s.io/storageclasses":    true,
	}
}

const reconcilerPath = "../../hexagon/application/reconciler.go"

// TestRBACConsistencyChecksMappedToMarkers (slice-M5 §2.2, Review-
// Befund 4): jeder Check, der RequiredPermissions deklariert, muss
// einen passenden `+kubebuilder:rbac:`-Marker im Reconciler-File
// haben — sonst würde die Smoke-Laufzeit RBACInsufficient melden.
func TestRBACConsistencyChecksMappedToMarkers(t *testing.T) {
	parsed, err := markers.ParseRBACFile(reconcilerPath)
	if err != nil {
		t.Fatalf("ParseRBACFile: %v", err)
	}

	markerSet := make(map[string]bool)
	for _, m := range parsed {
		for _, tr := range m.Expand() {
			markerSet[tr.String()] = true
		}
	}

	var failures []string
	for _, c := range allMVPChecks() {
		for _, perm := range c.RequiredPermissions() {
			key := triplStringFromPermission(perm)
			if !markerSet[key] {
				failures = append(failures, c.Name()+" declares "+key+
					" but reconciler has no matching +kubebuilder:rbac: marker")
			}
		}
	}

	sort.Strings(failures)
	for _, f := range failures {
		t.Error(f)
	}
}

// TestRBACConsistencyMarkersBackedByChecks (slice-M5 §2.2, Review-
// Befund 4): jeder Marker im Reconciler-File muss entweder von einem
// Check beansprucht werden ODER explizit als operativ gewhitelistet
// sein. Drift in dieser Richtung deutet auf Reste alter Slices oder
// nicht eingetragene neue Checks hin.
func TestRBACConsistencyMarkersBackedByChecks(t *testing.T) {
	parsed, err := markers.ParseRBACFile(reconcilerPath)
	if err != nil {
		t.Fatalf("ParseRBACFile: %v", err)
	}

	checkSet := make(map[string]bool)
	for _, c := range allMVPChecks() {
		for _, perm := range c.RequiredPermissions() {
			checkSet[triplStringFromPermission(perm)] = true
		}
	}

	operative := operativeMarkersWhitelist()
	var failures []string
	for _, m := range parsed {
		for _, tr := range m.Expand() {
			key := tr.String()
			if checkSet[key] {
				continue
			}
			if operative[key] {
				continue
			}
			failures = append(failures,
				m.Position+": marker "+key+" is not claimed by any Check.RequiredPermissions() "+
					"and not whitelisted as operative")
		}
	}

	sort.Strings(failures)
	for _, f := range failures {
		t.Error(f)
	}
}

// triplStringFromPermission baut den `Triple.String()`-Schlüssel aus
// einer `domain.PermissionRequest` (Group="" → "core", Subresource
// hinten an Resource mit "/").
func triplStringFromPermission(p domain.PermissionRequest) string {
	g := p.Group
	if g == "" {
		g = "core"
	}
	res := p.Resource
	if p.Subresource != "" {
		res = res + "/" + p.Subresource
	}
	var b strings.Builder
	b.WriteString(p.Verb)
	b.WriteByte(' ')
	b.WriteString(g)
	b.WriteByte('/')
	b.WriteString(res)
	return b.String()
}
