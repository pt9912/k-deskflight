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
}

func newFakeRegistry() *fakeRegistry {
	return &fakeRegistry{checks: make(map[string]domain.Check)}
}

func (f *fakeRegistry) Register(c domain.Check) { f.checks[c.Name()] = c }

func (f *fakeRegistry) Resolve(name string) (domain.Check, bool) {
	c, ok := f.checks[name]
	return c, ok
}

func (f *fakeRegistry) ListByProfile(
	_ string,
	spec map[string]domain.CheckSpec,
) ([]domain.Check, []port.CheckSelectionIssue) {
	out := make([]domain.Check, 0, len(spec))
	for name := range spec {
		if c, ok := f.checks[name]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

// stubCheck liefert ein vorgegebenes Result, ohne den echten Check-
// Code aus internal/adapter/check/ zu importieren.
type stubCheck struct {
	name   string
	result domain.Result
}

func (s stubCheck) Name() string                                              { return s.name }
func (s stubCheck) SpecKind() string                                          { return domain.KubernetesVersionSpecKind }
func (s stubCheck) Run(_ context.Context, _ domain.CheckSpec) domain.Result   { return s.result }

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
