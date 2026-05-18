/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package domain_test

import (
	"context"
	"testing"

	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

func TestClusterResourcesSpecKind(t *testing.T) {
	spec := domain.ClusterResourcesSpec{MinCPU: "1", MinMemory: "1Gi"}
	if got, want := spec.Kind(), "clusterResources"; got != want {
		t.Errorf("Kind(): got %q, want %q", got, want)
	}
}

func TestClusterResourcesSpecValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		spec    domain.ClusterResourcesSpec
		wantErr bool
	}{
		{"both set, integer", domain.ClusterResourcesSpec{MinCPU: "4", MinMemory: "8Gi"}, false},
		{"both set, milli + Mi", domain.ClusterResourcesSpec{MinCPU: "500m", MinMemory: "512Mi"}, false},
		{"production defaults", domain.ClusterResourcesSpec{
			MinCPU:    domain.DefaultClusterResourcesMinCPUProduction,
			MinMemory: domain.DefaultClusterResourcesMinMemoryProduction,
		}, false},
		{"evaluation defaults", domain.ClusterResourcesSpec{
			MinCPU:    domain.DefaultClusterResourcesMinCPUEvaluation,
			MinMemory: domain.DefaultClusterResourcesMinMemoryEvaluation,
		}, false},

		{"empty cpu", domain.ClusterResourcesSpec{MinCPU: "", MinMemory: "1Gi"}, true},
		{"empty memory", domain.ClusterResourcesSpec{MinCPU: "1", MinMemory: ""}, true},
		{"both empty", domain.ClusterResourcesSpec{}, true},
		{"whitespace cpu", domain.ClusterResourcesSpec{MinCPU: "   ", MinMemory: "1Gi"}, true},
		{"whitespace memory", domain.ClusterResourcesSpec{MinCPU: "1", MinMemory: "\t"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.spec.Validate(context.Background())
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate(%+v): err = %v, wantErr = %v", tc.spec, err, tc.wantErr)
			}
		})
	}
}

func TestClusterResourcesProfileDefaults(t *testing.T) {
	t.Parallel()

	cases := []struct {
		constant string
		want     string
	}{
		{domain.DefaultClusterResourcesMinCPUProduction, "4"},
		{domain.DefaultClusterResourcesMinMemoryProduction, "8Gi"},
		{domain.DefaultClusterResourcesMinCPUEvaluation, "2"},
		{domain.DefaultClusterResourcesMinMemoryEvaluation, "4Gi"},
	}

	for _, tc := range cases {
		if tc.constant != tc.want {
			t.Errorf("default constant: got %q, want %q", tc.constant, tc.want)
		}
	}
}
