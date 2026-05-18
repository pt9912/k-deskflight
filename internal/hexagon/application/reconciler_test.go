/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package application_test

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	preflightv1alpha1 "github.com/pt9912/k-deskflight/api/v1alpha1"
	"github.com/pt9912/k-deskflight/internal/hexagon/application"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

// fixedClock liefert einen deterministischen Zeitstempel.
func fixedClock() func() time.Time {
	return func() time.Time {
		return time.Date(2026, time.May, 17, 12, 0, 0, 0, time.UTC)
	}
}

// fakeRegistry implementiert port.CheckRegistry für Tests, ohne den
// echten adapter-Package zu importieren (depguard `application-no-adapter`
// gilt auch für _test.go in diesem Pfad).
type fakeRegistry struct {
	checks map[string]domain.Check
	// issues schaltet den Selection-Issue-Pfad an: nicht-leer →
	// ListByProfile gibt diese Issues zurück und kein active-Set.
	issues []port.CheckSelectionIssue
}

func newFakeRegistry() *fakeRegistry {
	return &fakeRegistry{checks: make(map[string]domain.Check)}
}

func (f *fakeRegistry) Register(c domain.Check) { f.checks[c.Name()] = c }

func (f *fakeRegistry) Resolve(name string) (domain.Check, bool) {
	c, ok := f.checks[name]
	return c, ok
}

func (f *fakeRegistry) All() []domain.Check {
	out := make([]domain.Check, 0, len(f.checks))
	for _, c := range f.checks {
		out = append(out, c)
	}
	return out
}

func (f *fakeRegistry) ListByProfile(
	_ string,
	spec map[string]domain.CheckSpec,
) ([]domain.Check, []port.CheckSelectionIssue) {
	if len(f.issues) > 0 {
		return nil, f.issues
	}
	out := make([]domain.Check, 0, len(spec))
	for name := range spec {
		if c, ok := f.checks[name]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

// stubCheck liefert ein vorgegebenes Result, ohne den echten Check-
// Code aus internal/adapter/check/ zu importieren. `kind` bleibt
// für Tests, die nur KubernetesVersion brauchen, optional und fällt
// auf den KubernetesVersion-SpecKind zurück. `conditionType` und
// `permissions` bleiben optional — Tests, die SAR-Pfade nicht
// aktivieren, brauchen nichts zu setzen.
type stubCheck struct {
	name          string
	kind          string
	conditionType string
	result        domain.Result
	permissions   []domain.PermissionRequest
}

func (s stubCheck) Name() string { return s.name }
func (s stubCheck) SpecKind() string {
	if s.kind == "" {
		return domain.KubernetesVersionSpecKind
	}
	return s.kind
}
func (s stubCheck) ConditionType() string {
	if s.conditionType == "" {
		return s.name + "Ready"
	}
	return s.conditionType
}
func (s stubCheck) RequiredPermissions() []domain.PermissionRequest         { return s.permissions }
func (s stubCheck) Run(_ context.Context, _ domain.CheckSpec) domain.Result { return s.result }

// recordingCheck verhält sich wie stubCheck, schreibt aber den
// empfangenen CheckSpec in das Ziel-Pointer-Feld. Verwendet für
// Profile-Default-Tests, die verifizieren, dass der Reconciler die
// richtigen Werte an den Check übergibt.
type recordingCheck struct {
	name          string
	kind          string
	conditionType string
	result        domain.Result
	permissions   []domain.PermissionRequest
	received      *domain.CheckSpec
}

func (s *recordingCheck) Name() string     { return s.name }
func (s *recordingCheck) SpecKind() string { return s.kind }
func (s *recordingCheck) ConditionType() string {
	if s.conditionType == "" {
		return s.name + "Ready"
	}
	return s.conditionType
}
func (s *recordingCheck) RequiredPermissions() []domain.PermissionRequest { return s.permissions }
func (s *recordingCheck) Run(_ context.Context, spec domain.CheckSpec) domain.Result {
	if s.received != nil {
		*s.received = spec
	}
	return s.result
}

func newSchemeWithAPI(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := preflightv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("register v1alpha1 scheme: %v", err)
	}
	return scheme
}

// TestReconcileDefaultActivatesKubernetesVersion verifiziert das
// Default-Aktivierungs-Verhalten (Befund 1 aus Review nach Push c93683a..315b5dd):
// Eine CR ohne expliziten checks.kubernetesVersion-Block aktiviert den
// Check trotzdem, mit dem Default-Min aus ADR 0009 §2.2.
//
// Hinweis ab M4: `buildSpecMap` aktiviert per Default zusätzlich
// cert-manager und clusterResources (slice-M4 §2.2). Dieser Test
// registriert bewusst nur KubernetesVersion in der fakeRegistry, damit
// die Assertion auf K8s-Version fokussiert bleibt — die Multi-Check-
// Default-Aktivierung deckt `TestReconcileMultiCheckAllPassed` ab.
func TestReconcileDefaultActivatesKubernetesVersion(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "default-activation",
			Namespace:  "default",
			Generation: 1,
		},
		// Bewusst leeres Spec — Defaults sollen greifen.
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	registry := newFakeRegistry()
	registry.Register(stubCheck{
		name: domain.KubernetesVersionSpecKind,
		result: domain.Result{
			Name:           "KubernetesVersionReady",
			Status:         domain.StatusTrue,
			Reason:         "KubernetesVersionReady",
			Severity:       domain.SeverityInfo,
			LastTransition: fixedClock()(),
		},
	})

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: registry,
		Now:      fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "default-activation", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "default-activation", Namespace: "default"}, &after); err != nil {
		t.Fatalf("Get after reconcile: %v", err)
	}

	if after.Status.Phase != preflightv1alpha1.PhasePassed {
		t.Errorf("Phase: got %q, want Passed (default activation must run)", after.Status.Phase)
	}
	if after.Status.Summary.ChecksTotal != 1 {
		t.Errorf("ChecksTotal: got %d, want 1 (KubernetesVersion default-active)", after.Status.Summary.ChecksTotal)
	}
	if len(after.Status.Conditions) != 1 || after.Status.Conditions[0].Type != "KubernetesVersionReady" {
		t.Errorf("Conditions: got %+v, want one KubernetesVersionReady entry", after.Status.Conditions)
	}
}

// TestReconcileNotFound deckt den AR-009-Phase-1-NotFound-Pfad.
func TestReconcileNotFound(t *testing.T) {
	scheme := newSchemeWithAPI(t)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: newFakeRegistry(),
		Now:      fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "gone", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile NotFound returned error: %v", err)
	}
}

// TestReconcileIdempotentFailed verifiziert (Befund 2 aus Review):
// auch terminale Non-Passed-Phasen (Failed/Warning/Unknown) skippen
// bei gleicher Generation, damit kein Status-Event-Spam entsteht.
func TestReconcileIdempotentFailed(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	originalLastChecked := metav1.NewTime(time.Date(2026, time.May, 17, 10, 0, 0, 0, time.UTC))
	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "stuck-failed",
			Namespace:  "default",
			Generation: 1,
		},
		Status: preflightv1alpha1.OpenDeskPreflightCheckStatus{
			Phase:              preflightv1alpha1.PhaseFailed,
			ObservedGeneration: 1,
			Summary:            preflightv1alpha1.Summary{LastChecked: &originalLastChecked},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: newFakeRegistry(),
		Now:      fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "stuck-failed", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "stuck-failed", Namespace: "default"}, &after); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if after.Status.Phase != preflightv1alpha1.PhaseFailed {
		t.Errorf("Phase: got %q, want Failed (unchanged)", after.Status.Phase)
	}
	if !after.Status.Summary.LastChecked.Equal(&originalLastChecked) {
		t.Errorf("LastChecked changed: got %v, want %v (Failed at matching generation must not re-write)",
			after.Status.Summary.LastChecked, originalLastChecked)
	}
}

// TestReconcileIdempotent verifiziert die Generation-aligned-Skip-Klausel.
func TestReconcileIdempotent(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	originalLastChecked := metav1.NewTime(time.Date(2026, time.May, 17, 11, 0, 0, 0, time.UTC))
	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "idempotent",
			Namespace:  "default",
			Generation: 1,
		},
		Status: preflightv1alpha1.OpenDeskPreflightCheckStatus{
			Phase:              preflightv1alpha1.PhasePassed,
			ObservedGeneration: 1,
			Summary:            preflightv1alpha1.Summary{LastChecked: &originalLastChecked},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: newFakeRegistry(),
		Now:      fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "idempotent", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "idempotent", Namespace: "default"}, &after); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !after.Status.Summary.LastChecked.Equal(&originalLastChecked) {
		t.Errorf("LastChecked changed: got %v, want %v", after.Status.Summary.LastChecked, originalLastChecked)
	}
}

// TestReconcileKubernetesVersionPassed (M3 §7 #3): registrierter Check
// liefert True/info → Phase=Passed mit KubernetesVersionReady-Condition.
func TestReconcileKubernetesVersionPassed(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "k8s-ok",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: preflightv1alpha1.OpenDeskPreflightCheckSpec{
			Profile: preflightv1alpha1.ProfileProduction,
			Checks: preflightv1alpha1.ChecksSpec{
				KubernetesVersion: &preflightv1alpha1.KubernetesVersionCheckSpec{Min: "1.34"},
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	registry := newFakeRegistry()
	registry.Register(stubCheck{
		name: domain.KubernetesVersionSpecKind,
		result: domain.Result{
			Name:           "KubernetesVersionReady",
			Status:         domain.StatusTrue,
			Reason:         "KubernetesVersionReady",
			Severity:       domain.SeverityInfo,
			Message:        "server version 1.34.2 satisfies minimum 1.34",
			LastTransition: fixedClock()(),
		},
	})

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: registry,
		Now:      fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "k8s-ok", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "k8s-ok", Namespace: "default"}, &after); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if after.Status.Phase != preflightv1alpha1.PhasePassed {
		t.Errorf("Phase: got %q, want Passed", after.Status.Phase)
	}
	if after.Status.Summary.Passed != 1 {
		t.Errorf("Summary.Passed: got %d, want 1", after.Status.Summary.Passed)
	}
	if len(after.Status.Conditions) != 1 || after.Status.Conditions[0].Type != "KubernetesVersionReady" {
		t.Fatalf("Conditions: got %+v, want one KubernetesVersionReady entry", after.Status.Conditions)
	}
	if after.Status.Conditions[0].Severity != preflightv1alpha1.SeverityInfo {
		t.Errorf("Condition.Severity: got %q, want info", after.Status.Conditions[0].Severity)
	}
}

// TestReconcileKubernetesVersionFailed (M3 §7 #3): registrierter Check
// liefert False/critical → Phase=Failed mit KubernetesVersionReady-False.
func TestReconcileKubernetesVersionFailed(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "k8s-old",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: preflightv1alpha1.OpenDeskPreflightCheckSpec{
			Profile: preflightv1alpha1.ProfileProduction,
			Checks: preflightv1alpha1.ChecksSpec{
				KubernetesVersion: &preflightv1alpha1.KubernetesVersionCheckSpec{Min: "99.99"},
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	registry := newFakeRegistry()
	registry.Register(stubCheck{
		name: domain.KubernetesVersionSpecKind,
		result: domain.Result{
			Name:           "KubernetesVersionReady",
			Status:         domain.StatusFalse,
			Reason:         "KubernetesVersionTooOld",
			Severity:       domain.SeverityCritical,
			Message:        "server version 1.34.2 is below configured minimum 99.99",
			LastTransition: fixedClock()(),
		},
	})

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: registry,
		Now:      fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "k8s-old", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "k8s-old", Namespace: "default"}, &after); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if after.Status.Phase != preflightv1alpha1.PhaseFailed {
		t.Errorf("Phase: got %q, want Failed", after.Status.Phase)
	}
	if after.Status.Summary.Failed != 1 {
		t.Errorf("Summary.Failed: got %d, want 1", after.Status.Summary.Failed)
	}
	if len(after.Status.Conditions) != 1 {
		t.Fatalf("Conditions: got %d entries, want 1", len(after.Status.Conditions))
	}
	if after.Status.Conditions[0].Severity != preflightv1alpha1.SeverityCritical {
		t.Errorf("Condition.Severity: got %q, want critical", after.Status.Conditions[0].Severity)
	}
	if after.Status.Conditions[0].Reason != "KubernetesVersionTooOld" {
		t.Errorf("Condition.Reason: got %q, want KubernetesVersionTooOld", after.Status.Conditions[0].Reason)
	}
}

// TestReconcileSpecInvalid (M3 §7): malformed Min triggers Phase=Failed
// via Phase-2-Validation.
func TestReconcileSpecInvalid(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "bad-spec",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: preflightv1alpha1.OpenDeskPreflightCheckSpec{
			Profile: preflightv1alpha1.ProfileProduction,
			Checks: preflightv1alpha1.ChecksSpec{
				KubernetesVersion: &preflightv1alpha1.KubernetesVersionCheckSpec{Min: "garbage"},
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: newFakeRegistry(),
		Now:      fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "bad-spec", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "bad-spec", Namespace: "default"}, &after); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if after.Status.Phase != preflightv1alpha1.PhaseFailed {
		t.Errorf("Phase: got %q, want Failed", after.Status.Phase)
	}
	if len(after.Status.Conditions) != 1 || after.Status.Conditions[0].Type != "SpecInvalid" {
		t.Fatalf("Conditions: got %+v, want one SpecInvalid entry", after.Status.Conditions)
	}
}

// registerAllM4StubChecks ist ein Test-Helper, der die fünf MVP-Checks
// (KubernetesVersion + die vier M4-Checks) als stubs in die fakeRegistry
// einträgt. Jeder Stub liefert ein vorgegebenes Result-Tripel — die
// Tests selbst wählen Status/Severity, der Helper sorgt nur für
// Namens-Vollständigkeit.
func registerAllM4StubChecks(reg *fakeRegistry, results map[string]domain.Result) {
	specs := []struct {
		name string
		kind string
	}{
		{domain.KubernetesVersionSpecKind, domain.KubernetesVersionSpecKind},
		{domain.StorageClassSpecKind, domain.StorageClassSpecKind},
		{domain.IngressClassSpecKind, domain.IngressClassSpecKind},
		{domain.CertManagerSpecKind, domain.CertManagerSpecKind},
		{domain.ClusterResourcesSpecKind, domain.ClusterResourcesSpecKind},
	}
	for _, s := range specs {
		reg.Register(stubCheck{
			name:   s.name,
			kind:   s.kind,
			result: results[s.name],
		})
	}
}

// TestReconcileMultiCheckAllPassed (slice-M4 §3.3 / §7): alle aktiven
// Checks liefern True/info → Phase=Passed, fünf Conditions, Summary
// summiert korrekt. Sample-CR setzt StorageClass und IngressClass
// explizit; KubernetesVersion + CertManager + ClusterResources werden
// per Default aktiviert.
func TestReconcileMultiCheckAllPassed(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "multi-pass",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: preflightv1alpha1.OpenDeskPreflightCheckSpec{
			Profile: preflightv1alpha1.ProfileProduction,
			Checks: preflightv1alpha1.ChecksSpec{
				KubernetesVersion: &preflightv1alpha1.KubernetesVersionCheckSpec{Min: "1.34"},
				StorageClass:      &preflightv1alpha1.StorageClassCheckSpec{Names: []string{"standard"}, RequireDefault: true},
				IngressClass:      &preflightv1alpha1.IngressClassCheckSpec{Names: []string{"nginx"}},
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	now := fixedClock()()
	registry := newFakeRegistry()
	registerAllM4StubChecks(registry, map[string]domain.Result{
		domain.KubernetesVersionSpecKind: passedResult("KubernetesVersionReady", "KubernetesVersionReady", now),
		domain.StorageClassSpecKind:      passedResult("StorageClassReady", "StorageClassReady", now),
		domain.IngressClassSpecKind:      passedResult("IngressClassReady", "IngressClassReady", now),
		domain.CertManagerSpecKind:       passedResult("CertManagerInstalled", "CertManagerInstalled", now),
		domain.ClusterResourcesSpecKind:  passedResult("ClusterResourcesReady", "ResourcesSufficient", now),
	})

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: registry,
		Now:      fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "multi-pass", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "multi-pass", Namespace: "default"}, &after); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if after.Status.Phase != preflightv1alpha1.PhasePassed {
		t.Errorf("Phase: got %q, want Passed", after.Status.Phase)
	}
	if after.Status.Summary.ChecksTotal != 5 {
		t.Errorf("ChecksTotal: got %d, want 5 (KubernetesVersion + 4 M4 checks)", after.Status.Summary.ChecksTotal)
	}
	if after.Status.Summary.Passed != 5 {
		t.Errorf("Summary.Passed: got %d, want 5", after.Status.Summary.Passed)
	}
	if len(after.Status.Conditions) != 5 {
		t.Errorf("Conditions: got %d, want 5; entries=%+v", len(after.Status.Conditions), after.Status.Conditions)
	}
}

// TestReconcileMultiCheckMixedFailed (slice-M4 §3.3 / §7): ein
// critical-Failed kollabiert die Gesamtphase auf Failed, andere
// Passed-Checks bleiben mit eigenen Conditions sichtbar.
func TestReconcileMultiCheckMixedFailed(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "multi-mixed",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: preflightv1alpha1.OpenDeskPreflightCheckSpec{
			Profile: preflightv1alpha1.ProfileProduction,
			Checks: preflightv1alpha1.ChecksSpec{
				KubernetesVersion: &preflightv1alpha1.KubernetesVersionCheckSpec{Min: "1.34"},
				StorageClass:      &preflightv1alpha1.StorageClassCheckSpec{Names: []string{"missing-class"}},
				IngressClass:      &preflightv1alpha1.IngressClassCheckSpec{Names: []string{"nginx"}},
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	now := fixedClock()()
	registry := newFakeRegistry()
	registerAllM4StubChecks(registry, map[string]domain.Result{
		domain.KubernetesVersionSpecKind: passedResult("KubernetesVersionReady", "KubernetesVersionReady", now),
		domain.StorageClassSpecKind: {
			Name:           "StorageClassReady",
			Status:         domain.StatusFalse,
			Reason:         "StorageClassMissing",
			Severity:       domain.SeverityCritical,
			Message:        "missing storage classes: missing-class",
			LastTransition: now,
		},
		domain.IngressClassSpecKind:     passedResult("IngressClassReady", "IngressClassReady", now),
		domain.CertManagerSpecKind:      passedResult("CertManagerInstalled", "CertManagerInstalled", now),
		domain.ClusterResourcesSpecKind: passedResult("ClusterResourcesReady", "ResourcesSufficient", now),
	})

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: registry,
		Now:      fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "multi-mixed", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "multi-mixed", Namespace: "default"}, &after); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if after.Status.Phase != preflightv1alpha1.PhaseFailed {
		t.Errorf("Phase: got %q, want Failed (one critical fail collapses overall phase)", after.Status.Phase)
	}
	if after.Status.Summary.Failed != 1 {
		t.Errorf("Summary.Failed: got %d, want 1", after.Status.Summary.Failed)
	}
	if after.Status.Summary.Passed != 4 {
		t.Errorf("Summary.Passed: got %d, want 4", after.Status.Summary.Passed)
	}
	if len(after.Status.Conditions) != 5 {
		t.Errorf("Conditions: got %d, want 5; entries=%+v", len(after.Status.Conditions), after.Status.Conditions)
	}
}

// TestReconcileProfileDefaultsProduction (slice-M4 §2.3 / §3.3):
// production-Profil setzt ClusterResources auf 4 CPU / 8Gi, wenn die
// CR keinen expliziten Sub-Spec liefert.
func TestReconcileProfileDefaultsProduction(t *testing.T) {
	assertClusterResourcesDefaults(t, preflightv1alpha1.ProfileProduction,
		domain.DefaultClusterResourcesMinCPUProduction,
		domain.DefaultClusterResourcesMinMemoryProduction)
}

// TestReconcileProfileDefaultsEvaluation (slice-M4 §2.3 / §3.3):
// evaluation-Profil setzt ClusterResources auf 2 CPU / 4Gi.
func TestReconcileProfileDefaultsEvaluation(t *testing.T) {
	assertClusterResourcesDefaults(t, preflightv1alpha1.ProfileEvaluation,
		domain.DefaultClusterResourcesMinCPUEvaluation,
		domain.DefaultClusterResourcesMinMemoryEvaluation)
}

func assertClusterResourcesDefaults(t *testing.T, profile preflightv1alpha1.Profile, wantCPU, wantMem string) {
	t.Helper()
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "profile-defaults-" + string(profile),
			Namespace:  "default",
			Generation: 1,
		},
		Spec: preflightv1alpha1.OpenDeskPreflightCheckSpec{
			Profile: profile,
			// Bewusst leerer Checks-Block — ClusterResources soll per
			// Profile-Default aktiviert werden.
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	var receivedSpec domain.CheckSpec
	registry := newFakeRegistry()
	registry.Register(&recordingCheck{
		name:     domain.ClusterResourcesSpecKind,
		kind:     domain.ClusterResourcesSpecKind,
		received: &receivedSpec,
		result: domain.Result{
			Name:           "ClusterResourcesReady",
			Status:         domain.StatusTrue,
			Reason:         "ResourcesSufficient",
			Severity:       domain.SeverityInfo,
			LastTransition: fixedClock()(),
		},
	})

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: registry,
		Now:      fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: cr.Name, Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	got, ok := receivedSpec.(domain.ClusterResourcesSpec)
	if !ok {
		t.Fatalf("recorded spec is %T, want ClusterResourcesSpec", receivedSpec)
	}
	if got.MinCPU != wantCPU {
		t.Errorf("MinCPU for profile %q: got %q, want %q", profile, got.MinCPU, wantCPU)
	}
	if got.MinMemory != wantMem {
		t.Errorf("MinMemory for profile %q: got %q, want %q", profile, got.MinMemory, wantMem)
	}
}

// permissionedCheck deklariert eine Permission und dient dem
// SetupWithManager-Validation-Test (slice-M5 Review-Befund 5).
type permissionedCheck struct{}

func (permissionedCheck) Name() string          { return "permissioned" }
func (permissionedCheck) SpecKind() string      { return "permissioned" }
func (permissionedCheck) ConditionType() string { return "PermissionedReady" }
func (permissionedCheck) RequiredPermissions() []domain.PermissionRequest {
	return []domain.PermissionRequest{{Group: "g", Resource: "r", Verb: "list"}}
}
func (permissionedCheck) Run(_ context.Context, _ domain.CheckSpec) domain.Result {
	return domain.Result{}
}

// TestValidateConfigRejectsMissingAccessReviewer (slice-M5 Review-
// Befund 5): wenn mindestens ein registrierter Check
// RequiredPermissions deklariert, MUSS der Reconciler einen
// AccessReviewer haben. Lücke wird zur Startup-Zeit erkannt, nicht
// zur Smoke-Laufzeit als InternalError.
func TestValidateConfigRejectsMissingAccessReviewer(t *testing.T) {
	t.Parallel()
	registry := newFakeRegistry()
	registry.Register(permissionedCheck{})

	reconciler := &application.Reconciler{
		Registry: registry,
		// AccessReviewer bewusst nil
	}
	if err := reconciler.Validate(); err == nil {
		t.Errorf("Validate: expected error, got nil (RequiredPermissions ohne AccessReviewer)")
	}
}

// TestValidateConfigAllowsNoPermissionedChecks: wenn alle Checks
// RequiredPermissions=nil haben (Discovery-only-Checks), darf
// AccessReviewer nil bleiben.
func TestValidateConfigAllowsNoPermissionedChecks(t *testing.T) {
	t.Parallel()
	registry := newFakeRegistry()
	registry.Register(stubCheck{name: "no-perms", conditionType: "NoPermsReady"})

	reconciler := &application.Reconciler{Registry: registry}
	if err := reconciler.Validate(); err != nil {
		t.Errorf("Validate: unexpected error %v", err)
	}
}

// fakeAccessReviewer ist ein konfigurierbarer port.AccessReviewer für
// die slice-M5-Reconciler-Tests. `outcomes` mapped CanonicalString →
// (allowed, err); Default ist (true, nil).
type fakeAccessReviewer struct {
	outcomes map[string]struct {
		allowed bool
		err     error
	}
}

func (f *fakeAccessReviewer) CanI(_ context.Context, req domain.PermissionRequest) (bool, error) {
	if o, ok := f.outcomes[req.CanonicalString()]; ok {
		return o.allowed, o.err
	}
	return true, nil
}

// panickingCheck panickt während Run; wird vom Per-Check-Recover
// (RunCheckSafely) als InternalError klassifiziert (slice-M5 §2.4).
type panickingCheck struct{}

func (panickingCheck) Name() string                                    { return domain.KubernetesVersionSpecKind }
func (panickingCheck) SpecKind() string                                { return domain.KubernetesVersionSpecKind }
func (panickingCheck) ConditionType() string                           { return "KubernetesVersionReady" }
func (panickingCheck) RequiredPermissions() []domain.PermissionRequest { return nil }
func (panickingCheck) Run(_ context.Context, _ domain.CheckSpec) domain.Result {
	panic("synthetic check panic")
}

// hangingCheck blockiert bis ctx.Done(); simuliert Per-Check-Timeout
// (slice-M5 §2.5).
type hangingCheck struct{}

func (hangingCheck) Name() string                                    { return domain.KubernetesVersionSpecKind }
func (hangingCheck) SpecKind() string                                { return domain.KubernetesVersionSpecKind }
func (hangingCheck) ConditionType() string                           { return "KubernetesVersionReady" }
func (hangingCheck) RequiredPermissions() []domain.PermissionRequest { return nil }
func (hangingCheck) Run(ctx context.Context, _ domain.CheckSpec) domain.Result {
	<-ctx.Done()
	return domain.Result{Name: "KubernetesVersionReady", Status: domain.StatusTrue}
}

// panickingRegistry panickt in ListByProfile; testet den Outer-Recover
// (slice-M5 §2.4).
type panickingRegistry struct{}

func (panickingRegistry) Register(_ domain.Check)               {}
func (panickingRegistry) Resolve(_ string) (domain.Check, bool) { return nil, false }
func (panickingRegistry) All() []domain.Check                   { return nil }
func (panickingRegistry) ListByProfile(_ string, _ map[string]domain.CheckSpec) ([]domain.Check, []port.CheckSelectionIssue) {
	panic("synthetic registry panic")
}

// TestReconcileRBACInsufficient (slice-M5 §7 #12): CanI returnt
// (false, nil) für eine deklarierte Permission → Result Unknown +
// Reason RBACInsufficient + Severity critical.
func TestReconcileRBACInsufficient(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "rbac-denied",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: preflightv1alpha1.OpenDeskPreflightCheckSpec{
			Profile: preflightv1alpha1.ProfileProduction,
			Checks: preflightv1alpha1.ChecksSpec{
				KubernetesVersion: &preflightv1alpha1.KubernetesVersionCheckSpec{Min: "1.34"},
			},
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).WithStatusSubresource(cr).Build()

	perm := domain.PermissionRequest{Group: "storage.k8s.io", Resource: "storageclasses", Verb: "list"}
	registry := newFakeRegistry()
	registry.Register(stubCheck{
		name:          domain.KubernetesVersionSpecKind,
		kind:          domain.KubernetesVersionSpecKind,
		conditionType: "KubernetesVersionReady",
		permissions:   []domain.PermissionRequest{perm},
		result:        domain.Result{Name: "KubernetesVersionReady", Status: domain.StatusTrue},
	})
	reviewer := &fakeAccessReviewer{outcomes: map[string]struct {
		allowed bool
		err     error
	}{perm.CanonicalString(): {allowed: false}}}

	reconciler := &application.Reconciler{
		Client:         client,
		Scheme:         scheme,
		Registry:       registry,
		AccessReviewer: reviewer,
		Now:            fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "rbac-denied", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "rbac-denied", Namespace: "default"}, &after); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if after.Status.Phase != preflightv1alpha1.PhaseUnknown {
		t.Errorf("Phase: got %q, want Unknown", after.Status.Phase)
	}
	if len(after.Status.Conditions) != 1 {
		t.Fatalf("Conditions: got %d, want 1", len(after.Status.Conditions))
	}
	if after.Status.Conditions[0].Reason != "RBACInsufficient" {
		t.Errorf("Reason: got %q, want RBACInsufficient", after.Status.Conditions[0].Reason)
	}
}

// TestReconcilePerCheckPanic (slice-M5 §7 #9): Panic in Check.Run wird
// vom Per-Check-Recover gefangen → Result Unknown/InternalError. Der
// Reconciler kommt ans Ende durch und schreibt Status.
func TestReconcilePerCheckPanic(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "check-panic",
			Namespace:  "default",
			Generation: 1,
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).WithStatusSubresource(cr).Build()

	registry := newFakeRegistry()
	registry.checks[domain.KubernetesVersionSpecKind] = panickingCheck{}

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: registry,
		Now:      fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "check-panic", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v (per-check-recover must NOT propagate)", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "check-panic", Namespace: "default"}, &after); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(after.Status.Conditions) == 0 {
		t.Fatal("Conditions: empty (per-check-recover must produce InternalError result)")
	}
	if after.Status.Conditions[0].Reason != "InternalError" {
		t.Errorf("Reason: got %q, want InternalError", after.Status.Conditions[0].Reason)
	}
}

// TestReconcilePerCheckTimeout (slice-M5 §7 #9): Check hängt länger
// als CheckTimeout → Result Unknown/Timeout. 50ms-CheckTimeout für
// deterministische Test-Laufzeit.
func TestReconcilePerCheckTimeout(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "check-timeout",
			Namespace:  "default",
			Generation: 1,
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).WithStatusSubresource(cr).Build()

	registry := newFakeRegistry()
	registry.checks[domain.KubernetesVersionSpecKind] = hangingCheck{}

	reconciler := &application.Reconciler{
		Client:       client,
		Scheme:       scheme,
		Registry:     registry,
		CheckTimeout: 50 * time.Millisecond,
		Now:          fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "check-timeout", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "check-timeout", Namespace: "default"}, &after); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(after.Status.Conditions) == 0 {
		t.Fatal("Conditions: empty (timeout-recover must produce Timeout result)")
	}
	if after.Status.Conditions[0].Reason != "Timeout" {
		t.Errorf("Reason: got %q, want Timeout", after.Status.Conditions[0].Reason)
	}
	if after.Status.Conditions[0].Severity != preflightv1alpha1.SeverityCritical {
		t.Errorf("Severity: got %q, want critical", after.Status.Conditions[0].Severity)
	}
}

// TestReconcileOuterPanic (slice-M5 §7 #9): Panic AUSSERHALB der
// Per-Check-Pipeline (z. B. in der Registry) wird vom Reconciler-Outer-
// Recover gefangen → Status Phase=Unknown, Condition=ReconcileError,
// Reason=ReconcilePanic.
func TestReconcileOuterPanic(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "outer-panic",
			Namespace:  "default",
			Generation: 1,
		},
	}
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cr).WithStatusSubresource(cr).Build()

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: panickingRegistry{},
		Now:      fixedClock(),
	}

	_, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "outer-panic", Namespace: "default"},
	})
	if err == nil {
		t.Errorf("Reconcile: expected error from outer-recover, got nil")
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if getErr := client.Get(context.Background(),
		types.NamespacedName{Name: "outer-panic", Namespace: "default"}, &after); getErr != nil {
		t.Fatalf("Get: %v", getErr)
	}
	if after.Status.Phase != preflightv1alpha1.PhaseUnknown {
		t.Errorf("Phase: got %q, want Unknown (best-effort outer-recover status write)", after.Status.Phase)
	}
	if len(after.Status.Conditions) != 1 || after.Status.Conditions[0].Type != "ReconcileError" {
		t.Fatalf("Conditions: got %+v, want one ReconcileError entry", after.Status.Conditions)
	}
	if after.Status.Conditions[0].Reason != "ReconcilePanic" {
		t.Errorf("Reason: got %q, want ReconcilePanic", after.Status.Conditions[0].Reason)
	}
}

// passedResult ist ein kleiner Test-Helper für die Multi-Check-Tests.
func passedResult(condType, reason string, now time.Time) domain.Result {
	return domain.Result{
		Name:           condType,
		Status:         domain.StatusTrue,
		Reason:         reason,
		Severity:       domain.SeverityInfo,
		LastTransition: now,
	}
}

// TestReconcileSelectionIssue verifiziert den AR-009-Phase-3-
// Selection-Issue-Pfad: Registry meldet einen `UnknownCheck`-Issue →
// Reconciler endet mit Phase=Failed und SpecInvalid-Condition.
func TestReconcileSelectionIssue(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "selection-issue",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: preflightv1alpha1.OpenDeskPreflightCheckSpec{
			Profile: preflightv1alpha1.ProfileProduction,
			Checks: preflightv1alpha1.ChecksSpec{
				KubernetesVersion: &preflightv1alpha1.KubernetesVersionCheckSpec{Min: "1.34"},
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	registry := newFakeRegistry()
	registry.issues = []port.CheckSelectionIssue{
		{Name: "phantomCheck", Reason: "UnknownCheck"},
	}

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: registry,
		Now:      fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "selection-issue", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "selection-issue", Namespace: "default"}, &after); err != nil {
		t.Fatalf("Get: %v", err)
	}

	if after.Status.Phase != preflightv1alpha1.PhaseFailed {
		t.Errorf("Phase: got %q, want Failed", after.Status.Phase)
	}
	if len(after.Status.Conditions) != 1 || after.Status.Conditions[0].Type != "SpecInvalid" {
		t.Fatalf("Conditions: got %+v, want one SpecInvalid entry", after.Status.Conditions)
	}
	if after.Status.Conditions[0].Reason != "UnknownCheck" {
		t.Errorf("Reason: got %q, want UnknownCheck", after.Status.Conditions[0].Reason)
	}
}

// TestReconcileNowFallback deckt den Now-nil-Branch in `(r *Reconciler).now()`:
// wenn r.Now nicht gesetzt ist, fällt der Reconciler auf time.Now zurück.
// Der Test verifiziert nur, dass Reconcile mit r.Now=nil nicht panickt
// und einen plausiblen Zeitstempel schreibt (nicht den fixedClock-Wert).
func TestReconcileNowFallback(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "now-fallback",
			Namespace:  "default",
			Generation: 1,
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: newFakeRegistry(),
		// Now bewusst nicht gesetzt — Default-Fallback zieht.
	}

	before := time.Now()
	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "now-fallback", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	after := time.Now()

	var got preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "now-fallback", Namespace: "default"}, &got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status.Summary.LastChecked == nil {
		t.Fatalf("LastChecked nil — Reconciler ist offenbar nicht durchgelaufen")
	}
	stamp := got.Status.Summary.LastChecked.Time
	if stamp.Before(before.Add(-time.Second)) || stamp.After(after.Add(time.Second)) {
		t.Errorf("LastChecked %v outside expected window [%v..%v]", stamp, before, after)
	}
}

// TestReconcileSpecInvalidIngressClass deckt den buildSpecMap-
// Validation-Pfad für einen M4-Check (Plan §3.3) — bewusst leere
// Names-Liste löst die Validate-Klausel der IngressClassSpec aus.
func TestReconcileSpecInvalidIngressClass(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "bad-ingress",
			Namespace:  "default",
			Generation: 1,
		},
		Spec: preflightv1alpha1.OpenDeskPreflightCheckSpec{
			Profile: preflightv1alpha1.ProfileProduction,
			Checks: preflightv1alpha1.ChecksSpec{
				IngressClass: &preflightv1alpha1.IngressClassCheckSpec{Names: nil},
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	reconciler := &application.Reconciler{
		Client:   client,
		Scheme:   scheme,
		Registry: newFakeRegistry(),
		Now:      fixedClock(),
	}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "bad-ingress", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(context.Background(),
		types.NamespacedName{Name: "bad-ingress", Namespace: "default"}, &after); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if after.Status.Phase != preflightv1alpha1.PhaseFailed {
		t.Errorf("Phase: got %q, want Failed", after.Status.Phase)
	}
	if len(after.Status.Conditions) != 1 || after.Status.Conditions[0].Type != "SpecInvalid" {
		t.Fatalf("Conditions: got %+v, want one SpecInvalid entry", after.Status.Conditions)
	}
}
