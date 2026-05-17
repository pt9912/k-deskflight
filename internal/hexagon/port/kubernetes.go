/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

// Package port deklariert die Abhängigkeits-Inversions-Interfaces des
// Operators (architecture.md AR-004). Application konsumiert; Adapter
// implementiert.
package port

import "context"

// KubernetesAPI bündelt die Discovery-/Inspect-Operationen, die der
// Operator gegen den Ziel-Cluster benötigt (architecture.md AR-009).
// In M3 ist nur `ServerVersion` belegt — M4 ergänzt StorageClass-/
// IngressClass-Discovery, M5 SelfSubjectAccessReview.
type KubernetesAPI interface {
	// ServerVersion liefert die rohe Version-String-Ausgabe von
	// `/version` (z. B. "v1.34.2", "1.34.2+abc"). Der Adapter normalisiert
	// nichts — semantische Auswertung passiert in der Check-Implementierung.
	ServerVersion(ctx context.Context) (string, error)
}
