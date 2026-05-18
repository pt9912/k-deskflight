/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package check

import (
	"context"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

const (
	// CheckNameClusterResources ist der stabile Identifier (architecture.md AR-012, LH-F-015).
	CheckNameClusterResources = "clusterResources"

	// ConditionTypeClusterResourcesReady ist der Condition-Type im CR-Status.
	ConditionTypeClusterResourcesReady = "ClusterResourcesReady"

	reasonResourcesSufficient        = "ResourcesSufficient"
	reasonInsufficientCPU            = "InsufficientCPU"
	reasonInsufficientMemory         = "InsufficientMemory"
	reasonInsufficientResources      = "InsufficientResources"
	reasonClusterResourcesInvalidSpec = "InvalidSpec"
	reasonClusterResourcesLookup     = "ClusterResourcesLookupFailed"
)

// ClusterResources prüft, ob die Summe der Allocatable-Werte aller
// Ready-Nodes die konfigurierten Mindestmengen erreicht
// (`LH-F-015`/`LH-AK-009`). Spec-Quantities werden hier (Adapter-Layer)
// per `resource.ParseQuantity` ausgewertet; der Domain-/Port-Layer
// kennt nur skalare Einheiten.
type ClusterResources struct {
	disc port.NodeDiscovery
	now  func() time.Time
}

// NewClusterResources baut den Check mit einer Node-Discovery-Quelle.
func NewClusterResources(disc port.NodeDiscovery, now func() time.Time) *ClusterResources {
	if now == nil {
		now = time.Now
	}
	return &ClusterResources{disc: disc, now: now}
}

// Name erfüllt das Check-Interface.
func (c *ClusterResources) Name() string { return CheckNameClusterResources }

// SpecKind erfüllt das Check-Interface.
func (c *ClusterResources) SpecKind() string { return domain.ClusterResourcesSpecKind }

// RequiredPermissions deklariert das `list`-Recht auf Core-`nodes`
// (slice-M5 §2.2). Group ist leer für die Core-API-Gruppe.
func (c *ClusterResources) RequiredPermissions() []domain.PermissionRequest {
	return []domain.PermissionRequest{
		{Group: "", Resource: "nodes", Verb: "list"},
	}
}

// Run summiert Allocatable-CPU/Memory über alle Ready-Nodes und
// vergleicht gegen die konfigurierten Mindestmengen.
func (c *ClusterResources) Run(ctx context.Context, spec domain.CheckSpec) domain.Result {
	s, ok := spec.(domain.ClusterResourcesSpec)
	if !ok || spec.Kind() != c.SpecKind() {
		return c.unknown(reasonClusterResourcesInvalidSpec,
			fmt.Sprintf("expected spec kind %q, got %q", c.SpecKind(), spec.Kind()))
	}

	minCPU, err := resource.ParseQuantity(s.MinCPU)
	if err != nil {
		return c.unknown(reasonClusterResourcesInvalidSpec,
			fmt.Sprintf("clusterResources.minCPU %q: %v", s.MinCPU, err))
	}
	minMem, err := resource.ParseQuantity(s.MinMemory)
	if err != nil {
		return c.unknown(reasonClusterResourcesInvalidSpec,
			fmt.Sprintf("clusterResources.minMemory %q: %v", s.MinMemory, err))
	}

	nodes, err := c.disc.ListReadyNodes(ctx)
	if err != nil {
		return c.unknown(reasonClusterResourcesLookup,
			fmt.Sprintf("node lookup failed: %v", err))
	}

	var totalCPUMilli, totalMemBytes int64
	for _, n := range nodes {
		totalCPUMilli += n.AllocatableCPUMilli
		totalMemBytes += n.AllocatableMemoryBytes
	}

	minCPUMilli := minCPU.MilliValue()
	minMemBytes := minMem.Value()

	cpuShort := totalCPUMilli < minCPUMilli
	memShort := totalMemBytes < minMemBytes

	if cpuShort || memShort {
		var diagnostics []string
		if cpuShort {
			diagnostics = append(diagnostics,
				fmt.Sprintf("cpu: %dm allocatable, need %dm", totalCPUMilli, minCPUMilli))
		}
		if memShort {
			diagnostics = append(diagnostics,
				fmt.Sprintf("memory: %d bytes allocatable, need %d bytes", totalMemBytes, minMemBytes))
		}

		reason := reasonInsufficientResources
		switch {
		case cpuShort && !memShort:
			reason = reasonInsufficientCPU
		case memShort && !cpuShort:
			reason = reasonInsufficientMemory
		}
		return c.failed(reason, strings.Join(diagnostics, "; "))
	}

	return c.passed(reasonResourcesSufficient,
		fmt.Sprintf("allocatable %dm CPU / %d bytes memory across %d Ready node(s) satisfies %dm / %d minimum",
			totalCPUMilli, totalMemBytes, len(nodes), minCPUMilli, minMemBytes))
}

func (c *ClusterResources) passed(reason, message string) domain.Result {
	return domain.Result{
		Name:           ConditionTypeClusterResourcesReady,
		Status:         domain.StatusTrue,
		Reason:         reason,
		Severity:       domain.SeverityInfo,
		Message:        message,
		LastTransition: c.now().UTC(),
	}
}

func (c *ClusterResources) failed(reason, message string) domain.Result {
	return domain.Result{
		Name:           ConditionTypeClusterResourcesReady,
		Status:         domain.StatusFalse,
		Reason:         reason,
		Severity:       domain.SeverityCritical,
		Message:        message,
		LastTransition: c.now().UTC(),
	}
}

func (c *ClusterResources) unknown(reason, message string) domain.Result {
	return domain.Result{
		Name:           ConditionTypeClusterResourcesReady,
		Status:         domain.StatusUnknown,
		Reason:         reason,
		Severity:       domain.SeverityCritical,
		Message:        message,
		LastTransition: c.now().UTC(),
	}
}
