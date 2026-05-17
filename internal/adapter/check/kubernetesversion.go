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

	"github.com/Masterminds/semver/v3"

	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

const (
	// CheckNameKubernetesVersion ist der stabile Identifier für den
	// KubernetesVersion-Check (architecture.md AR-012, LH-F-008).
	CheckNameKubernetesVersion = "kubernetesVersion"

	// ConditionTypeKubernetesVersionReady ist der Condition-Type im CR-Status.
	ConditionTypeKubernetesVersionReady = "KubernetesVersionReady"

	// Stable Reason-Codes für Aggregator + Watches.
	reasonReady           = "KubernetesVersionReady"
	reasonTooOld          = "KubernetesVersionTooOld"
	reasonInvalidSpec     = "InvalidSpec"
	reasonLookupFailed    = "ServerVersionLookupFailed"
	reasonParseFailed     = "ServerVersionParseFailed"
	reasonMinParseFailed  = "MinVersionParseFailed"
)

// KubernetesVersion ist die konkrete Implementierung des
// KubernetesVersion-Checks (architecture.md AR-012). Konsumiert
// `port.KubernetesAPI` zur Discovery der Server-Version und vergleicht
// gegen den Min-Wert aus `domain.KubernetesVersionSpec`.
type KubernetesVersion struct {
	api port.KubernetesAPI
	now func() time.Time
}

// NewKubernetesVersion baut den Check mit einer KubernetesAPI-Quelle.
// `now`-Override ist als zweiter optionaler Parameter implementiert,
// damit Tests determistische `LastTransition`-Zeitstempel asserten
// können — production-Path nutzt `time.Now`.
func NewKubernetesVersion(api port.KubernetesAPI, now func() time.Time) *KubernetesVersion {
	if now == nil {
		now = time.Now
	}
	return &KubernetesVersion{api: api, now: now}
}

// Name erfüllt das Check-Interface.
func (k *KubernetesVersion) Name() string {
	return CheckNameKubernetesVersion
}

// SpecKind erfüllt das Check-Interface.
func (k *KubernetesVersion) SpecKind() string {
	return domain.KubernetesVersionSpecKind
}

// Run führt die Versions-Discovery + Vergleich aus und liefert ein
// normalisiertes Result. Nicht-Spec-Match wird vom Aufrufer (Registry/
// Reconciler) abgefangen, defensiv prüfen wir hier trotzdem.
func (k *KubernetesVersion) Run(ctx context.Context, spec domain.CheckSpec) domain.Result {
	versionSpec, ok := spec.(domain.KubernetesVersionSpec)
	if !ok || spec.Kind() != k.SpecKind() {
		return k.unknown(reasonInvalidSpec,
			fmt.Sprintf("expected spec kind %q, got %q", k.SpecKind(), spec.Kind()))
	}

	rawVersion, err := k.api.ServerVersion(ctx)
	if err != nil {
		return k.unknown(reasonLookupFailed,
			fmt.Sprintf("server version lookup failed: %v", err))
	}

	serverVer, err := semver.NewVersion(stripVersionPrefix(rawVersion))
	if err != nil {
		return k.unknown(reasonParseFailed,
			fmt.Sprintf("cannot parse server version %q: %v", rawVersion, err))
	}

	minVer, err := semver.NewVersion(stripVersionPrefix(versionSpec.Min))
	if err != nil {
		return k.unknown(reasonMinParseFailed,
			fmt.Sprintf("cannot parse spec.checks.kubernetesVersion.min %q: %v", versionSpec.Min, err))
	}

	if serverVer.LessThan(minVer) {
		return domain.Result{
			Name:           ConditionTypeKubernetesVersionReady,
			Status:         domain.StatusFalse,
			Reason:         reasonTooOld,
			Severity:       domain.SeverityCritical,
			Message:        fmt.Sprintf("server version %s is below configured minimum %s", serverVer.Original(), minVer.Original()),
			LastTransition: k.now().UTC(),
		}
	}

	return domain.Result{
		Name:           ConditionTypeKubernetesVersionReady,
		Status:         domain.StatusTrue,
		Reason:         reasonReady,
		Severity:       domain.SeverityInfo,
		Message:        fmt.Sprintf("server version %s satisfies minimum %s", serverVer.Original(), minVer.Original()),
		LastTransition: k.now().UTC(),
	}
}

func (k *KubernetesVersion) unknown(reason, message string) domain.Result {
	return domain.Result{
		Name:           ConditionTypeKubernetesVersionReady,
		Status:         domain.StatusUnknown,
		Reason:         reason,
		Severity:       domain.SeverityCritical,
		Message:        message,
		LastTransition: k.now().UTC(),
	}
}

// stripVersionPrefix normalisiert Versionsstrings aus discovery
// (z. B. "v1.34.2", "v1.34.2+abc") so dass Masterminds/semver sie
// parsen kann.
func stripVersionPrefix(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")
	// Build-Metadata-Suffix (`+abc`) ist semver-konform; semver-Lib
	// schluckt es. Nichts zu tun.
	return value
}
