/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Profile selects the preset of check defaults to apply
// (architecture.md AR-006, LH-PROF-002/003, ADR 0009 §2.2).
//
// +kubebuilder:validation:Enum=production;evaluation
type Profile string

const (
	// ProfileProduction is the default profile — strict thresholds suitable for productive clusters.
	ProfileProduction Profile = "production"

	// ProfileEvaluation relaxes thresholds for evaluation/test clusters.
	ProfileEvaluation Profile = "evaluation"
)

// Phase is the overall execution state of a preflight reconcile run
// (architecture.md AR-006, LH-F-006).
//
// +kubebuilder:validation:Enum=Pending;Running;Passed;Warning;Failed;Unknown
type Phase string

const (
	// PhasePending is the initial state before any check has been started.
	PhasePending Phase = "Pending"

	// PhaseRunning indicates that checks are currently being executed.
	PhaseRunning Phase = "Running"

	// PhasePassed indicates all checks completed successfully.
	PhasePassed Phase = "Passed"

	// PhaseWarning indicates checks completed but at least one returned a warning.
	PhaseWarning Phase = "Warning"

	// PhaseFailed indicates at least one check returned a critical failure.
	PhaseFailed Phase = "Failed"

	// PhaseUnknown indicates the reconcile could not determine an outcome.
	PhaseUnknown Phase = "Unknown"
)

// Severity represents the operational severity of a single check result
// (architecture.md AR-006/AR-014).
//
// +kubebuilder:validation:Enum=info;warning;critical
type Severity string

const (
	// SeverityInfo is the default severity for purely informational conditions.
	SeverityInfo Severity = "info"

	// SeverityWarning marks a recoverable or advisory finding.
	SeverityWarning Severity = "warning"

	// SeverityCritical marks a finding that must be addressed before installation.
	SeverityCritical Severity = "critical"
)

// KubernetesVersionCheckSpec parameterises the kubernetesVersion check (LH-F-008).
type KubernetesVersionCheckSpec struct {
	// Min is the minimum acceptable Kubernetes server version.
	// Default follows ADR 0009 §2.2 (production profile floor).
	//
	// +kubebuilder:default="1.34"
	// +kubebuilder:validation:Pattern=`^[0-9]+\.[0-9]+(\.[0-9]+)?$`
	Min string `json:"min,omitempty"`
}

// StorageClassCheckSpec parameterises the storageClass check (LH-F-010 / LH-F-011).
type StorageClassCheckSpec struct {
	// Names lists the StorageClass names that must be present in the cluster.
	// At least one entry is required when this sub-spec is set; the reconciler
	// activates the check only when the sub-spec is non-nil (no profile default,
	// slice-M4 §2.2).
	//
	// +kubebuilder:validation:MinItems=1
	// +optional
	Names []string `json:"names,omitempty"`

	// RequireDefault demands that a Default-marked StorageClass exists. The
	// adapter tolerates both the GA annotation (storageclass.kubernetes.io/
	// is-default-class) and the legacy beta key (slice-M4 §9).
	//
	// +optional
	RequireDefault bool `json:"requireDefault,omitempty"`
}

// IngressClassCheckSpec parameterises the ingressClass check (LH-F-012).
type IngressClassCheckSpec struct {
	// Names lists the IngressClass names that must be present in the cluster.
	// At least one entry is required when this sub-spec is set.
	//
	// +kubebuilder:validation:MinItems=1
	Names []string `json:"names,omitempty"`
}

// CertManagerCheckSpec parameterises the cert-manager existence check (LH-F-013).
// Currently parameter-less — the check verifies registration of the
// cert-manager.io API group only. ClusterIssuer detail validation (LH-F-014)
// is deferred to v0.2 (ADR 0010). Note that a missing cert-manager produces
// Severity=warning, not critical (slice-M4 §9, bridged in the check Message).
type CertManagerCheckSpec struct{}

// ClusterResourcesCheckSpec parameterises the clusterResources check (LH-F-015 /
// LH-AK-009). When this sub-spec is omitted the reconciler applies profile-
// based defaults (slice-M4 §2.3): production 4 CPU / 8Gi, evaluation 2 CPU / 4Gi.
// Explicit values override the profile defaults field-by-field.
type ClusterResourcesCheckSpec struct {
	// MinCPU is the minimum allocatable CPU as a Kubernetes resource.Quantity
	// string (e.g. "4", "500m", "2.5").
	//
	// +kubebuilder:validation:Pattern=`^[0-9]+(\.[0-9]+)?(m|Ki|Mi|Gi|Ti|Pi|Ei|K|M|G|T|P|E)?$`
	// +optional
	MinCPU string `json:"minCPU,omitempty"`

	// MinMemory is the minimum allocatable memory as a Kubernetes resource.Quantity
	// string (e.g. "8Gi", "2048Mi", "1G").
	//
	// +kubebuilder:validation:Pattern=`^[0-9]+(\.[0-9]+)?(m|Ki|Mi|Gi|Ti|Pi|Ei|K|M|G|T|P|E)?$`
	// +optional
	MinMemory string `json:"minMemory,omitempty"`
}

// ChecksSpec selects and configures the individual preflight checks. The M4
// slice closes the MVP-Pflichtset (LH-PRI-001): KubernetesVersion (M3) plus
// StorageClass, IngressClass, CertManager, and ClusterResources. Profile-based
// defaults are applied code-side in the reconciler, not via CRD defaults.
type ChecksSpec struct {
	// KubernetesVersion configures the Kubernetes version check (LH-F-008).
	KubernetesVersion *KubernetesVersionCheckSpec `json:"kubernetesVersion,omitempty"`

	// StorageClass configures the StorageClass-presence check (LH-F-010 / LH-F-011).
	StorageClass *StorageClassCheckSpec `json:"storageClass,omitempty"`

	// IngressClass configures the IngressClass-presence check (LH-F-012).
	IngressClass *IngressClassCheckSpec `json:"ingressClass,omitempty"`

	// CertManager configures the cert-manager existence check (LH-F-013).
	CertManager *CertManagerCheckSpec `json:"certManager,omitempty"`

	// ClusterResources configures the cluster-resources check (LH-F-015 / LH-AK-009).
	ClusterResources *ClusterResourcesCheckSpec `json:"clusterResources,omitempty"`
}

// OpenDeskPreflightCheckSpec defines the desired state of an OpenDeskPreflightCheck.
type OpenDeskPreflightCheckSpec struct {
	// Profile selects the preset of check defaults to apply.
	//
	// +kubebuilder:default=production
	Profile Profile `json:"profile,omitempty"`

	// Interval steuert das Wiederholintervall des Reconciles als
	// `time.ParseDuration`-String (z. B. `5m`, `30s`, `1h30m`).
	// Default `5m`, Bounds `[30s, 24h]`. **Bewusst ohne
	// `+kubebuilder:validation:Pattern`** (slice-M6 §2.3.1): AR-010.1
	// verlangt liveness-sicheres CR-Spec-Scope-Verhalten — ungültige
	// Werte sollen den CRD-Validator passieren und im Reconciler-
	// Normalisierer (`application.NormalizeInterval`) auf einen
	// erlaubten Wert geklemmt werden (mit
	// `Status.Conditions[ConfigurationInvalid]=True`,
	// Reason=`IntervalNormalized`, Severity=`warning`).
	//
	// Plain `string` (kein Pointer) — `""` und „nicht gesetzt" werden
	// vom Normalisierer identisch behandelt, daher kein semantischer
	// Bedarf für `*string` (slice-M6 §4 Step-1-Review-Fixup Befund 5).
	//
	// +kubebuilder:default="5m"
	// +optional
	Interval string `json:"interval,omitempty"`

	// Checks selects and configures the individual preflight checks.
	Checks ChecksSpec `json:"checks,omitempty"`
}

// Condition is a Kubernetes-style status entry extended with an explicit
// Severity field (architecture.md AR-006). Severity drives the aggregation
// rules in AR-014.
type Condition struct {
	// Type is a CamelCase name identifying the condition (e.g. KubernetesVersionReady).
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Type string `json:"type"`

	// Status is True, False, or Unknown.
	//
	// +kubebuilder:validation:Required
	Status metav1.ConditionStatus `json:"status"`

	// Reason is a programmatic identifier indicating the cause of the condition.
	//
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Reason string `json:"reason"`

	// Message is a human-readable explanation of the condition.
	Message string `json:"message,omitempty"`

	// LastTransitionTime is the last time the condition transitioned.
	//
	// +kubebuilder:validation:Required
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// Severity is the operational severity associated with the condition.
	// Default `info` per AR-006 — producers should set warning/critical
	// when applicable for AR-014 aggregation.
	//
	// +kubebuilder:default=info
	Severity Severity `json:"severity,omitempty"`
}

// Summary aggregates per-result counts for a single reconcile cycle (LH-F-007).
type Summary struct {
	// Passed counts checks that completed successfully.
	Passed int32 `json:"passed"`

	// Warning counts checks that completed with a non-fatal warning.
	Warning int32 `json:"warning"`

	// Failed counts checks that completed with a critical failure.
	Failed int32 `json:"failed"`

	// Unknown counts checks whose outcome could not be determined.
	Unknown int32 `json:"unknown"`

	// ChecksTotal is the number of active checks for this reconcile run.
	ChecksTotal int32 `json:"checksTotal"`

	// LastChecked is the timestamp of the last reconcile run that populated this Summary.
	LastChecked *metav1.Time `json:"lastChecked,omitempty"`
}

// OpenDeskPreflightCheckStatus captures the observed state of a preflight run.
type OpenDeskPreflightCheckStatus struct {
	// Phase is the overall reconcile phase (LH-F-006).
	Phase Phase `json:"phase,omitempty"`

	// ObservedGeneration is the metadata.generation that the controller has
	// most recently fully reconciled (architecture.md AR-006).
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Summary aggregates per-result counts (LH-F-007).
	Summary Summary `json:"summary,omitempty"`

	// Conditions detail per-check outcomes (LH-F-005).
	//
	// +listType=map
	// +listMapKey=type
	Conditions []Condition `json:"conditions,omitempty"`
}

// OpenDeskPreflightCheck is the Schema for the opendeskpreflightchecks API (LH-F-001).
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=opdc
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type OpenDeskPreflightCheck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenDeskPreflightCheckSpec   `json:"spec,omitempty"`
	Status OpenDeskPreflightCheckStatus `json:"status,omitempty"`
}

// OpenDeskPreflightCheckList contains a list of OpenDeskPreflightCheck.
//
// +kubebuilder:object:root=true
type OpenDeskPreflightCheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenDeskPreflightCheck `json:"items"`
}

// Scheme-Registrierung passiert zentral in groupversion_info.go via
// `addKnownTypes` — wir haben hier bewusst kein `init()`, weil die
// apimachinery-`NewSchemeBuilder`-Konstruktion alle Typen-Registrierungen
// an einer Stelle bündelt (gochecknoinits-konform).
