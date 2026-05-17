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

// ChecksSpec selects and configures the individual preflight checks. MVP scope
// (LH-PRI-001) only declares kubernetesVersion; further sibling fields (ingress,
// certManager, storage, resources, rbac) are introduced with M3+ slices.
type ChecksSpec struct {
	// KubernetesVersion configures the Kubernetes version check (LH-F-008).
	KubernetesVersion *KubernetesVersionCheckSpec `json:"kubernetesVersion,omitempty"`
}

// OpenDeskPreflightCheckSpec defines the desired state of an OpenDeskPreflightCheck.
type OpenDeskPreflightCheckSpec struct {
	// Profile selects the preset of check defaults to apply.
	//
	// +kubebuilder:default=production
	Profile Profile `json:"profile,omitempty"`

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
