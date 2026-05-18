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

func TestStorageClassSpecKind(t *testing.T) {
	spec := domain.StorageClassSpec{Names: []string{"standard"}}
	if got, want := spec.Kind(), "storageClass"; got != want {
		t.Errorf("Kind(): got %q, want %q", got, want)
	}
}

func TestStorageClassSpecValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		spec    domain.StorageClassSpec
		wantErr bool
	}{
		{"names only", domain.StorageClassSpec{Names: []string{"standard"}}, false},
		{"multiple names", domain.StorageClassSpec{Names: []string{"standard", "fast"}}, false},
		{"requireDefault only", domain.StorageClassSpec{RequireDefault: true}, false},
		{"names and requireDefault", domain.StorageClassSpec{Names: []string{"standard"}, RequireDefault: true}, false},

		{"empty spec", domain.StorageClassSpec{}, true},
		{"empty name entry", domain.StorageClassSpec{Names: []string{"standard", ""}}, true},
		{"whitespace name entry", domain.StorageClassSpec{Names: []string{"   "}}, true},
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
