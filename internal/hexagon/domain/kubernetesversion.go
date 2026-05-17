/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package domain

import (
	"context"
	"fmt"
	"strings"
)

// KubernetesVersionSpecKind ist der stabile Spec-Token, mit dem der
// Reconciler eine KubernetesVersion-Spec gegen einen Check matcht
// (architecture.md AR-012).
const KubernetesVersionSpecKind = "kubernetesVersion"

// KubernetesVersionSpec ist die domain-seitige CheckSpec für den
// KubernetesVersion-Check (LH-F-008, ADR 0009 §2.2).
//
// Das `Min`-Feld ist ein Semver-String in der Form `<major>.<minor>`
// oder `<major>.<minor>.<patch>`; ein optionales `v`-Prefix wird
// toleriert. Die Validierung passiert in `Validate`, der eigentliche
// Vergleich in der Adapter-Schicht.
type KubernetesVersionSpec struct {
	Min string
}

// Kind erfüllt das CheckSpec-Interface.
func (s KubernetesVersionSpec) Kind() string {
	return KubernetesVersionSpecKind
}

// Validate prüft, dass `Min` ein lesbarer Semver-String ist. Wir machen
// hier eine bewusst minimale Syntax-Prüfung (Pattern matched dem CRD-
// Pattern `^[0-9]+\.[0-9]+(\.[0-9]+)?$` aus AR-006); der Domain-Layer
// darf keine externen Semver-Bibliotheken kennen, deshalb keine
// Constraint-Auswertung hier.
func (s KubernetesVersionSpec) Validate(_ context.Context) error {
	value := strings.TrimPrefix(s.Min, "v")
	if value == "" {
		return fmt.Errorf("kubernetesVersion.min: empty")
	}

	parts := strings.Split(value, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return fmt.Errorf("kubernetesVersion.min: %q is not in form major.minor[.patch]", s.Min)
	}
	for _, segment := range parts {
		if segment == "" {
			return fmt.Errorf("kubernetesVersion.min: %q has empty version segment", s.Min)
		}
		for _, char := range segment {
			if char < '0' || char > '9' {
				return fmt.Errorf("kubernetesVersion.min: %q has non-digit in segment %q", s.Min, segment)
			}
		}
	}
	return nil
}
