/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package k8s_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/pt9912/k-deskflight/internal/adapter/k8s"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

// sarReactor liefert eine PrependReactor-Funktion, die jede
// SelfSubjectAccessReview-Create-Anfrage mit dem konfigurierten
// `Allowed`-Wert beantwortet. Das echo-Pattern matched, was der
// reale apiserver tut: er füllt das Status-Feld auf der gleichen
// Resource, die der Client einliefert.
func sarReactor(allowed bool, captured **authzv1.SelfSubjectAccessReview) ktesting.ReactionFunc {
	return func(action ktesting.Action) (bool, runtime.Object, error) {
		create, _ := action.(ktesting.CreateAction)
		sar := create.GetObject().(*authzv1.SelfSubjectAccessReview).DeepCopy()
		if captured != nil {
			*captured = sar
		}
		sar.Status.Allowed = allowed
		return true, sar, nil
	}
}

func TestAccessReviewAdapterAllowed(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("create", "selfsubjectaccessreviews", sarReactor(true, nil))

	adapter := k8s.NewAccessReviewAdapter(cs)
	allowed, err := adapter.CanI(context.Background(), domain.PermissionRequest{
		Group: "storage.k8s.io", Resource: "storageclasses", Verb: "list",
	})
	if err != nil {
		t.Fatalf("CanI: unexpected error %v", err)
	}
	if !allowed {
		t.Errorf("CanI: got false, want true")
	}
}

func TestAccessReviewAdapterDenied(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("create", "selfsubjectaccessreviews", sarReactor(false, nil))

	adapter := k8s.NewAccessReviewAdapter(cs)
	allowed, err := adapter.CanI(context.Background(), domain.PermissionRequest{
		Group: "storage.k8s.io", Resource: "storageclasses", Verb: "list",
	})
	if err != nil {
		t.Fatalf("CanI: unexpected error %v", err)
	}
	if allowed {
		t.Errorf("CanI: got true, want false")
	}
}

func TestAccessReviewAdapterSubsystemError(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("create", "selfsubjectaccessreviews",
		func(_ ktesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("auth webhook unreachable")
		})

	adapter := k8s.NewAccessReviewAdapter(cs)
	allowed, err := adapter.CanI(context.Background(), domain.PermissionRequest{
		Group: "storage.k8s.io", Resource: "storageclasses", Verb: "list",
	})
	if err == nil {
		t.Errorf("CanI: expected error, got nil")
	}
	if allowed {
		t.Errorf("CanI: bool must be false on error, got true")
	}
	// Pflicht-Check: die Error-Message nennt den CanonicalString der
	// fehlgeschlagenen Permission, damit Oncall im Log sieht, welcher
	// SAR-Call scheiterte.
	if err != nil && !strings.Contains(err.Error(), "list storage.k8s.io/storageclasses") {
		t.Errorf("error message must include canonical permission; got %q", err.Error())
	}
}

// TestAccessReviewAdapterFieldMapping fixiert slice-M5 §2.1: alle
// PermissionRequest-Felder werden 1:1 auf ResourceAttributes
// übertragen. Drift hier wäre ein Sicherheitsbug — wenn z. B.
// Namespace nicht durchgereicht würde, könnte ein Check fälschlich
// als allowed durchgehen.
func TestAccessReviewAdapterFieldMapping(t *testing.T) {
	var captured *authzv1.SelfSubjectAccessReview
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("create", "selfsubjectaccessreviews", sarReactor(true, &captured))

	adapter := k8s.NewAccessReviewAdapter(cs)
	req := domain.PermissionRequest{
		Group:       "apps",
		Resource:    "deployments",
		Subresource: "scale",
		Verb:        "patch",
		Namespace:   "test-ns",
	}
	if _, err := adapter.CanI(context.Background(), req); err != nil {
		t.Fatalf("CanI: %v", err)
	}
	if captured == nil {
		t.Fatalf("reactor did not capture the SAR")
	}
	attrs := captured.Spec.ResourceAttributes
	if attrs == nil {
		t.Fatalf("ResourceAttributes is nil")
	}
	if attrs.Group != "apps" {
		t.Errorf("Group: got %q, want %q", attrs.Group, "apps")
	}
	if attrs.Resource != "deployments" {
		t.Errorf("Resource: got %q, want %q", attrs.Resource, "deployments")
	}
	if attrs.Subresource != "scale" {
		t.Errorf("Subresource: got %q, want %q", attrs.Subresource, "scale")
	}
	if attrs.Verb != "patch" {
		t.Errorf("Verb: got %q, want %q", attrs.Verb, "patch")
	}
	if attrs.Namespace != "test-ns" {
		t.Errorf("Namespace: got %q, want %q", attrs.Namespace, "test-ns")
	}
}

// TestAccessReviewAdapterCoreGroupMapping verifiziert die
// Core-API-Konvention: Group="" wird unverändert übertragen.
func TestAccessReviewAdapterCoreGroupMapping(t *testing.T) {
	var captured *authzv1.SelfSubjectAccessReview
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("create", "selfsubjectaccessreviews", sarReactor(true, &captured))

	adapter := k8s.NewAccessReviewAdapter(cs)
	req := domain.PermissionRequest{Group: "", Resource: "nodes", Verb: "list"}
	if _, err := adapter.CanI(context.Background(), req); err != nil {
		t.Fatalf("CanI: %v", err)
	}
	if captured.Spec.ResourceAttributes.Group != "" {
		t.Errorf("Group: got %q, want empty (core API)", captured.Spec.ResourceAttributes.Group)
	}
}
