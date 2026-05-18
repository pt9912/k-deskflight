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

func TestIngressClassSpecKind(t *testing.T) {
	spec := domain.IngressClassSpec{Names: []string{"nginx"}}
	if got, want := spec.Kind(), "ingressClass"; got != want {
		t.Errorf("Kind(): got %q, want %q", got, want)
	}
}

func TestIngressClassSpecValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		spec    domain.IngressClassSpec
		wantErr bool
	}{
		{"single name", domain.IngressClassSpec{Names: []string{"nginx"}}, false},
		{"multiple names", domain.IngressClassSpec{Names: []string{"nginx", "traefik"}}, false},

		{"empty names", domain.IngressClassSpec{}, true},
		{"nil names", domain.IngressClassSpec{Names: nil}, true},
		{"empty entry", domain.IngressClassSpec{Names: []string{"nginx", ""}}, true},
		{"whitespace entry", domain.IngressClassSpec{Names: []string{"\t"}}, true},
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
