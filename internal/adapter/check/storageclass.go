/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package check

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

const (
	// CheckNameStorageClass ist der stabile Identifier (architecture.md AR-012, LH-F-010/-011).
	CheckNameStorageClass = "storageClass"

	// ConditionTypeStorageClassReady ist der Condition-Type im CR-Status.
	ConditionTypeStorageClassReady = "StorageClassReady"

	// Stable Reason-Codes für Aggregator + Watches.
	reasonStorageClassReady          = "StorageClassReady"
	reasonStorageClassMissing        = "StorageClassMissing"
	reasonDefaultStorageClassMissing = "DefaultStorageClassMissing"
	reasonStorageClassInvalidSpec    = "InvalidSpec"
	reasonStorageClassLookupFailed   = "StorageClassLookupFailed"
)

// StorageClass ist die konkrete Implementierung des StorageClass-Checks
// (LH-F-010/-011). Konsumiert `port.StorageClassDiscovery`.
type StorageClass struct {
	disc port.StorageClassDiscovery
	now  func() time.Time
}

// NewStorageClass baut den Check mit einer Discovery-Quelle. `now=nil`
// fällt auf `time.Now` zurück.
func NewStorageClass(disc port.StorageClassDiscovery, now func() time.Time) *StorageClass {
	if now == nil {
		now = time.Now
	}
	return &StorageClass{disc: disc, now: now}
}

// Name erfüllt das Check-Interface.
func (c *StorageClass) Name() string { return CheckNameStorageClass }

// SpecKind erfüllt das Check-Interface.
func (c *StorageClass) SpecKind() string { return domain.StorageClassSpecKind }

// ConditionType erfüllt das Check-Interface (slice-M5 §2.2).
func (c *StorageClass) ConditionType() string { return ConditionTypeStorageClassReady }

// RequiredPermissions deklariert das `list`-Recht auf
// `storage.k8s.io/storageclasses`. Konsistent zu den
// `+kubebuilder:rbac:`-Markern am Reconciler; `rbac_consistency_test.go`
// (slice-M5 §2.2) erzwingt die 1:1-Deckung.
func (c *StorageClass) RequiredPermissions() []domain.PermissionRequest {
	return []domain.PermissionRequest{
		{Group: "storage.k8s.io", Resource: "storageclasses", Verb: "list"},
	}
}

// Run prüft, ob alle konfigurierten StorageClass-Namen vorhanden sind
// und ob — falls verlangt — eine Default-StorageClass markiert ist.
func (c *StorageClass) Run(ctx context.Context, spec domain.CheckSpec) domain.Result {
	s, ok := spec.(domain.StorageClassSpec)
	if !ok || spec.Kind() != c.SpecKind() {
		return c.unknown(reasonStorageClassInvalidSpec,
			fmt.Sprintf("expected spec kind %q, got %q", c.SpecKind(), spec.Kind()))
	}

	classes, err := c.disc.ListStorageClasses(ctx)
	if err != nil {
		return c.unknown(reasonStorageClassLookupFailed,
			fmt.Sprintf("storage class lookup failed: %v", err))
	}

	present := make(map[string]bool, len(classes))
	var hasDefault bool
	for _, cl := range classes {
		present[cl.Name] = true
		if cl.IsDefault {
			hasDefault = true
		}
	}

	var missing []string
	for _, n := range s.Names {
		if !present[n] {
			missing = append(missing, n)
		}
	}

	var diagnostics []string
	if len(missing) > 0 {
		diagnostics = append(diagnostics,
			fmt.Sprintf("missing storage classes: %s", strings.Join(missing, ", ")))
	}
	if s.RequireDefault && !hasDefault {
		diagnostics = append(diagnostics,
			"no default StorageClass present (annotation storageclass.kubernetes.io/is-default-class or legacy beta key required)")
	}

	if len(diagnostics) > 0 {
		reason := reasonStorageClassMissing
		if len(missing) == 0 {
			reason = reasonDefaultStorageClassMissing
		}
		return c.failed(reason, strings.Join(diagnostics, "; "))
	}

	return c.passed(reasonStorageClassReady,
		fmt.Sprintf("storage classes ok (%d configured, default: %s)",
			len(s.Names), defaultLabel(s.RequireDefault, hasDefault)))
}

func defaultLabel(require, has bool) string {
	switch {
	case !require:
		return "not required"
	case has:
		return "present"
	default:
		return "missing"
	}
}

func (c *StorageClass) passed(reason, message string) domain.Result {
	return domain.Result{
		Name:           ConditionTypeStorageClassReady,
		Status:         domain.StatusTrue,
		Reason:         reason,
		Severity:       domain.SeverityInfo,
		Message:        message,
		LastTransition: c.now().UTC(),
	}
}

func (c *StorageClass) failed(reason, message string) domain.Result {
	return domain.Result{
		Name:           ConditionTypeStorageClassReady,
		Status:         domain.StatusFalse,
		Reason:         reason,
		Severity:       domain.SeverityCritical,
		Message:        message,
		LastTransition: c.now().UTC(),
	}
}

func (c *StorageClass) unknown(reason, message string) domain.Result {
	return domain.Result{
		Name:           ConditionTypeStorageClassReady,
		Status:         domain.StatusUnknown,
		Reason:         reason,
		Severity:       domain.SeverityCritical,
		Message:        message,
		LastTransition: c.now().UTC(),
	}
}
