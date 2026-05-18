/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package domain

import "strings"

// PermissionRequest beschreibt ein einzelnes Recht, das ein Check vor
// seiner Ausführung gegen den Cluster prüfen lässt (architecture.md
// AR-018, slice-M5 §2.2). Der Adapter unter
// `internal/adapter/k8s/access_review.go` mapped diese Struktur auf
// `authorizationv1.SelfSubjectAccessReview.Spec.ResourceAttributes`,
// damit der Domain-Layer k8s-Imports vermeidet (AR-005 `domain-isolation`).
type PermissionRequest struct {
	// Group ist die API-Gruppe; leer entspricht der Core-API.
	Group string

	// Resource ist der Plural-Resource-Name (z. B. "storageclasses").
	Resource string

	// Subresource ist optional (z. B. "status").
	Subresource string

	// Verb ist die HTTP-/CRUD-Operation (z. B. "list", "get", "create").
	Verb string

	// Namespace bleibt leer für cluster-scoped Resourcen.
	Namespace string
}

// CanonicalString liefert eine deterministische String-Repräsentation
// für Logging, Cache-Keys und Test-Vergleiche. Format:
// `{verb} {group|core}/{resource}[/{subresource}][@{namespace}]`.
func (p PermissionRequest) CanonicalString() string {
	var b strings.Builder
	b.WriteString(p.Verb)
	b.WriteByte(' ')
	if p.Group == "" {
		b.WriteString("core")
	} else {
		b.WriteString(p.Group)
	}
	b.WriteByte('/')
	b.WriteString(p.Resource)
	if p.Subresource != "" {
		b.WriteByte('/')
		b.WriteString(p.Subresource)
	}
	if p.Namespace != "" {
		b.WriteByte('@')
		b.WriteString(p.Namespace)
	}
	return b.String()
}
