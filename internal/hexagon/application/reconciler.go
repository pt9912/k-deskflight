/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

// Package application contains the reconciler use-case that orchestrates
// preflight check execution (architecture.md AR-004 hexagonal application
// layer, AR-009 reconcile path).
package application

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	preflightv1alpha1 "github.com/pt9912/k-deskflight/api/v1alpha1"
)

// Reconciler reconciles OpenDeskPreflightCheck resources.
//
// Slice M2 (this implementation) is intentionally minimal: fetch the CR,
// idempotently set status.phase = Passed with an empty Summary and no
// Conditions. The full six-phase AR-009 reconcile path (run-context,
// timeout, cross-field-validate, active-check resolve, aggregate, status
// write) arrives with M3 once the first check (Kubernetes version) is wired.
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k-deskflight.geo-terrain.net,resources=opendeskpreflightchecks,verbs=get;list;watch
// +kubebuilder:rbac:groups=k-deskflight.geo-terrain.net,resources=opendeskpreflightchecks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingressclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=selfsubjectaccessreviews;selfsubjectrulesreviews,verbs=create
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch

// Reconcile implements the controller-runtime Reconcile interface
// (architecture.md AR-009). Slice M2 collapses Pending → Running → Passed
// into a single status write because there is no work between them; M3
// reintroduces the intermediate phases with the first real check.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var cr preflightv1alpha1.OpenDeskPreflightCheck
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if apierrors.IsNotFound(err) {
			// CR deleted between event dispatch and Get: nothing to do.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get OpenDeskPreflightCheck: %w", err)
	}

	// Idempotency: skip if already reconciled at the current generation.
	if cr.Status.Phase == preflightv1alpha1.PhasePassed &&
		cr.Status.ObservedGeneration == cr.Generation {
		return ctrl.Result{}, nil
	}

	now := metav1.Now()
	cr.Status.Phase = preflightv1alpha1.PhasePassed
	cr.Status.ObservedGeneration = cr.Generation
	cr.Status.Summary = preflightv1alpha1.Summary{LastChecked: &now}
	cr.Status.Conditions = nil

	if err := r.Status().Update(ctx, &cr); err != nil {
		return ctrl.Result{}, fmt.Errorf("update OpenDeskPreflightCheck status: %w", err)
	}

	logger.Info("reconciled",
		"name", req.Name,
		"namespace", req.Namespace,
		"phase", cr.Status.Phase,
		"generation", cr.Generation,
	)
	return ctrl.Result{}, nil
}

// SetupWithManager registers the reconciler with the controller-runtime manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&preflightv1alpha1.OpenDeskPreflightCheck{}).
		Named("opendeskpreflightcheck").
		Complete(r)
}
