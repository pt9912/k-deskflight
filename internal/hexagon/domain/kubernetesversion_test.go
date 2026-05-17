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

func TestKubernetesVersionSpecKind(t *testing.T) {
	spec := domain.KubernetesVersionSpec{Min: "1.34"}
	if got, want := spec.Kind(), "kubernetesVersion"; got != want {
		t.Errorf("Kind(): got %q, want %q", got, want)
	}
}

func TestKubernetesVersionSpecValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		min     string
		wantErr bool
	}{
		{"major.minor", "1.34", false},
		{"major.minor.patch", "1.34.2", false},
		{"v-prefix major.minor", "v1.34", false},
		{"v-prefix major.minor.patch", "v1.34.2", false},

		{"empty", "", true},
		{"only major", "1", true},
		{"four segments", "1.2.3.4", true},
		{"non-digit", "1.34a", true},
		{"empty middle", "1..3", true},
		{"trailing dot", "1.34.", true},
		{"only v prefix", "v", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			spec := domain.KubernetesVersionSpec{Min: tc.min}
			err := spec.Validate(context.Background())
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate(%q): err = %v, wantErr = %v", tc.min, err, tc.wantErr)
			}
		})
	}
}
