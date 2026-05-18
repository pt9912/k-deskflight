/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package port

import "context"

// IngressClassInfo ist die port-seitige Repräsentation einer
// Cluster-IngressClass (architecture.md AR-009, slice-M4 §2.1).
// `Controller` (z. B. `k8s.io/ingress-nginx`) wird mitgeliefert für
// künftige Detail-Filter; der MVP-Check (`LH-F-012`) nutzt nur `Name`.
type IngressClassInfo struct {
	Name       string
	Controller string
}

// IngressClassDiscovery liefert die im Cluster installierten
// IngressClasses (`LH-F-012`). Implementiert vom
// `internal/adapter/k8s/ingress.go`-Adapter.
type IngressClassDiscovery interface {
	ListIngressClasses(ctx context.Context) ([]IngressClassInfo, error)
}
