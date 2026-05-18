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
	// CheckNameIngressClass ist der stabile Identifier (architecture.md AR-012, LH-F-012).
	CheckNameIngressClass = "ingressClass"

	// ConditionTypeIngressClassReady ist der Condition-Type im CR-Status.
	ConditionTypeIngressClassReady = "IngressClassReady"

	reasonIngressClassReady        = "IngressClassReady"
	reasonIngressClassMissing      = "IngressClassMissing"
	reasonIngressClassInvalidSpec  = "InvalidSpec"
	reasonIngressClassLookupFailed = "IngressClassLookupFailed"
)

// IngressClass ist die konkrete Implementierung des IngressClass-Checks
// (LH-F-012). Konsumiert `port.IngressClassDiscovery`.
type IngressClass struct {
	disc port.IngressClassDiscovery
	now  func() time.Time
}

// NewIngressClass baut den Check mit einer Discovery-Quelle.
func NewIngressClass(disc port.IngressClassDiscovery, now func() time.Time) *IngressClass {
	if now == nil {
		now = time.Now
	}
	return &IngressClass{disc: disc, now: now}
}

// Name erfüllt das Check-Interface.
func (c *IngressClass) Name() string { return CheckNameIngressClass }

// SpecKind erfüllt das Check-Interface.
func (c *IngressClass) SpecKind() string { return domain.IngressClassSpecKind }

// RequiredPermissions deklariert das `list`-Recht auf
// `networking.k8s.io/ingressclasses` (slice-M5 §2.2).
func (c *IngressClass) RequiredPermissions() []domain.PermissionRequest {
	return []domain.PermissionRequest{
		{Group: "networking.k8s.io", Resource: "ingressclasses", Verb: "list"},
	}
}

// Run prüft, ob alle konfigurierten IngressClass-Namen im Cluster
// vorhanden sind.
func (c *IngressClass) Run(ctx context.Context, spec domain.CheckSpec) domain.Result {
	s, ok := spec.(domain.IngressClassSpec)
	if !ok || spec.Kind() != c.SpecKind() {
		return c.unknown(reasonIngressClassInvalidSpec,
			fmt.Sprintf("expected spec kind %q, got %q", c.SpecKind(), spec.Kind()))
	}

	classes, err := c.disc.ListIngressClasses(ctx)
	if err != nil {
		return c.unknown(reasonIngressClassLookupFailed,
			fmt.Sprintf("ingress class lookup failed: %v", err))
	}

	present := make(map[string]bool, len(classes))
	for _, cl := range classes {
		present[cl.Name] = true
	}

	var missing []string
	for _, n := range s.Names {
		if !present[n] {
			missing = append(missing, n)
		}
	}

	if len(missing) > 0 {
		return c.failed(reasonIngressClassMissing,
			fmt.Sprintf("missing ingress classes: %s", strings.Join(missing, ", ")))
	}

	return c.passed(reasonIngressClassReady,
		fmt.Sprintf("ingress classes ok (%d configured)", len(s.Names)))
}

func (c *IngressClass) passed(reason, message string) domain.Result {
	return domain.Result{
		Name:           ConditionTypeIngressClassReady,
		Status:         domain.StatusTrue,
		Reason:         reason,
		Severity:       domain.SeverityInfo,
		Message:        message,
		LastTransition: c.now().UTC(),
	}
}

func (c *IngressClass) failed(reason, message string) domain.Result {
	return domain.Result{
		Name:           ConditionTypeIngressClassReady,
		Status:         domain.StatusFalse,
		Reason:         reason,
		Severity:       domain.SeverityCritical,
		Message:        message,
		LastTransition: c.now().UTC(),
	}
}

func (c *IngressClass) unknown(reason, message string) domain.Result {
	return domain.Result{
		Name:           ConditionTypeIngressClassReady,
		Status:         domain.StatusUnknown,
		Reason:         reason,
		Severity:       domain.SeverityCritical,
		Message:        message,
		LastTransition: c.now().UTC(),
	}
}
