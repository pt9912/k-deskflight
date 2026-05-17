/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

// Package v1alpha1 contains the API schema definition for the
// k-deskflight.geo-terrain.net/v1alpha1 group (architecture.md AR-006,
// ADR 0006).
//
// +kubebuilder:object:generate=true
// +groupName=k-deskflight.geo-terrain.net
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is the API group/version this package describes.
	GroupVersion = schema.GroupVersion{
		Group:   "k-deskflight.geo-terrain.net",
		Version: "v1alpha1",
	}

	// SchemeBuilder collects the addToScheme functions for this group/version.
	// Uses apimachinery's runtime.SchemeBuilder (not the deprecated
	// controller-runtime helper) to keep this api/ package light on transitive
	// imports — staticcheck SA1019 / kubernetes deprecation note.
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme registers this group's types into a runtime.Scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&OpenDeskPreflightCheck{},
		&OpenDeskPreflightCheckList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
