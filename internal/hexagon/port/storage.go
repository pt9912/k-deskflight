/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package port

import "context"

// StorageClassInfo ist die port-seitige Repräsentation einer
// Cluster-StorageClass (architecture.md AR-009, slice-M4 §2.1).
// `Provisioner` wird mitgeliefert, damit künftige Checks (`LH-F-014`+)
// Filter darauf legen können — der MVP-Check (`LH-F-010`/`LH-F-011`)
// nutzt nur `Name` und `IsDefault`.
type StorageClassInfo struct {
	Name        string
	IsDefault   bool
	Provisioner string
}

// StorageClassDiscovery liefert die im Cluster installierten
// StorageClasses (`LH-F-010`/`LH-F-011`). Implementiert vom
// `internal/adapter/k8s/storage.go`-Adapter.
type StorageClassDiscovery interface {
	ListStorageClasses(ctx context.Context) ([]StorageClassInfo, error)
}
