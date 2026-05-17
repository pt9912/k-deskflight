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
	"errors"
	"fmt"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	preflightv1alpha1 "github.com/pt9912/k-deskflight/api/v1alpha1"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

// defaultReconcileTimeout deckelt einen Reconcile-Lauf
// (architecture.md AR-009 Phase 1, M3-minimal). M5 ergänzt
// Cross-Constraint-Auswertung gegen CHECK_TIMEOUT_SECONDS und
// OPERATOR_STRICT_CONFIG.
const defaultReconcileTimeout = 120 * time.Second

// Reconciler reconciles OpenDeskPreflightCheck resources
// (architecture.md AR-009). M3 implementiert die Phasen 1+3+4
// (sequenziell, ohne Worker-Pool) + 5 + 6. Voller AR-009-Pfad
// (Worker-Pool, Panic-Boundary, Cross-Constraint-Härtung) kommt
// mit M5.
type Reconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Registry port.CheckRegistry
	Now      func() time.Time
}

// +kubebuilder:rbac:groups=k-deskflight.geo-terrain.net,resources=opendeskpreflightchecks,verbs=get;list;watch
// +kubebuilder:rbac:groups=k-deskflight.geo-terrain.net,resources=opendeskpreflightchecks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingressclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=selfsubjectaccessreviews;selfsubjectrulesreviews,verbs=create
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch

// Reconcile implements the controller-runtime Reconcile interface.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	runCtx, cancel := context.WithTimeout(ctx, defaultReconcileTimeout)
	defer cancel()

	var cr preflightv1alpha1.OpenDeskPreflightCheck
	if err := r.Get(runCtx, req.NamespacedName, &cr); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get OpenDeskPreflightCheck: %w", err)
	}

	if r.isAlreadyReconciled(&cr) {
		return ctrl.Result{}, nil
	}

	specs, validationErrs := buildSpecMap(runCtx, &cr.Spec.Checks)
	if len(validationErrs) > 0 {
		return ctrl.Result{}, r.writeStatus(runCtx, &cr, r.specInvalidOutput(validationErrs))
	}

	profile := profileWithDefault(cr.Spec.Profile)
	active, issues := r.Registry.ListByProfile(profile, specs)
	if len(issues) > 0 {
		return ctrl.Result{}, r.writeStatus(runCtx, &cr, r.selectionIssueOutput(issues, len(specs)))
	}

	results := r.runChecks(runCtx, active, specs)
	out := Aggregate(results, r.now())

	logger.Info("reconciled",
		"name", req.Name,
		"namespace", req.Namespace,
		"phase", out.Phase,
		"generation", cr.Generation,
		"checksTotal", out.Summary.ChecksTotal,
		"failed", out.Summary.Failed,
		"warning", out.Summary.Warning,
		"unknown", out.Summary.Unknown,
	)

	return ctrl.Result{}, r.writeStatus(runCtx, &cr, out)
}

// SetupWithManager registers the reconciler with the controller-runtime manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&preflightv1alpha1.OpenDeskPreflightCheck{}).
		Named("opendeskpreflightcheck").
		Complete(r)
}

func (r *Reconciler) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

func (r *Reconciler) isAlreadyReconciled(cr *preflightv1alpha1.OpenDeskPreflightCheck) bool {
	return cr.Status.Phase == preflightv1alpha1.PhasePassed &&
		cr.Status.ObservedGeneration == cr.Generation
}

func (r *Reconciler) runChecks(
	ctx context.Context,
	active []domain.Check,
	specs map[string]domain.CheckSpec,
) []domain.Result {
	results := make([]domain.Result, 0, len(active))
	for _, check := range active {
		spec, ok := specs[check.Name()]
		if !ok {
			continue
		}
		results = append(results, check.Run(ctx, spec))
	}
	return results
}

// writeStatus persistiert das AggregationOutput in den CR-Status
// (AR-009 Phase 6). ObservedGeneration wird auf
// metadata.generation gehoben.
func (r *Reconciler) writeStatus(
	ctx context.Context,
	cr *preflightv1alpha1.OpenDeskPreflightCheck,
	out AggregationOutput,
) error {
	cr.Status.Phase = out.Phase
	cr.Status.ObservedGeneration = cr.Generation
	cr.Status.Summary = out.Summary
	cr.Status.Conditions = out.Conditions

	if err := r.Status().Update(ctx, cr); err != nil {
		return fmt.Errorf("update OpenDeskPreflightCheck status: %w", err)
	}
	return nil
}

// specInvalidOutput formuliert die Phase-Failed-Antwort, wenn
// CheckSpec.Validate fehlschlägt (AR-009 Phase 2 minimal).
func (r *Reconciler) specInvalidOutput(errs []specError) AggregationOutput {
	now := r.now().UTC()
	messages := make([]string, 0, len(errs))
	for _, e := range errs {
		messages = append(messages, fmt.Sprintf("%s: %s", e.check, e.err.Error()))
	}
	return AggregationOutput{
		Phase: preflightv1alpha1.PhaseFailed,
		Summary: preflightv1alpha1.Summary{
			ChecksTotal: 0,
			LastChecked: ptrMetaTime(metav1.NewTime(now)),
		},
		Conditions: []preflightv1alpha1.Condition{{
			Type:               "SpecInvalid",
			Status:             metav1.ConditionFalse,
			Reason:             "SpecInvalid",
			Message:            strings.Join(messages, "; "),
			LastTransitionTime: metav1.NewTime(now),
			Severity:           preflightv1alpha1.SeverityCritical,
		}},
	}
}

// selectionIssueOutput formuliert die Phase-Failed-Antwort, wenn
// CheckRegistry-Resolution unauflösbare Einträge meldet (AR-009 Phase 3).
func (r *Reconciler) selectionIssueOutput(
	issues []port.CheckSelectionIssue,
	specCount int,
) AggregationOutput {
	now := r.now().UTC()
	messages := make([]string, 0, len(issues))
	for _, i := range issues {
		messages = append(messages, fmt.Sprintf("%s: %s", i.Name, i.Reason))
	}
	return AggregationOutput{
		Phase: preflightv1alpha1.PhaseFailed,
		Summary: preflightv1alpha1.Summary{
			ChecksTotal: int32(specCount),
			LastChecked: ptrMetaTime(metav1.NewTime(now)),
		},
		Conditions: []preflightv1alpha1.Condition{{
			Type:               "SpecInvalid",
			Status:             metav1.ConditionFalse,
			Reason:             issues[0].Reason,
			Message:            strings.Join(messages, "; "),
			LastTransitionTime: metav1.NewTime(now),
			Severity:           preflightv1alpha1.SeverityCritical,
		}},
	}
}

// specError ist ein interner Tupel-Typ für die Phase-2-Validation.
type specError struct {
	check string
	err   error
}

// buildSpecMap übersetzt die api/v1alpha1.ChecksSpec in eine
// `map[string]domain.CheckSpec`, ruft auf jeder Sub-Spec `Validate`
// und sammelt Fehler. In M3 ist nur `KubernetesVersion` belegt;
// M4 erweitert um die anderen Sibling-Felder.
func buildSpecMap(
	ctx context.Context,
	checks *preflightv1alpha1.ChecksSpec,
) (map[string]domain.CheckSpec, []specError) {
	specs := make(map[string]domain.CheckSpec)
	var errs []specError

	if checks.KubernetesVersion != nil {
		spec := domain.KubernetesVersionSpec{Min: checks.KubernetesVersion.Min}
		if err := spec.Validate(ctx); err != nil {
			errs = append(errs, specError{check: domain.KubernetesVersionSpecKind, err: err})
		} else {
			specs[domain.KubernetesVersionSpecKind] = spec
		}
	}

	if len(errs) > 0 {
		return specs, errs
	}
	return specs, nil
}

func profileWithDefault(p preflightv1alpha1.Profile) string {
	if p == "" {
		return string(preflightv1alpha1.ProfileProduction)
	}
	return string(p)
}

// ErrReconcileTimeout wird platzhalterhaft exportiert, damit M5
// (Robustheit) den Timeout-Pfad explizit testen kann, ohne diesen
// File wieder umzuschreiben. M3 selbst nutzt den Wrap nicht — wenn
// runCtx.Err() != nil ist, kommt der Fehler vom controller-runtime
// als reguläres Reconcile-Failure heraus.
var ErrReconcileTimeout = errors.New("reconcile timeout exceeded")
