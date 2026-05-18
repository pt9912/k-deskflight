/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package port

import "context"

// APIGroupDiscovery prüft die Registrierung einer API-Gruppe am
// Kubernetes-API-Server (architecture.md AR-009, slice-M4 §2.1).
//
// Genutzt vom cert-manager-Existence-Check (`LH-F-013`): Erfolg
// bedeutet, dass `cert-manager.io`-CRDs im Cluster bekannt sind —
// nicht, dass cert-manager-Pods laufen oder lauffähig sind.
type APIGroupDiscovery interface {
	HasAPIGroup(ctx context.Context, name string) (bool, error)
}
