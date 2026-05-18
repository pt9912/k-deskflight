/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package domain

import "context"

// CertManagerSpecKind ist der stabile Spec-Token für den
// cert-manager-Existence-Check (architecture.md AR-012, LH-F-013).
const CertManagerSpecKind = "certManager"

// CertManagerAPIGroup ist die API-Gruppe, deren Vorhandensein der
// cert-manager-Check prüft. ClusterIssuer-Detailprüfung (`LH-F-014`)
// bleibt v0.2 (`ADR 0010`).
const CertManagerAPIGroup = "cert-manager.io"

// CertManagerSpec ist parameterlos — der Check prüft ausschließlich,
// ob die cert-manager-API-Gruppe im Cluster registriert ist.
//
// Default-Severity bei fehlendem cert-manager ist `warning`
// (slice-M4 §9): OpenDesk ist auch ohne cert-manager deploybar.
type CertManagerSpec struct{}

// Kind erfüllt das CheckSpec-Interface.
func (s CertManagerSpec) Kind() string { return CertManagerSpecKind }

// Validate ist immer erfolgreich — die Spec trägt keine Parameter.
func (s CertManagerSpec) Validate(_ context.Context) error { return nil }
