/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package check

import (
	"context"
	"fmt"
	"time"

	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

const (
	// CheckNameCertManager ist der stabile Identifier (architecture.md AR-012, LH-F-013).
	CheckNameCertManager = "certManager"

	// ConditionTypeCertManagerInstalled ist der Condition-Type im CR-Status.
	ConditionTypeCertManagerInstalled = "CertManagerInstalled"

	reasonCertManagerInstalled    = "CertManagerInstalled"
	reasonCertManagerMissing      = "CertManagerMissing"
	reasonCertManagerInvalidSpec  = "InvalidSpec"
	reasonCertManagerLookupFailed = "CertManagerLookupFailed"

	// certManagerMissingMessage ist der Status-Message-Text für den
	// `Missing`-Pfad. **Pflicht-Inhalt** gemäß slice-M4 §3.1 / §7 #11:
	// nennt die zwei legitimen Alternativen (Install oder externe
	// TLS-Terminierung), damit Anwender die Warning-Severity direkt
	// aus dem CR-Status verstehen — Brücke zur M6-Doku
	// (`docs/user/conditions-katalog.md`).
	certManagerMissingMessage = "cert-manager.io API group not registered — install cert-manager or configure external TLS termination (severity warning, not failing)"
)

// CertManager prüft die Existenz der `cert-manager.io`-API-Gruppe
// (`LH-F-013`). Bei Fehlen liefert der Check Severity `warning`
// (slice-M4 §9 — bewusste Entscheidung, kein Outcome-Blocker).
type CertManager struct {
	disc port.APIGroupDiscovery
	now  func() time.Time
}

// NewCertManager baut den Check mit einer APIGroup-Discovery-Quelle.
func NewCertManager(disc port.APIGroupDiscovery, now func() time.Time) *CertManager {
	if now == nil {
		now = time.Now
	}
	return &CertManager{disc: disc, now: now}
}

// Name erfüllt das Check-Interface.
func (c *CertManager) Name() string { return CheckNameCertManager }

// SpecKind erfüllt das Check-Interface.
func (c *CertManager) SpecKind() string { return domain.CertManagerSpecKind }

// Run prüft, ob die cert-manager-API-Gruppe registriert ist.
func (c *CertManager) Run(ctx context.Context, spec domain.CheckSpec) domain.Result {
	if _, ok := spec.(domain.CertManagerSpec); !ok || spec.Kind() != c.SpecKind() {
		return c.unknown(reasonCertManagerInvalidSpec,
			fmt.Sprintf("expected spec kind %q, got %q", c.SpecKind(), spec.Kind()))
	}

	present, err := c.disc.HasAPIGroup(ctx, domain.CertManagerAPIGroup)
	if err != nil {
		return c.unknown(reasonCertManagerLookupFailed,
			fmt.Sprintf("api group lookup failed: %v", err))
	}

	if !present {
		return domain.Result{
			Name:           ConditionTypeCertManagerInstalled,
			Status:         domain.StatusFalse,
			Reason:         reasonCertManagerMissing,
			Severity:       domain.SeverityWarning,
			Message:        certManagerMissingMessage,
			LastTransition: c.now().UTC(),
		}
	}

	return domain.Result{
		Name:           ConditionTypeCertManagerInstalled,
		Status:         domain.StatusTrue,
		Reason:         reasonCertManagerInstalled,
		Severity:       domain.SeverityInfo,
		Message:        fmt.Sprintf("cert-manager.io API group registered (%s)", domain.CertManagerAPIGroup),
		LastTransition: c.now().UTC(),
	}
}

func (c *CertManager) unknown(reason, message string) domain.Result {
	return domain.Result{
		Name:           ConditionTypeCertManagerInstalled,
		Status:         domain.StatusUnknown,
		Reason:         reason,
		Severity:       domain.SeverityCritical,
		Message:        message,
		LastTransition: c.now().UTC(),
	}
}
