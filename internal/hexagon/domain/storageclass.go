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

// StorageClassSpecKind ist der stabile Spec-Token für den
// StorageClass-Check (architecture.md AR-012, LH-F-010/-011).
const StorageClassSpecKind = "storageClass"

// StorageClassSpec parametriert den StorageClass-Check.
//
// `Names` enthält die im Cluster erwarteten StorageClass-Namen
// (`LH-F-010`). `RequireDefault` verlangt zusätzlich die Existenz
// einer als Default markierten StorageClass (`LH-F-011`). Mindestens
// eines von beiden muss gesetzt sein — die Validate-Methode lehnt
// die leere Spec ab, weil sie keine Prüfgrundlage trägt.
type StorageClassSpec struct {
	Names          []string
	RequireDefault bool
}

// Kind erfüllt das CheckSpec-Interface.
func (s StorageClassSpec) Kind() string { return StorageClassSpecKind }

// Validate prüft, dass mindestens ein Name oder `RequireDefault`
// gesetzt ist und keine Name-Einträge leer sind. Existenz im
// Cluster bewertet erst der Adapter.
func (s StorageClassSpec) Validate(_ context.Context) error {
	if len(s.Names) == 0 && !s.RequireDefault {
		return fmt.Errorf("storageClass: either names or requireDefault must be set")
	}
	for i, name := range s.Names {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("storageClass.names[%d]: empty entry", i)
		}
	}
	return nil
}
