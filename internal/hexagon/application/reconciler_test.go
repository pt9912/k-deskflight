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
)

// newSchemeWithAPI baut das scheme für die fake-client-Tests und stoppt
// den Test früh, wenn die scheme-Registrierung fehlschlägt — alle anderen
// Assertions hängen davon ab.
func newSchemeWithAPI(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := preflightv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("register v1alpha1 scheme: %v", err)
	}
	return scheme
}

// TestReconcileSmokeTransitionToPassed verifiziert den M2-Minimal-Pfad
// (slice-M2 §7 #6): leerer CR → Reconciler schreibt Phase=Passed,
// ObservedGeneration alignment, leere Conditions, gesetztes
// Summary.LastChecked. M3+ erweitert um echte Check-Aggregation.
func TestReconcileSmokeTransitionToPassed(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "smoke",
			Namespace:  "default",
			Generation: 1,
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	reconciler := &application.Reconciler{Client: client, Scheme: scheme}

	res, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "smoke", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}
	if res.RequeueAfter != 0 {
		t.Fatalf("expected no requeue, got %+v", res)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(
		context.Background(),
		types.NamespacedName{Name: "smoke", Namespace: "default"},
		&after,
	); err != nil {
		t.Fatalf("Get after reconcile: %v", err)
	}

	if after.Status.Phase != preflightv1alpha1.PhasePassed {
		t.Errorf("Status.Phase: got %q, want %q", after.Status.Phase, preflightv1alpha1.PhasePassed)
	}
	if after.Status.ObservedGeneration != cr.Generation {
		t.Errorf("Status.ObservedGeneration: got %d, want %d", after.Status.ObservedGeneration, cr.Generation)
	}
	if len(after.Status.Conditions) != 0 {
		t.Errorf("Status.Conditions: got %d entries, want 0 (M2 has no check logic)", len(after.Status.Conditions))
	}
	if after.Status.Summary.LastChecked == nil {
		t.Error("Status.Summary.LastChecked: got nil, want non-nil")
	}
}

// TestReconcileNotFound verifiziert dass der Reconcile-Pfad sauber
// terminiert, wenn die CR zwischen Event und Get gelöscht wurde
// (architecture.md AR-009 Phase 1: NotFound → kein Requeue).
func TestReconcileNotFound(t *testing.T) {
	scheme := newSchemeWithAPI(t)
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	reconciler := &application.Reconciler{Client: client, Scheme: scheme}

	res, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "gone", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Reconcile on NotFound returned error: %v", err)
	}
	if res.RequeueAfter != 0 {
		t.Errorf("expected no requeue on NotFound, got %+v", res)
	}
}

// TestReconcileIdempotent verifiziert die Idempotency-Klausel im
// Reconciler: wenn Phase bereits Passed ist und ObservedGeneration mit
// metadata.generation übereinstimmt, soll der Reconciler ohne Status-
// Write zurückkehren (kein Hot-Loop).
func TestReconcileIdempotent(t *testing.T) {
	scheme := newSchemeWithAPI(t)

	// Why: metav1.Time serialisiert nur sekundengenau (RFC3339), die
	// fake-client-Storage trunkiert deshalb. Wir nutzen ein festes
	// sekundengenaues Literal, damit der Vergleich nach `Get` stabil ist.
	originalLastChecked := metav1.NewTime(time.Date(2026, time.May, 17, 12, 0, 0, 0, time.UTC))
	cr := &preflightv1alpha1.OpenDeskPreflightCheck{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "idempotent",
			Namespace:  "default",
			Generation: 1,
		},
		Status: preflightv1alpha1.OpenDeskPreflightCheckStatus{
			Phase:              preflightv1alpha1.PhasePassed,
			ObservedGeneration: 1,
			Summary: preflightv1alpha1.Summary{
				LastChecked: &originalLastChecked,
			},
		},
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	reconciler := &application.Reconciler{Client: client, Scheme: scheme}

	if _, err := reconciler.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "idempotent", Namespace: "default"},
	}); err != nil {
		t.Fatalf("Reconcile returned error: %v", err)
	}

	var after preflightv1alpha1.OpenDeskPreflightCheck
	if err := client.Get(
		context.Background(),
		types.NamespacedName{Name: "idempotent", Namespace: "default"},
		&after,
	); err != nil {
		t.Fatalf("Get after reconcile: %v", err)
	}

	if !after.Status.Summary.LastChecked.Equal(&originalLastChecked) {
		t.Errorf("Status.Summary.LastChecked changed: got %v, want %v (idempotent path should not touch status)",
			after.Status.Summary.LastChecked, originalLastChecked)
	}
}
