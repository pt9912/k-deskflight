/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package port

import "context"

// NodeInfo ist die port-seitige Repräsentation eines Cluster-Nodes
// (architecture.md AR-009, slice-M4 §2.1).
//
// `Allocatable*`-Werte werden bereits adapterseitig aus
// `resource.Quantity` in skalare Einheiten umgerechnet — Domain darf
// `resource.Quantity` nicht importieren (AR-005). Spec-Quantities
// werden im Adapter-Check ebenfalls in dieselben Einheiten geparst
// und dann skalar verglichen.
type NodeInfo struct {
	Name                   string
	AllocatableCPUMilli    int64
	AllocatableMemoryBytes int64
}

// NodeDiscovery liefert die `Ready`-Nodes des Clusters samt ihrer
// Allocatable-Werte (`LH-F-015`). Implementiert vom
// `internal/adapter/k8s/nodes.go`-Adapter, der Nodes ohne
// `NodeReady=True` ausfiltert.
type NodeDiscovery interface {
	ListReadyNodes(ctx context.Context) ([]NodeInfo, error)
}
