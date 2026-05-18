/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package application_test

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pt9912/k-deskflight/internal/hexagon/application"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

// fakeReviewer ist ein konfigurierbarer port.AccessReviewer-Stub.
// Entscheidung pro Permission über die `outcomes`-Map; Default ist
// allowed=true (für die Pfade, die SAR nicht selbst testen).
type fakeReviewer struct {
	outcomes map[string]struct {
		allowed bool
		err     error
	}
	calls atomic.Int32
	// panicOn schaltet eine künstliche Panic im CanI-Pfad an — für
	// den Per-Check-Recover-Test (slice-M5 §2.4, Review-Befund 3).
	panicOn string
}

func (f *fakeReviewer) CanI(_ context.Context, req domain.PermissionRequest) (bool, error) {
	f.calls.Add(1)
	if f.panicOn != "" && req.CanonicalString() == f.panicOn {
		panic("synthetic CanI panic")
	}
	if o, ok := f.outcomes[req.CanonicalString()]; ok {
		return o.allowed, o.err
	}
	return true, nil
}

// runnerCheck ist ein konfigurierbarer domain.Check-Stub für den
// Runner-Test. `run` wird in `Run` aufgerufen und darf das Result
// liefern oder blocken/panicken.
type runnerCheck struct {
	name          string
	conditionType string
	perms         []domain.PermissionRequest
	run           func(ctx context.Context) domain.Result
}

func (c *runnerCheck) Name() string                                       { return c.name }
func (c *runnerCheck) SpecKind() string                                   { return c.name }
func (c *runnerCheck) ConditionType() string                              { return c.conditionType }
func (c *runnerCheck) RequiredPermissions() []domain.PermissionRequest    { return c.perms }
func (c *runnerCheck) Run(ctx context.Context, _ domain.CheckSpec) domain.Result {
	if c.run == nil {
		return domain.Result{Name: c.conditionType, Status: domain.StatusTrue}
	}
	return c.run(ctx)
}

func fixedTime() func() time.Time {
	return func() time.Time {
		return time.Date(2026, time.May, 18, 12, 0, 0, 0, time.UTC)
	}
}

// TestRunCheckSafelyPassedNoPermissions deckt den Discovery-Pfad:
// keine RequiredPermissions, Check liefert True, Runner gibt den
// Check-Result unverändert weiter.
func TestRunCheckSafelyPassedNoPermissions(t *testing.T) {
	t.Parallel()
	want := domain.Result{
		Name:     "KubernetesVersionReady",
		Status:   domain.StatusTrue,
		Reason:   "KubernetesVersionReady",
		Severity: domain.SeverityInfo,
		Message:  "ok",
	}
	chk := &runnerCheck{
		name:          "kubernetesVersion",
		conditionType: "KubernetesVersionReady",
		run:           func(_ context.Context) domain.Result { return want },
	}

	got := application.RunCheckSafely(
		context.Background(), nil, &fakeReviewer{}, application.NewPermissionCache(), chk, nil, fixedTime(), 0)

	if got.Status != domain.StatusTrue || got.Reason != "KubernetesVersionReady" {
		t.Errorf("got %+v, want passed", got)
	}
}

// TestRunCheckSafelyRBACInsufficient: CanI returnt (false, nil) →
// Check wird übersprungen, Result Unknown/RBACInsufficient/critical
// (slice-M5 §2.3).
func TestRunCheckSafelyRBACInsufficient(t *testing.T) {
	t.Parallel()
	perm := domain.PermissionRequest{Group: "storage.k8s.io", Resource: "storageclasses", Verb: "list"}
	rev := &fakeReviewer{
		outcomes: map[string]struct {
			allowed bool
			err     error
		}{
			perm.CanonicalString(): {allowed: false},
		},
	}
	runCalled := false
	chk := &runnerCheck{
		name:          "storageClass",
		conditionType: "StorageClassReady",
		perms:         []domain.PermissionRequest{perm},
		run: func(_ context.Context) domain.Result {
			runCalled = true
			return domain.Result{}
		},
	}

	got := application.RunCheckSafely(
		context.Background(), nil, rev, application.NewPermissionCache(), chk, nil, fixedTime(), 0)

	if runCalled {
		t.Errorf("check.Run must NOT be called when permissions denied")
	}
	if got.Name != "StorageClassReady" {
		t.Errorf("Result.Name: got %q, want %q (ConditionType from check)", got.Name, "StorageClassReady")
	}
	if got.Status != domain.StatusUnknown || got.Reason != "RBACInsufficient" {
		t.Errorf("got %+v, want Unknown/RBACInsufficient", got)
	}
	if got.Severity != domain.SeverityCritical {
		t.Errorf("Severity: got %q, want critical", got.Severity)
	}
	if !strings.Contains(got.Message, "list storage.k8s.io/storageclasses") {
		t.Errorf("Message must name the missing permission; got %q", got.Message)
	}
}

// TestRunCheckSafelyRBACCheckFailed: CanI returnt error → Result
// RBACCheckFailed (separater Reason vs. RBACInsufficient, slice-M5
// §2.3, Review-Befund 1).
func TestRunCheckSafelyRBACCheckFailed(t *testing.T) {
	t.Parallel()
	perm := domain.PermissionRequest{Group: "storage.k8s.io", Resource: "storageclasses", Verb: "list"}
	rev := &fakeReviewer{
		outcomes: map[string]struct {
			allowed bool
			err     error
		}{
			perm.CanonicalString(): {err: errors.New("webhook unreachable")},
		},
	}
	chk := &runnerCheck{
		name:          "storageClass",
		conditionType: "StorageClassReady",
		perms:         []domain.PermissionRequest{perm},
		run:           func(_ context.Context) domain.Result { t.Fatal("Run must not be called"); return domain.Result{} },
	}

	got := application.RunCheckSafely(
		context.Background(), nil, rev, application.NewPermissionCache(), chk, nil, fixedTime(), 0)

	if got.Reason != "RBACCheckFailed" {
		t.Errorf("Reason: got %q, want RBACCheckFailed", got.Reason)
	}
	if got.Status != domain.StatusUnknown || got.Severity != domain.SeverityCritical {
		t.Errorf("got %+v, want Unknown/critical", got)
	}
}

// TestRunCheckSafelyCheckFailedPrecedence: ein Check mit gemischtem
// Outcome (eine Permission denied, eine subsystem-failed) muss
// RBACCheckFailed liefern — das verhindert, dass eine echte
// RBAC-Lücke einen Auth-Subsystem-Ausfall maskiert.
func TestRunCheckSafelyCheckFailedPrecedence(t *testing.T) {
	t.Parallel()
	denied := domain.PermissionRequest{Group: "a", Resource: "r1", Verb: "list"}
	failed := domain.PermissionRequest{Group: "b", Resource: "r2", Verb: "list"}
	rev := &fakeReviewer{
		outcomes: map[string]struct {
			allowed bool
			err     error
		}{
			denied.CanonicalString(): {allowed: false},
			failed.CanonicalString(): {err: errors.New("subsystem down")},
		},
	}
	chk := &runnerCheck{
		name:          "multi",
		conditionType: "MultiReady",
		perms:         []domain.PermissionRequest{denied, failed},
	}

	got := application.RunCheckSafely(
		context.Background(), nil, rev, application.NewPermissionCache(), chk, nil, fixedTime(), 0)

	if got.Reason != "RBACCheckFailed" {
		t.Errorf("Reason: got %q, want RBACCheckFailed (failed > denied precedence)", got.Reason)
	}
}

// TestRunCheckSafelyPanicInCanI: ein Panic im CanI-Pfad muss vom
// per-Check-Recover gefangen werden, NICHT vom äußeren Reconciler-
// Recover (slice-M5 §2.4, Review-Befund 3).
func TestRunCheckSafelyPanicInCanI(t *testing.T) {
	t.Parallel()
	perm := domain.PermissionRequest{Group: "x", Resource: "y", Verb: "list"}
	rev := &fakeReviewer{panicOn: perm.CanonicalString()}
	chk := &runnerCheck{
		name:          "panicky",
		conditionType: "PanickyReady",
		perms:         []domain.PermissionRequest{perm},
	}

	// Test fängt selbst keinen panic — RunCheckSafely darf nicht
	// hochpropagieren.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RunCheckSafely must not let panic escape: %v", r)
		}
	}()

	got := application.RunCheckSafely(
		context.Background(), nil, rev, application.NewPermissionCache(), chk, nil, fixedTime(), 0)

	if got.Reason != "InternalError" {
		t.Errorf("Reason: got %q, want InternalError", got.Reason)
	}
	if got.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want Unknown", got.Status)
	}
}

// TestRunCheckSafelyPanicInRun: Panic in check.Run wird ebenfalls
// gefangen. Symmetrisch zu §2.4-Anforderung.
func TestRunCheckSafelyPanicInRun(t *testing.T) {
	t.Parallel()
	chk := &runnerCheck{
		name:          "boom",
		conditionType: "BoomReady",
		run: func(_ context.Context) domain.Result {
			panic("synthetic run panic")
		},
	}

	got := application.RunCheckSafely(
		context.Background(), nil, &fakeReviewer{}, application.NewPermissionCache(), chk, nil, fixedTime(), 0)

	if got.Reason != "InternalError" {
		t.Errorf("Reason: got %q, want InternalError", got.Reason)
	}
}

// TestRunCheckSafelyPerCheckTimeout: Check hängt länger als der
// 30s-Default → Result Unknown/Timeout (slice-M5 §2.5). Wir nutzen
// runWithTimeout indirekt über RunCheckSafely; um den Test schnell zu
// halten, hängt der Check explizit auf ctx.Done(), und wir cancel den
// Parent-Context künstlich, um den Pfad zu triggern. Das matched dann
// allerdings den Canceled-Pfad — also nutzen wir lieber einen Test mit
// kurzem Parent-Deadline.
//
// Saubere Variante: Parent-Deadline kurz, sodass classifyContextEnd den
// ReconcileTimeout-Pfad nimmt (Parent gewinnt vor Per-Check-Sentinel).
// Test für den eigenen PerCheck-Timeout-Pfad braucht runWithTimeout
// direkt — den exposen wir nicht, also testen wir das via einen
// Check, der lange genug schläft.
//
// Performance: 100ms Parent-Deadline reicht für den Pfad-Test.
func TestRunCheckSafelyParentDeadlineExceeded(t *testing.T) {
	t.Parallel()
	chk := &runnerCheck{
		name:          "slow",
		conditionType: "SlowReady",
		run: func(ctx context.Context) domain.Result {
			<-ctx.Done()
			return domain.Result{Name: "SlowReady", Status: domain.StatusTrue}
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	got := application.RunCheckSafely(
		ctx, nil, &fakeReviewer{}, application.NewPermissionCache(), chk, nil, fixedTime(), 0)

	if got.Reason != "ReconcileTimeout" {
		t.Errorf("Reason: got %q, want ReconcileTimeout (Parent-Deadline)", got.Reason)
	}
	if got.Severity != domain.SeverityCritical {
		t.Errorf("Severity: got %q, want critical", got.Severity)
	}
}

// TestRunCheckSafelyParentCancel: explizites Parent-Cancel → Reason
// ReconcileCanceled/info (slice-M5 §2.5, Cause-aware Klassifikation).
func TestRunCheckSafelyParentCancel(t *testing.T) {
	t.Parallel()
	chk := &runnerCheck{
		name:          "slow",
		conditionType: "SlowReady",
		run: func(ctx context.Context) domain.Result {
			<-ctx.Done()
			return domain.Result{Name: "SlowReady", Status: domain.StatusTrue}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel asynchron nach kurzer Zeit, damit der Run-Pfad blockiert.
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	got := application.RunCheckSafely(
		ctx, nil, &fakeReviewer{}, application.NewPermissionCache(), chk, nil, fixedTime(), 0)

	if got.Reason != "ReconcileCanceled" {
		t.Errorf("Reason: got %q, want ReconcileCanceled", got.Reason)
	}
	if got.Severity != domain.SeverityInfo {
		t.Errorf("Severity: got %q, want info (Cancel ist kein Check-Fehler)", got.Severity)
	}
}

// TestRunCheckSafelyConditionTypeFromCheck verifiziert slice-M5 §2.2:
// synthetic Results nutzen den ConditionType des Checks (StorageClassReady),
// nicht den Spec-Kind (storageClass). Damit bleibt die Condition-Type-
// Identität über alle Reconcile-Pfade konsistent.
func TestRunCheckSafelyConditionTypeFromCheck(t *testing.T) {
	t.Parallel()
	perm := domain.PermissionRequest{Group: "g", Resource: "r", Verb: "list"}
	rev := &fakeReviewer{
		outcomes: map[string]struct {
			allowed bool
			err     error
		}{
			perm.CanonicalString(): {allowed: false},
		},
	}
	chk := &runnerCheck{
		name:          "storageClass",       // Spec-Kind
		conditionType: "StorageClassReady",  // Condition-Type
		perms:         []domain.PermissionRequest{perm},
	}

	got := application.RunCheckSafely(
		context.Background(), nil, rev, application.NewPermissionCache(), chk, nil, fixedTime(), 0)

	if got.Name != "StorageClassReady" {
		t.Errorf("Result.Name: got %q, want %q (Check.ConditionType, NICHT Spec-Kind)",
			got.Name, "StorageClassReady")
	}
}

// TestPermissionCacheDedupesSameRequest fixiert slice-M5 §2.3:
// derselbe `PermissionRequest` über zwei aufeinanderfolgende Checks
// löst nur einen einzigen CanI-Call aus.
func TestPermissionCacheDedupesSameRequest(t *testing.T) {
	t.Parallel()
	perm := domain.PermissionRequest{Group: "g", Resource: "r", Verb: "list"}
	rev := &fakeReviewer{}
	chk1 := &runnerCheck{
		name: "a", conditionType: "AReady",
		perms: []domain.PermissionRequest{perm},
		run:   func(_ context.Context) domain.Result { return domain.Result{Name: "AReady", Status: domain.StatusTrue} },
	}
	chk2 := &runnerCheck{
		name: "b", conditionType: "BReady",
		perms: []domain.PermissionRequest{perm},
		run:   func(_ context.Context) domain.Result { return domain.Result{Name: "BReady", Status: domain.StatusTrue} },
	}

	cache := application.NewPermissionCache()
	application.RunCheckSafely(context.Background(), nil, rev, cache, chk1, nil, fixedTime(), 0)
	application.RunCheckSafely(context.Background(), nil, rev, cache, chk2, nil, fixedTime(), 0)

	if got := rev.calls.Load(); got != 1 {
		t.Errorf("CanI calls: got %d, want 1 (cache dedupe)", got)
	}
}

// TestRunCheckSafelyLastTransitionUsesClock fixiert deterministische
// Zeitstempel-Zuweisung via die `now`-Closure.
func TestRunCheckSafelyLastTransitionUsesClock(t *testing.T) {
	t.Parallel()
	perm := domain.PermissionRequest{Group: "g", Resource: "r", Verb: "list"}
	rev := &fakeReviewer{
		outcomes: map[string]struct {
			allowed bool
			err     error
		}{
			perm.CanonicalString(): {allowed: false},
		},
	}
	chk := &runnerCheck{name: "c", conditionType: "CReady", perms: []domain.PermissionRequest{perm}}

	got := application.RunCheckSafely(
		context.Background(), nil, rev, application.NewPermissionCache(), chk, nil, fixedTime(), 0)

	want := time.Date(2026, time.May, 18, 12, 0, 0, 0, time.UTC)
	if !got.LastTransition.Equal(want) {
		t.Errorf("LastTransition: got %v, want %v", got.LastTransition, want)
	}
}
