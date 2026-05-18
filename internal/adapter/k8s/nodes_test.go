/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package k8s_test

import (
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/pt9912/k-deskflight/internal/adapter/k8s"
)

func makeNode(name string, ready corev1.ConditionStatus, cpu, memory string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(memory),
			},
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: ready}},
		},
	}
}

func TestNodeAdapterListReadyNodesFiltersAndConverts(t *testing.T) {
	cs := fake.NewSimpleClientset(
		makeNode("ready-1", corev1.ConditionTrue, "4", "8Gi"),
		makeNode("ready-2", corev1.ConditionTrue, "2", "4Gi"),
		makeNode("not-ready", corev1.ConditionFalse, "8", "16Gi"),
	)

	adapter := k8s.NewNodeAdapter(cs)
	nodes, err := adapter.ListReadyNodes(context.Background())
	if err != nil {
		t.Fatalf("ListReadyNodes: %v", err)
	}

	if got, want := len(nodes), 2; got != want {
		t.Fatalf("count: got %d, want %d (not-ready node must be filtered)", got, want)
	}

	byName := map[string]struct {
		cpuMilli int64
		memBytes int64
	}{}
	for _, n := range nodes {
		byName[n.Name] = struct {
			cpuMilli int64
			memBytes int64
		}{n.AllocatableCPUMilli, n.AllocatableMemoryBytes}
	}

	if byName["ready-1"].cpuMilli != 4000 {
		t.Errorf("ready-1 cpu: got %d, want 4000", byName["ready-1"].cpuMilli)
	}
	if byName["ready-1"].memBytes != int64(8*1024*1024*1024) {
		t.Errorf("ready-1 memory: got %d, want %d", byName["ready-1"].memBytes, int64(8*1024*1024*1024))
	}
	if _, present := byName["not-ready"]; present {
		t.Errorf("not-ready node must be excluded")
	}
}

func TestNodeAdapterListReadyNodesUnknownCondition(t *testing.T) {
	// Node ohne NodeReady-Condition wird wie not-ready behandelt.
	cs := fake.NewSimpleClientset(&corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "no-condition"},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		},
	})

	adapter := k8s.NewNodeAdapter(cs)
	nodes, err := adapter.ListReadyNodes(context.Background())
	if err != nil {
		t.Fatalf("ListReadyNodes: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("count: got %d, want 0 (node without NodeReady condition must be filtered)", len(nodes))
	}
}

func TestNodeAdapterListReadyNodesError(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("list", "nodes", func(_ ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("forbidden")
	})

	adapter := k8s.NewNodeAdapter(cs)
	if _, err := adapter.ListReadyNodes(context.Background()); err == nil {
		t.Errorf("expected error, got nil")
	}
}
