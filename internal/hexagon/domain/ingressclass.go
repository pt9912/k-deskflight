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

// IngressClassSpecKind ist der stabile Spec-Token für den
// IngressClass-Check (architecture.md AR-012, LH-F-012).
const IngressClassSpecKind = "ingressClass"

// IngressClassSpec parametriert den IngressClass-Check.
//
// `Names` enthält die im Cluster erwarteten IngressClass-Namen
// (`LH-F-012`). Mindestens ein Eintrag muss gesetzt sein.
type IngressClassSpec struct {
	Names []string
}

// Kind erfüllt das CheckSpec-Interface.
func (s IngressClassSpec) Kind() string { return IngressClassSpecKind }

// Validate prüft, dass mindestens ein Name gesetzt ist und keine
// Einträge leer sind.
func (s IngressClassSpec) Validate(_ context.Context) error {
	if len(s.Names) == 0 {
		return fmt.Errorf("ingressClass.names: at least one entry required")
	}
	for i, name := range s.Names {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("ingressClass.names[%d]: empty entry", i)
		}
	}
	return nil
}
