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

	// AR-009 / Roadmap §3 M2: sichtbare Phasen-Transition Pending →
	// Running → Final. Pending signalisiert „Controller hat den CR für
	// diese Generation gesehen"; Running markiert die Check-Execution-
	// Phase; Final ist das Aggregat. Drei sequenzielle Status-Updates
	// sind günstig (lokale Generation bleibt, controller-runtime liefert
	// die neue resourceVersion direkt in das Objekt zurück).
	if err := r.markPhase(runCtx, &cr, preflightv1alpha1.PhasePending); err != nil {
		return ctrl.Result{}, err
	}

	profile := profileWithDefault(cr.Spec.Profile)
	defaults := defaultsForProfile(profile)

	specs, validationErrs := buildSpecMap(runCtx, &cr.Spec.Checks, defaults)
	if len(validationErrs) > 0 {
		return ctrl.Result{}, r.writeStatus(runCtx, &cr, r.specInvalidOutput(validationErrs))
	}

	active, issues := r.Registry.ListByProfile(profile, specs)
	if len(issues) > 0 {
		return ctrl.Result{}, r.writeStatus(runCtx, &cr, r.selectionIssueOutput(issues, len(specs)))
	}

	if err := r.markPhase(runCtx, &cr, preflightv1alpha1.PhaseRunning); err != nil {
		return ctrl.Result{}, err
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

// isAlreadyReconciled überspringt den Reconcile, wenn die aktuelle
// CR-Generation bereits in einer **terminalen** Phase liegt — das
// schließt nicht nur Passed ein, sondern auch Failed/Warning/Unknown.
// Andernfalls würde controller-runtime auf jeden Resync (Default
// 10h-Cache-Resync, manuelle Trigger) ein erneutes Reconcile auslösen
// und mit neuen LastTransition-Zeitstempeln Status-Events erzeugen,
// obwohl sich am tatsächlichen Zustand nichts geändert hat. Echter
// Re-Run pro Intervall kommt mit AR-010 in M5.
//
// Pending und Running sind bewusst NICHT terminal — wenn ein Reconcile
// nach Pending-Write abstürzt, soll der nächste Versuch sauber neu
// laufen.
func (r *Reconciler) isAlreadyReconciled(cr *preflightv1alpha1.OpenDeskPreflightCheck) bool {
	if cr.Status.ObservedGeneration != cr.Generation {
		return false
	}
	switch cr.Status.Phase {
	case preflightv1alpha1.PhasePassed,
		preflightv1alpha1.PhaseWarning,
		preflightv1alpha1.PhaseFailed,
		preflightv1alpha1.PhaseUnknown:
		return true
	}
	return false
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

// markPhase schreibt eine Zwischen-Phase (Pending/Running) ohne die
// Conditions oder Summary anzufassen. Beobachter sehen damit den
// Reconcile-Fortschritt; die ObservedGeneration wird erst beim
// finalen `writeStatus` gehoben, damit ein abgebrochener Reconcile-
// Zyklus auf Pending/Running-Stand nicht versehentlich als „diese
// Generation ist fertig" interpretiert wird.
func (r *Reconciler) markPhase(
	ctx context.Context,
	cr *preflightv1alpha1.OpenDeskPreflightCheck,
	phase preflightv1alpha1.Phase,
) error {
	if cr.Status.Phase == phase {
		return nil
	}
	cr.Status.Phase = phase
	if err := r.Status().Update(ctx, cr); err != nil {
		return fmt.Errorf("mark phase %q: %w", phase, err)
	}
	return nil
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

// profileDefaults bündelt die code-seitigen Defaults, die pro Profile
// in `buildSpecMap` greifen, wenn der Anwender keine expliziten Werte
// gesetzt hat (slice-M4 §2.3). Aktuell nur für ClusterResources
// relevant; Storage/Ingress haben keine sinnvollen Profile-Defaults
// (Existenz ist binär, kein Floor möglich), cert-manager ist
// parameterlos.
type profileDefaults struct {
	clusterResourcesMinCPU    string
	clusterResourcesMinMemory string
}

// defaultsForProfile mappt den Profile-String auf die zugehörigen
// Code-Defaults. Unbekannte Profile-Werte fallen auf Production zurück
// — das CRD-OpenAPI-Enum (`production|evaluation`) verhindert das
// normalerweise, aber wir behalten den defensiven Pfad.
func defaultsForProfile(profile string) profileDefaults {
	switch profile {
	case string(preflightv1alpha1.ProfileEvaluation):
		return profileDefaults{
			clusterResourcesMinCPU:    domain.DefaultClusterResourcesMinCPUEvaluation,
			clusterResourcesMinMemory: domain.DefaultClusterResourcesMinMemoryEvaluation,
		}
	default:
		return profileDefaults{
			clusterResourcesMinCPU:    domain.DefaultClusterResourcesMinCPUProduction,
			clusterResourcesMinMemory: domain.DefaultClusterResourcesMinMemoryProduction,
		}
	}
}

// buildSpecMap übersetzt die api/v1alpha1.ChecksSpec in eine
// `map[string]domain.CheckSpec`, ruft auf jeder Sub-Spec `Validate`
// und sammelt Fehler.
//
// **Default-Aktivierung pro Check** (slice-M4 §2.2):
//
//   - KubernetesVersion: immer aktiv (ADR 0009 §2.2 — `DefaultKubernetesVersionMin`).
//   - StorageClass: nur bei explizitem Sub-Spec.
//   - IngressClass: nur bei explizitem Sub-Spec.
//   - CertManager: immer aktiv (parameterlos, reine Existence-Prüfung).
//   - ClusterResources: immer aktiv mit Profile-Defaults aus `defaults`
//     (slice-M4 §2.3 — production 4 CPU/8Gi, evaluation 2 CPU/4Gi).
//
// Explizite User-Werte überschreiben Profile-Defaults feldweise; ein
// leeres Sub-Spec `clusterResources: {}` läuft also nach wie vor mit den
// Profile-Defaults durch. Die CRD-Schema-Defaults aus
// `api/v1alpha1/opendeskpreflightcheck_types.go` greifen nur für
// KubernetesVersion.Min (`"1.34"`), nicht für die M4-Felder.
func buildSpecMap(
	ctx context.Context,
	checks *preflightv1alpha1.ChecksSpec,
	defaults profileDefaults,
) (map[string]domain.CheckSpec, []specError) {
	specs := make(map[string]domain.CheckSpec)
	var errs []specError

	add := func(kind string, spec domain.CheckSpec) {
		if err := spec.Validate(ctx); err != nil {
			errs = append(errs, specError{check: kind, err: err})
			return
		}
		specs[kind] = spec
	}

	versionSpec := domain.KubernetesVersionSpec{Min: domain.DefaultKubernetesVersionMin}
	if checks.KubernetesVersion != nil && checks.KubernetesVersion.Min != "" {
		versionSpec.Min = checks.KubernetesVersion.Min
	}
	add(domain.KubernetesVersionSpecKind, versionSpec)

	if checks.StorageClass != nil {
		add(domain.StorageClassSpecKind, domain.StorageClassSpec{
			Names:          append([]string(nil), checks.StorageClass.Names...),
			RequireDefault: checks.StorageClass.RequireDefault,
		})
	}

	if checks.IngressClass != nil {
		add(domain.IngressClassSpecKind, domain.IngressClassSpec{
			Names: append([]string(nil), checks.IngressClass.Names...),
		})
	}

	add(domain.CertManagerSpecKind, domain.CertManagerSpec{})

	resourcesSpec := domain.ClusterResourcesSpec{
		MinCPU:    defaults.clusterResourcesMinCPU,
		MinMemory: defaults.clusterResourcesMinMemory,
	}
	if checks.ClusterResources != nil {
		if v := checks.ClusterResources.MinCPU; v != "" {
			resourcesSpec.MinCPU = v
		}
		if v := checks.ClusterResources.MinMemory; v != "" {
			resourcesSpec.MinMemory = v
		}
	}
	add(domain.ClusterResourcesSpecKind, resourcesSpec)

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
