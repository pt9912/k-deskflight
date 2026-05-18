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
	"log/slog"
	"runtime/debug"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	preflightv1alpha1 "github.com/pt9912/k-deskflight/api/v1alpha1"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

// Reconciler-Konstanten für Outer-Recover-Pfade (slice-M5 §2.4).
const (
	conditionTypeReconcileError = "ReconcileError"
	reasonReconcilePanic        = "ReconcilePanic"
)

// defaultReconcileTimeout deckelt einen Reconcile-Lauf
// (architecture.md AR-009 Phase 1, M3-minimal). M5 ergänzt
// Cross-Constraint-Auswertung gegen CHECK_TIMEOUT_SECONDS und
// OPERATOR_STRICT_CONFIG.
const defaultReconcileTimeout = 120 * time.Second

// Reconciler reconciles OpenDeskPreflightCheck resources
// (architecture.md AR-009). M3 implementiert die Phasen 1+3+4
// (sequenziell, ohne Worker-Pool) + 5 + 6. Mit M5 kommen
// Per-Check-Timeout, SAR-Pre-Execution, Per-Check- und Outer-Recover
// sowie Secret-Sanitize-Hooks dazu. Voller AR-009-Pfad mit
// Worker-Pool + OPERATOR_STRICT_CONFIG bleibt v0.2.
type Reconciler struct {
	client.Client

	Scheme   *runtime.Scheme
	Registry port.CheckRegistry

	// AccessReviewer prüft Cluster-Rechte pro Check vor dem Run
	// (slice-M5 §2.3). Nil-safe für Tests, die keine Permissions
	// auf ihren Stubs deklarieren — wenn ein Check `RequiredPermissions`
	// liefert, MUSS `AccessReviewer` gesetzt sein (sonst panickt der
	// CanI-Call und wird vom Per-Check-Recover als InternalError
	// klassifiziert).
	AccessReviewer port.AccessReviewer

	// Logger ist der strukturierte Logger für Reconcile-Diagnose,
	// SAR-Fehler, Panic-Traces und Per-Result-Summary (slice-M5 §2.6).
	// Nil fällt auf `slog.Default()` zurück, damit M3/M4-Tests keine
	// Setup-Pflicht haben.
	Logger *slog.Logger

	// CheckTimeout deckelt einen einzelnen Check-Lauf (slice-M5 §2.5).
	// `0` fällt auf den 30 s-Default (`runner.go.defaultCheckTimeout`).
	// Tests, die den Per-Check-Timeout-Pfad ausüben, setzen einen
	// kürzeren Wert.
	CheckTimeout time.Duration

	Now func() time.Time
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
// Outer-defer-recover am Anfang (slice-M5 §2.4): jeder Panic, der
// vom Per-Check-Recover in `runner.go` nicht abgefangen wurde (z. B.
// in `buildSpecMap`, im Aggregator, in `writeStatus`), landet hier
// und endet als Phase=Unknown + Condition=ReconcileError + Reason=
// ReconcilePanic, sofern das CR-Objekt schon geladen ist. Andernfalls
// wird der Panic nur geloggt und der Reconcile mit Fehler beendet
// (controller-runtime re-queued).
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, retErr error) {
	runCtx, cancel := context.WithTimeout(ctx, defaultReconcileTimeout)
	defer cancel()

	var cr preflightv1alpha1.OpenDeskPreflightCheck
	var crLoaded bool

	defer func() {
		if rec := recover(); rec != nil {
			result = ctrl.Result{}
			retErr = r.handleReconcilePanic(ctx, runCtx, req, &cr, crLoaded, rec)
		}
	}()

	if err := r.Get(runCtx, req.NamespacedName, &cr); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, fmt.Errorf("get OpenDeskPreflightCheck: %w", err)
	}
	crLoaded = true

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

	// Per-Result-Diagnose-Log (slice-M5 §2.6, Folge-Review-Befund 3):
	// jeder Result-Eintrag wandert über LogResult, damit beide
	// Sanitize-Hooks (Message + Attrs) anliegen.
	for _, res := range results {
		LogResult(runCtx, r.logger(), slog.LevelInfo, "check result",
			res,
			slog.String("name", req.Name),
			slog.String("namespace", req.Namespace),
		)
	}

	r.logger().LogAttrs(runCtx, slog.LevelInfo, "reconciled",
		SanitizeAttrs(
			slog.String("name", req.Name),
			slog.String("namespace", req.Namespace),
			slog.String("phase", string(out.Phase)),
			slog.Int64("generation", cr.Generation),
			slog.Int("checksTotal", int(out.Summary.ChecksTotal)),
			slog.Int("passed", int(out.Summary.Passed)),
			slog.Int("failed", int(out.Summary.Failed)),
			slog.Int("warning", int(out.Summary.Warning)),
			slog.Int("unknown", int(out.Summary.Unknown)),
		)...,
	)

	return ctrl.Result{}, r.writeStatus(runCtx, &cr, out)
}

// SetupWithManager registers the reconciler with the controller-runtime manager.
// Vor dem Manager-Setup läuft `Validate`, damit Wiring-Lücken
// (z. B. fehlender AccessReviewer für SAR-pflichtige Checks) als
// klarer Startup-Fehler erscheinen — nicht als Per-Check-InternalError
// zur Smoke-Laufzeit (slice-M5 Review-Befund 5; CI-Hotfix-Vorfall
// Schritt 4 ↔ 6).
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.Validate(); err != nil {
		return fmt.Errorf("reconciler config invalid: %w", err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&preflightv1alpha1.OpenDeskPreflightCheck{}).
		Named("opendeskpreflightcheck").
		Complete(r)
}

// Validate prüft Wiring-Konsistenz vor dem Manager-Setup (slice-M5
// Review-Befund 5). Aktuell deckt sie die SAR-Wiring-Lücke ab
// (Check.RequiredPermissions ohne AccessReviewer); v0.2 ergänzt
// Worker-Pool-Config, Metrics-Hook etc. Test-Pflicht-Stelle.
func (r *Reconciler) Validate() error {
	if r.Registry == nil {
		return errors.New("registry is nil")
	}
	if r.AccessReviewer != nil {
		return nil
	}
	for _, c := range r.Registry.All() {
		if len(c.RequiredPermissions()) > 0 {
			return fmt.Errorf("check %q declares RequiredPermissions but AccessReviewer is nil — wiring incomplete", c.Name())
		}
	}
	return nil
}

func (r *Reconciler) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

// logger returns the configured slog logger or the default; nil-safe
// for tests, die keinen Logger inject (slice-M5 §2.6).
func (r *Reconciler) logger() *slog.Logger {
	if r.Logger != nil {
		return r.Logger
	}
	return slog.Default()
}

// handleReconcilePanic ist der Outer-Recover-Pfad aus `Reconcile`
// (slice-M5 §2.4). Loggt den Panic, baut den Return-Wert, und
// schreibt best-effort einen `Phase=Unknown / Reason=ReconcilePanic`-
// Status — mit innerem Recover, der bei Sekundär-Panic im
// writeStatus-Pfad **nicht stumm** ist, sondern loggt (slice-M5
// Review-Befund 2).
func (r *Reconciler) handleReconcilePanic(
	ctx, runCtx context.Context,
	req ctrl.Request,
	cr *preflightv1alpha1.OpenDeskPreflightCheck,
	crLoaded bool,
	rec interface{},
) error {
	r.logger().LogAttrs(ctx, slog.LevelError, "panic recovered in Reconcile",
		SanitizeAttrs(
			slog.String("name", req.Name),
			slog.String("namespace", req.Namespace),
			slog.Any("recover", rec),
			slog.String("stack", string(debug.Stack())),
		)...,
	)
	if crLoaded {
		func() {
			defer func() {
				if innerRec := recover(); innerRec != nil {
					r.logger().LogAttrs(ctx, slog.LevelError, "panic in panic-recovery status write",
						SanitizeAttrs(
							slog.String("name", req.Name),
							slog.String("namespace", req.Namespace),
							slog.Any("inner_recover", innerRec),
							slog.String("inner_stack", string(debug.Stack())),
						)...,
					)
				}
			}()
			_ = r.writeStatus(runCtx, cr, r.reconcilePanicOutput(rec))
		}()
	}
	return fmt.Errorf("reconcile panic: %v", rec)
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

// runChecks orchestriert die Check-Pipeline via `RunCheckSafely`
// (slice-M5 §2.3, §2.4, §2.5): SAR-Pre-Execution, Per-Check-Recover,
// Per-Check-Timeout. Ein Run-lokaler `PermissionCache` dedupliziert
// SAR-Calls; mehrere Checks mit derselben Permission teilen sich das
// Ergebnis.
func (r *Reconciler) runChecks(
	ctx context.Context,
	active []domain.Check,
	specs map[string]domain.CheckSpec,
) []domain.Result {
	cache := NewPermissionCache()
	results := make([]domain.Result, 0, len(active))
	for _, check := range active {
		spec, ok := specs[check.Name()]
		if !ok {
			continue
		}
		res := RunCheckSafely(ctx, r.logger(), r.AccessReviewer, cache, check, spec, r.now, r.CheckTimeout)
		results = append(results, res)
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
//
// **Sanitize-Pflicht** (slice-M5 §2.6): vor jedem `Status().Update`
// läuft `SanitizeMessage` über alle Condition-Message-Felder. In M5
// ist das die Identitätsfunktion; das Aufruf-Pattern verankert die
// Stelle für v0.2-Pattern-Maskierung.
func (r *Reconciler) writeStatus(
	ctx context.Context,
	cr *preflightv1alpha1.OpenDeskPreflightCheck,
	out AggregationOutput,
) error {
	cr.Status.Phase = out.Phase
	cr.Status.ObservedGeneration = cr.Generation
	cr.Status.Summary = out.Summary
	cr.Status.Conditions = sanitizeConditions(out.Conditions)

	if err := r.Status().Update(ctx, cr); err != nil {
		return fmt.Errorf("update OpenDeskPreflightCheck status: %w", err)
	}
	return nil
}

// sanitizeConditions wendet SanitizeMessage auf jedes Condition.Message
// an (slice-M5 §2.6). Returnt eine neue Slice, damit der Aufrufer das
// Original nicht versehentlich teilt.
func sanitizeConditions(in []preflightv1alpha1.Condition) []preflightv1alpha1.Condition {
	if len(in) == 0 {
		return in
	}
	out := make([]preflightv1alpha1.Condition, len(in))
	copy(out, in)
	for i := range out {
		out[i].Message = SanitizeMessage(out[i].Message)
	}
	return out
}

// reconcilePanicOutput baut das AggregationOutput für den Outer-Recover-
// Pfad (slice-M5 §2.4): Phase=Unknown, Condition=ReconcileError mit
// Reason=ReconcilePanic. Die Panic-Detail-Message wird über
// `SanitizeMessage` geführt (slice-M5 Review-Befund 3), damit
// eventuelle Variablenwerte aus dem panickenden Frame nicht
// ungefiltert im Status landen (LH-SEC-002). In M5 ist
// `SanitizeMessage` Identität; v0.2 ersetzt das durch echte
// Pattern-Maskierung, deshalb muss der Hook hier verankert sein.
func (r *Reconciler) reconcilePanicOutput(panicVal interface{}) AggregationOutput {
	now := r.now().UTC()
	return AggregationOutput{
		Phase: preflightv1alpha1.PhaseUnknown,
		Summary: preflightv1alpha1.Summary{
			ChecksTotal: 0,
			LastChecked: ptrMetaTime(metav1.NewTime(now)),
		},
		Conditions: []preflightv1alpha1.Condition{{
			Type:               conditionTypeReconcileError,
			Status:             metav1.ConditionFalse,
			Reason:             reasonReconcilePanic,
			Message:            SanitizeMessage(fmt.Sprintf("reconcile panic recovered: %v", panicVal)),
			LastTransitionTime: metav1.NewTime(now),
			Severity:           preflightv1alpha1.SeverityCritical,
		}},
	}
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
