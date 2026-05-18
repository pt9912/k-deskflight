/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package domain

import (
	"context"
	"fmt"
	"strings"
)

// ClusterResourcesSpecKind ist der stabile Spec-Token für den
// ClusterResources-Check (architecture.md AR-012, LH-F-015).
const ClusterResourcesSpecKind = "clusterResources"

// Profile-Defaults für den ClusterResources-Check (slice-M4 §2.3).
// Werte sind k8s-`resource.Quantity`-Strings; das Quantity-Parsing
// findet im Adapter statt (Domain darf k8s nicht importieren —
// AR-005 depguard `domain-isolation`).
const (
	// DefaultClusterResourcesMinCPUProduction ist der Production-
	// Default für die geforderte Mindest-CPU (4 vCPU).
	DefaultClusterResourcesMinCPUProduction = "4"

	// DefaultClusterResourcesMinMemoryProduction ist der Production-
	// Default für den geforderten Mindest-RAM (8 GiB).
	DefaultClusterResourcesMinMemoryProduction = "8Gi"

	// DefaultClusterResourcesMinCPUEvaluation ist der Evaluation-
	// Default für die geforderte Mindest-CPU (2 vCPU).
	DefaultClusterResourcesMinCPUEvaluation = "2"

	// DefaultClusterResourcesMinMemoryEvaluation ist der Evaluation-
	// Default für den geforderten Mindest-RAM (4 GiB).
	DefaultClusterResourcesMinMemoryEvaluation = "4Gi"
)

// ClusterResourcesSpec parametriert den ClusterResources-Check.
//
// `MinCPU` und `MinMemory` sind Kubernetes-`resource.Quantity`-Strings
// (z. B. `"4"`, `"500m"`, `"8Gi"`, `"2048Mi"`). Die echte Parse- und
// Vergleichslogik liegt im Adapter; Validate prüft nur, dass die
// Felder nicht-leer sind.
type ClusterResourcesSpec struct {
	MinCPU    string
	MinMemory string
}

// Kind erfüllt das CheckSpec-Interface.
func (s ClusterResourcesSpec) Kind() string { return ClusterResourcesSpecKind }

// Validate prüft Syntax-Plausibilität: beide Felder müssen nicht-leer
// sein. Die Quantity-Format-Validierung übernimmt der CRD-OpenAPI-
// Pattern und der Adapter-Check via `resource.ParseQuantity`.
func (s ClusterResourcesSpec) Validate(_ context.Context) error {
	if strings.TrimSpace(s.MinCPU) == "" {
		return fmt.Errorf("clusterResources.minCPU: empty")
	}
	if strings.TrimSpace(s.MinMemory) == "" {
		return fmt.Errorf("clusterResources.minMemory: empty")
	}
	return nil
}
