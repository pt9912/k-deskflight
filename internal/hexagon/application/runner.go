/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

// defaultCheckTimeout deckelt einen einzelnen Check-Lauf (slice-M5
// §2.5, AR-009 §4 Step 4 Default-Wert). M5 hartkodiert 30 s;
// Env-Override (OPERATOR_STRICT_CONFIG + CHECK_TIMEOUT_SECONDS) kommt
// erst v0.2 zusammen mit AR-010.1.
const defaultCheckTimeout = 30 * time.Second

// Reason-Codes, die der Runner für synthetische Results emittiert
// (slice-M5 §2.3, §2.4, §2.5). Stabile Strings für Aggregator-Dedupe
// und Watch-Konsumenten.
const (
	reasonRBACInsufficient   = "RBACInsufficient"
	reasonRBACCheckFailed    = "RBACCheckFailed"
	reasonCheckTimeout       = "Timeout"
	reasonReconcileTimeout   = "ReconcileTimeout"
	reasonReconcileCanceled  = "ReconcileCanceled"
	reasonInternalError      = "InternalError"
)

// errPerCheckTimeout ist der Sentinel-Cause für `context.WithTimeoutCause`
// im Per-Check-Timeout (slice-M5 §2.5). `context.Cause(runCtx)` unterscheidet
// damit unseren eigenen Deadline von einer ererbten Parent-Deadline
// (z. B. RECONCILE_TIMEOUT_SECONDS).
var errPerCheckTimeout = errors.New("per-check timeout exceeded")

// permOutcome speichert ein einzelnes SAR-Ergebnis pro Reconcile-Lauf
// (slice-M5 §2.3). Der Caller muss `err != nil` zuerst prüfen — bei
// Subsystem-Ausfall ist `allowed` nicht zu interpretieren.
type permOutcome struct {
	allowed bool
	err     error
}

// PermissionCache ist ein Run-lokaler Cache, der `CanI`-Calls
// dedupliziert (slice-M5 §2.3). Mehrere Checks mit derselben
// `PermissionRequest` teilen sich das Ergebnis; damit landen wir bei
// `O(unique permissions)` SAR-Calls pro Reconcile statt
// `O(checks × permissions)`.
type PermissionCache map[domain.PermissionRequest]permOutcome

// NewPermissionCache liefert einen leeren, einsatzbereiten Cache.
func NewPermissionCache() PermissionCache {
	return make(PermissionCache)
}

// RunCheckSafely ist der Pflicht-Wrapper für jeden einzelnen Check-
// Aufruf im Reconcile-Pfad (slice-M5 §2.4). Der äußere `defer/recover`
// umschließt **den gesamten Per-Check-Pfad** — SAR-Loop und
// `runWithTimeout` zusammen — damit ein Panic in `AccessReviewer.CanI`
// nicht den Reconciler-Outer-Recover triggert, sondern nur den
// betroffenen Check auf `Unknown`/`InternalError` degradiert.
//
// `checkTimeout = 0` fällt auf `defaultCheckTimeout` zurück (30 s,
// slice-M5 §2.5). Tests, die den Per-Check-Timeout-Pfad ausüben
// wollen, übergeben einen kürzeren Wert.
//
// Outcome-Reihenfolge (slice-M5 §2.3):
//
//  1. `RBACCheckFailed` gewinnt vor `RBACInsufficient` — Auth-
//     Subsystem-Ausfall ist die wichtigere Operationsmeldung.
//  2. `RBACInsufficient` greift, wenn keine SAR-Calls fehlschlugen,
//     aber mindestens eine Permission denied wurde.
//  3. Sonst läuft der Check über `runWithTimeout`.
func RunCheckSafely(
	ctx context.Context,
	logger *slog.Logger,
	reviewer port.AccessReviewer,
	cache PermissionCache,
	check domain.Check,
	spec domain.CheckSpec,
	now func() time.Time,
	checkTimeout time.Duration,
) (res domain.Result) {
	defer func() {
		if r := recover(); r != nil {
			if logger != nil {
				logger.LogAttrs(ctx, slog.LevelError, "panic recovered in check pipeline",
					SanitizeAttrs(
						slog.String("check", check.Name()),
						slog.Any("recover", r),
						slog.String("stack", string(debug.Stack())),
					)...,
				)
			}
			res = internalErrorResult(check, now())
		}
	}()

	if checkTimeout <= 0 {
		checkTimeout = defaultCheckTimeout
	}

	insufficient, checkFailed := classifyPermissions(ctx, logger, reviewer, cache, check)
	switch {
	case len(checkFailed) > 0:
		return rbacCheckFailedResult(check, checkFailed, now())
	case len(insufficient) > 0:
		return rbacInsufficientResult(check, insufficient, now())
	}

	return runWithTimeout(ctx, check, spec, checkTimeout, now)
}

// classifyPermissions iteriert die `RequiredPermissions` eines Checks
// und sortiert sie in zwei Buckets: `denied` (CanI → false,nil) und
// `failed` (CanI → _, err). Der Cache wird in Place aktualisiert,
// damit Folge-Checks dieselbe Permission nicht erneut prüfen.
func classifyPermissions(
	ctx context.Context,
	logger *slog.Logger,
	reviewer port.AccessReviewer,
	cache PermissionCache,
	check domain.Check,
) (denied, failed []domain.PermissionRequest) {
	for _, p := range check.RequiredPermissions() {
		outcome, cached := cache[p]
		if !cached {
			allowed, err := reviewer.CanI(ctx, p)
			outcome = permOutcome{allowed: allowed, err: err}
			cache[p] = outcome
			if err != nil && logger != nil {
				logger.LogAttrs(ctx, slog.LevelError, "self subject access review failed",
					SanitizeAttrs(
						slog.String("check", check.Name()),
						slog.String("permission", p.CanonicalString()),
						slog.Any("err", err),
					)...,
				)
			}
		}
		switch {
		case outcome.err != nil:
			failed = append(failed, p)
		case !outcome.allowed:
			denied = append(denied, p)
		}
	}
	return denied, failed
}

// runWithTimeout führt `check.Run` mit einer per-Check-Deadline aus
// (slice-M5 §2.5). `context.WithTimeoutCause` setzt unseren Sentinel,
// damit `classifyContextEnd` unseren Per-Check-Timeout von einer
// ererbten Parent-Deadline unterscheidet.
func runWithTimeout(
	parentCtx context.Context,
	check domain.Check,
	spec domain.CheckSpec,
	checkTimeout time.Duration,
	now func() time.Time,
) domain.Result {
	runCtx, cancel := context.WithTimeoutCause(parentCtx, checkTimeout, errPerCheckTimeout)
	defer cancel()

	resultCh := make(chan domain.Result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// Late-Recover-Pfad: die äußere RunCheckSafely-Klausel
				// fängt den Hauptpfad; dieser hier hält die Goroutine
				// vor dem Abrauchen, falls die zeitliche Anordnung den
				// outer recover umgeht (z. B. Goroutine läuft nach
				// Timeout weiter und panickt dann).
				resultCh <- internalErrorResult(check, now())
			}
		}()
		resultCh <- check.Run(runCtx, spec)
	}()

	select {
	case res := <-resultCh:
		// Race-Härtung (Folge-Review-Befund 1): wenn der Context exakt
		// zum gleichen Zeitpunkt End-Zustand erreicht hat, gewinnt die
		// Context-Klassifikation gegen das vermeintliche Check-Result.
		if runCtx.Err() != nil {
			return classifyContextEnd(runCtx, check, checkTimeout, now)
		}
		return res
	case <-runCtx.Done():
		return classifyContextEnd(runCtx, check, checkTimeout, now)
	}
}

// classifyContextEnd unterscheidet drei Ende-Ursachen (slice-M5 §2.5):
// eigener Per-Check-Deadline (Sentinel `errPerCheckTimeout`), ererbter
// Parent-Deadline und expliziter Cancel.
func classifyContextEnd(
	runCtx context.Context,
	check domain.Check,
	checkTimeout time.Duration,
	now func() time.Time,
) domain.Result {
	switch {
	case errors.Is(runCtx.Err(), context.DeadlineExceeded):
		if errors.Is(context.Cause(runCtx), errPerCheckTimeout) {
			return timeoutResult(check, checkTimeout, now())
		}
		return reconcileTimeoutResult(check, now())
	case errors.Is(runCtx.Err(), context.Canceled):
		return reconcileCanceledResult(check, now())
	default:
		// Soll nicht passieren — wir bauen das Ergebnis defensiv als
		// InternalError, damit das Result auf jeden Fall stabil
		// returniert wird.
		return internalErrorResult(check, now())
	}
}

// rbacInsufficientResult baut das Synthetic-Result für den
// „Cluster-Admin muss handeln"-Pfad (slice-M5 §2.3).
func rbacInsufficientResult(check domain.Check, missing []domain.PermissionRequest, ts time.Time) domain.Result {
	return domain.Result{
		Name:           check.ConditionType(),
		Status:         domain.StatusUnknown,
		Reason:         reasonRBACInsufficient,
		Severity:       domain.SeverityCritical,
		Message:        SanitizeMessage(fmt.Sprintf("missing permissions: %s", joinPermissions(missing))),
		LastTransition: ts.UTC(),
	}
}

// rbacCheckFailedResult baut das Synthetic-Result für den
// „Auth-Subsystem-Ausfall"-Pfad (slice-M5 §2.3). Severity ist `critical`
// damit Oncall sofort sieht, dass die Diagnose blockiert ist.
func rbacCheckFailedResult(check domain.Check, failed []domain.PermissionRequest, ts time.Time) domain.Result {
	return domain.Result{
		Name:           check.ConditionType(),
		Status:         domain.StatusUnknown,
		Reason:         reasonRBACCheckFailed,
		Severity:       domain.SeverityCritical,
		Message:        SanitizeMessage(fmt.Sprintf("SelfSubjectAccessReview failed for: %s", joinPermissions(failed))),
		LastTransition: ts.UTC(),
	}
}

// timeoutResult baut das Synthetic-Result, wenn unser eigener
// Per-Check-Deadline zieht (slice-M5 §2.5, Cause = errPerCheckTimeout).
func timeoutResult(check domain.Check, checkTimeout time.Duration, ts time.Time) domain.Result {
	return domain.Result{
		Name:           check.ConditionType(),
		Status:         domain.StatusUnknown,
		Reason:         reasonCheckTimeout,
		Severity:       domain.SeverityCritical,
		Message:        SanitizeMessage(fmt.Sprintf("check exceeded per-check timeout (%s)", checkTimeout)),
		LastTransition: ts.UTC(),
	}
}

// reconcileTimeoutResult baut das Synthetic-Result, wenn ein
// Parent-Deadline (z. B. RECONCILE_TIMEOUT_SECONDS) zieht.
func reconcileTimeoutResult(check domain.Check, ts time.Time) domain.Result {
	return domain.Result{
		Name:           check.ConditionType(),
		Status:         domain.StatusUnknown,
		Reason:         reasonReconcileTimeout,
		Severity:       domain.SeverityCritical,
		Message:        SanitizeMessage("reconcile deadline exceeded before check completed"),
		LastTransition: ts.UTC(),
	}
}

// reconcileCanceledResult baut das Synthetic-Result für expliziten
// Parent-Cancel (Manager-Stop, Operator-Shutdown). Severity `info`,
// damit das Phase-Mapping auf `Warning` schwenkt, nicht `Failed`.
func reconcileCanceledResult(check domain.Check, ts time.Time) domain.Result {
	return domain.Result{
		Name:           check.ConditionType(),
		Status:         domain.StatusUnknown,
		Reason:         reasonReconcileCanceled,
		Severity:       domain.SeverityInfo,
		Message:        SanitizeMessage("reconcile canceled before check completed"),
		LastTransition: ts.UTC(),
	}
}

// internalErrorResult baut das Synthetic-Result für Panic-Pfade
// (slice-M5 §2.4). Severity `critical`, Stack-Trace landet im Logger,
// nicht im Status (LH-SEC-002 / LH-NF-007).
func internalErrorResult(check domain.Check, ts time.Time) domain.Result {
	return domain.Result{
		Name:           check.ConditionType(),
		Status:         domain.StatusUnknown,
		Reason:         reasonInternalError,
		Severity:       domain.SeverityCritical,
		Message:        SanitizeMessage("internal check error (panic recovered)"),
		LastTransition: ts.UTC(),
	}
}

// joinPermissions erzeugt eine alphabetisch sortierte, kommagetrennte
// CanonicalString-Liste für die Diagnose-Message. Sortierung sichert
// deterministische Output für Tests und Aggregator-Dedupe.
func joinPermissions(perms []domain.PermissionRequest) string {
	if len(perms) == 0 {
		return "(none)"
	}
	parts := make([]string, 0, len(perms))
	for _, p := range perms {
		parts = append(parts, p.CanonicalString())
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}
