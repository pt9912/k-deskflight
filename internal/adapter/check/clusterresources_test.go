/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package check_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pt9912/k-deskflight/internal/adapter/check"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

type fakeNodeDiscovery struct {
	nodes []port.NodeInfo
	err   error
}

func (f fakeNodeDiscovery) ListReadyNodes(_ context.Context) ([]port.NodeInfo, error) {
	return f.nodes, f.err
}

const (
	oneCPU      int64 = 1000             // 1000 millicpu
	oneGiBytes  int64 = 1024 * 1024 * 1024
)

func TestClusterResourcesPassedSingleNode(t *testing.T) {
	t.Parallel()
	disc := fakeNodeDiscovery{nodes: []port.NodeInfo{
		{Name: "node-1", AllocatableCPUMilli: 4 * oneCPU, AllocatableMemoryBytes: 8 * oneGiBytes},
	}}
	chk := check.NewClusterResources(disc, fixedClock())

	res := chk.Run(context.Background(), domain.ClusterResourcesSpec{MinCPU: "4", MinMemory: "8Gi"})

	if res.Status != domain.StatusTrue {
		t.Errorf("Status: got %q, want %q (message=%q)", res.Status, domain.StatusTrue, res.Message)
	}
	if res.Reason != "ResourcesSufficient" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "ResourcesSufficient")
	}
}

func TestClusterResourcesPassedSumAcrossNodes(t *testing.T) {
	t.Parallel()
	disc := fakeNodeDiscovery{nodes: []port.NodeInfo{
		{Name: "node-1", AllocatableCPUMilli: 2 * oneCPU, AllocatableMemoryBytes: 4 * oneGiBytes},
		{Name: "node-2", AllocatableCPUMilli: 2 * oneCPU, AllocatableMemoryBytes: 4 * oneGiBytes},
	}}
	chk := check.NewClusterResources(disc, fixedClock())

	res := chk.Run(context.Background(), domain.ClusterResourcesSpec{MinCPU: "4", MinMemory: "8Gi"})

	if res.Status != domain.StatusTrue {
		t.Errorf("Status: got %q, want %q (message=%q)", res.Status, domain.StatusTrue, res.Message)
	}
}

func TestClusterResourcesFailedCPUOnly(t *testing.T) {
	t.Parallel()
	disc := fakeNodeDiscovery{nodes: []port.NodeInfo{
		{Name: "node-1", AllocatableCPUMilli: 1 * oneCPU, AllocatableMemoryBytes: 16 * oneGiBytes},
	}}
	chk := check.NewClusterResources(disc, fixedClock())

	res := chk.Run(context.Background(), domain.ClusterResourcesSpec{MinCPU: "4", MinMemory: "8Gi"})

	if res.Status != domain.StatusFalse {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusFalse)
	}
	if res.Reason != "InsufficientCPU" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "InsufficientCPU")
	}
	if res.Severity != domain.SeverityCritical {
		t.Errorf("Severity: got %q, want %q", res.Severity, domain.SeverityCritical)
	}
}

func TestClusterResourcesFailedMemoryOnly(t *testing.T) {
	t.Parallel()
	disc := fakeNodeDiscovery{nodes: []port.NodeInfo{
		{Name: "node-1", AllocatableCPUMilli: 8 * oneCPU, AllocatableMemoryBytes: 2 * oneGiBytes},
	}}
	chk := check.NewClusterResources(disc, fixedClock())

	res := chk.Run(context.Background(), domain.ClusterResourcesSpec{MinCPU: "4", MinMemory: "8Gi"})

	if res.Status != domain.StatusFalse {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusFalse)
	}
	if res.Reason != "InsufficientMemory" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "InsufficientMemory")
	}
}

func TestClusterResourcesFailedBoth(t *testing.T) {
	t.Parallel()
	disc := fakeNodeDiscovery{nodes: []port.NodeInfo{
		{Name: "node-1", AllocatableCPUMilli: 1 * oneCPU, AllocatableMemoryBytes: 1 * oneGiBytes},
	}}
	chk := check.NewClusterResources(disc, fixedClock())

	res := chk.Run(context.Background(), domain.ClusterResourcesSpec{MinCPU: "4", MinMemory: "8Gi"})

	if res.Status != domain.StatusFalse {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusFalse)
	}
	if res.Reason != "InsufficientResources" {
		t.Errorf("Reason: got %q, want %q (both-short → generic reason)", res.Reason, "InsufficientResources")
	}
	if !strings.Contains(res.Message, "cpu:") || !strings.Contains(res.Message, "memory:") {
		t.Errorf("Message must diagnose both; got %q", res.Message)
	}
}

func TestClusterResourcesLookupFailed(t *testing.T) {
	t.Parallel()
	disc := fakeNodeDiscovery{err: errors.New("forbidden")}
	chk := check.NewClusterResources(disc, fixedClock())

	res := chk.Run(context.Background(), domain.ClusterResourcesSpec{MinCPU: "4", MinMemory: "8Gi"})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "ClusterResourcesLookupFailed" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "ClusterResourcesLookupFailed")
	}
}

func TestClusterResourcesInvalidMinCPUSpec(t *testing.T) {
	t.Parallel()
	disc := fakeNodeDiscovery{}
	chk := check.NewClusterResources(disc, fixedClock())

	res := chk.Run(context.Background(), domain.ClusterResourcesSpec{MinCPU: "not-a-quantity", MinMemory: "8Gi"})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "InvalidSpec" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "InvalidSpec")
	}
}

func TestClusterResourcesInvalidMinMemorySpec(t *testing.T) {
	t.Parallel()
	disc := fakeNodeDiscovery{}
	chk := check.NewClusterResources(disc, fixedClock())

	res := chk.Run(context.Background(), domain.ClusterResourcesSpec{MinCPU: "4", MinMemory: "not-a-quantity"})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "InvalidSpec" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "InvalidSpec")
	}
}

func TestClusterResourcesInvalidSpecType(t *testing.T) {
	t.Parallel()
	disc := fakeNodeDiscovery{}
	chk := check.NewClusterResources(disc, fixedClock())

	res := chk.Run(context.Background(), otherSpec{})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "InvalidSpec" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "InvalidSpec")
	}
}
