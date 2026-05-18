/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package k8s

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

// NodeAdapter implementiert `port.NodeDiscovery` gegen die Core-API.
// Filtert Nodes ohne `NodeReady=True` aus und konvertiert Allocatable
// in skalare Einheiten (millicpu, bytes), damit der Domain-/Port-
// Layer k8s `resource.Quantity` nicht kennen muss (AR-005). M4-untested.
type NodeAdapter struct {
	client kubernetes.Interface
}

// NewNodeAdapter baut den Adapter aus einem geteilten Clientset.
func NewNodeAdapter(c kubernetes.Interface) *NodeAdapter {
	return &NodeAdapter{client: c}
}

// ListReadyNodes liefert nur die Nodes mit `NodeReady=True`. Werte
// `AllocatableCPUMilli`/`AllocatableMemoryBytes` stammen aus
// `Status.Allocatable[cpu|memory].MilliValue()`/`.Value()`.
func (a *NodeAdapter) ListReadyNodes(ctx context.Context) ([]port.NodeInfo, error) {
	list, err := a.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	out := make([]port.NodeInfo, 0, len(list.Items))
	for _, n := range list.Items {
		if !isNodeReady(n.Status.Conditions) {
			continue
		}
		out = append(out, port.NodeInfo{
			Name:                   n.Name,
			AllocatableCPUMilli:    n.Status.Allocatable.Cpu().MilliValue(),
			AllocatableMemoryBytes: n.Status.Allocatable.Memory().Value(),
		})
	}
	return out, nil
}

func isNodeReady(conditions []corev1.NodeCondition) bool {
	for _, c := range conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
