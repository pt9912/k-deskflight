/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package domain_test

import (
	"testing"

	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

func TestPermissionRequestCanonicalString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		req  domain.PermissionRequest
		want string
	}{
		{
			"core list nodes",
			domain.PermissionRequest{Verb: "list", Resource: "nodes"},
			"list core/nodes",
		},
		{
			"storage list storageclasses",
			domain.PermissionRequest{Group: "storage.k8s.io", Resource: "storageclasses", Verb: "list"},
			"list storage.k8s.io/storageclasses",
		},
		{
			"networking list ingressclasses",
			domain.PermissionRequest{Group: "networking.k8s.io", Resource: "ingressclasses", Verb: "list"},
			"list networking.k8s.io/ingressclasses",
		},
		{
			"with subresource",
			domain.PermissionRequest{Group: "apps", Resource: "deployments", Subresource: "status", Verb: "get"},
			"get apps/deployments/status",
		},
		{
			"namespaced",
			domain.PermissionRequest{Group: "", Resource: "pods", Verb: "list", Namespace: "kube-system"},
			"list core/pods@kube-system",
		},
		{
			"subresource and namespace",
			domain.PermissionRequest{Group: "apps", Resource: "deployments", Subresource: "scale", Verb: "patch", Namespace: "ns-1"},
			"patch apps/deployments/scale@ns-1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.req.CanonicalString(); got != tc.want {
				t.Errorf("CanonicalString(): got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestPermissionRequestCanonicalStringStableAsMapKey deckt die
// Cache-Use-Case aus slice-M5 §2.3: zwei gleichwertige Requests
// produzieren denselben Schlüssel.
func TestPermissionRequestCanonicalStringStableAsMapKey(t *testing.T) {
	t.Parallel()
	a := domain.PermissionRequest{Group: "storage.k8s.io", Resource: "storageclasses", Verb: "list"}
	b := domain.PermissionRequest{Verb: "list", Resource: "storageclasses", Group: "storage.k8s.io"}
	if a.CanonicalString() != b.CanonicalString() {
		t.Errorf("CanonicalString unstable across field-order: %q vs %q",
			a.CanonicalString(), b.CanonicalString())
	}
}
