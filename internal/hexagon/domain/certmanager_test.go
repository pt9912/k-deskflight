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

func TestCertManagerSpecKind(t *testing.T) {
	spec := domain.CertManagerSpec{}
	if got, want := spec.Kind(), "certManager"; got != want {
		t.Errorf("Kind(): got %q, want %q", got, want)
	}
}

func TestCertManagerSpecValidate(t *testing.T) {
	spec := domain.CertManagerSpec{}
	if err := spec.Validate(context.Background()); err != nil {
		t.Errorf("Validate(): unexpected error %v", err)
	}
}

func TestCertManagerAPIGroup(t *testing.T) {
	if got, want := domain.CertManagerAPIGroup, "cert-manager.io"; got != want {
		t.Errorf("CertManagerAPIGroup: got %q, want %q", got, want)
	}
}
