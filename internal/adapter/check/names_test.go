/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package check_test

import (
	"testing"

	"github.com/pt9912/k-deskflight/internal/adapter/check"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

// TestCheckNamesAndDefaultClock deckt die Name()-Methoden aller
// Adapter-Checks plus den `now=nil`-Fallback in den jeweiligen
// New*-Konstruktoren (defaults auf time.Now).
func TestCheckNamesAndDefaultClock(t *testing.T) {
	t.Parallel()

	cases := []struct {
		want string
		got  string
	}{
		{"kubernetesVersion", check.NewKubernetesVersion(nil, nil).Name()},
		{"storageClass", check.NewStorageClass(nil, nil).Name()},
		{"ingressClass", check.NewIngressClass(nil, nil).Name()},
		{"certManager", check.NewCertManager(nil, nil).Name()},
		{"clusterResources", check.NewClusterResources(nil, nil).Name()},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("Name: got %q, want %q", tc.got, tc.want)
		}
	}

	// Konsistenz-Check: SpecKind matched Domain-Konstante.
	if check.NewStorageClass(nil, nil).SpecKind() != domain.StorageClassSpecKind {
		t.Errorf("storageClass SpecKind mismatch")
	}
}
