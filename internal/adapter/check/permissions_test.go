/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package check_test

import (
	"reflect"
	"testing"

	"github.com/pt9912/k-deskflight/internal/adapter/check"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

// TestRequiredPermissionsPerCheck fixiert slice-M5 §2.2: jeder
// MVP-Check liefert genau die in der Plan-Tabelle definierten Rechte.
// Drift hier ist Hinweis darauf, dass die `+kubebuilder:rbac:`-Marker
// am Reconciler ebenfalls angepasst werden müssen (Konsistenz-Test in
// `internal/hexagon/application/rbac_consistency_test.go` erzwingt
// das automatisch beim Schritt 5).
func TestRequiredPermissionsPerCheck(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		got  []domain.PermissionRequest
		want []domain.PermissionRequest
	}{
		{
			name: "kubernetesVersion — discovery-only, leer",
			got:  check.NewKubernetesVersion(nil, nil).RequiredPermissions(),
			want: nil,
		},
		{
			name: "certManager — discovery-only, leer",
			got:  check.NewCertManager(nil, nil).RequiredPermissions(),
			want: nil,
		},
		{
			name: "storageClass — list storage.k8s.io/storageclasses",
			got:  check.NewStorageClass(nil, nil).RequiredPermissions(),
			want: []domain.PermissionRequest{
				{Group: "storage.k8s.io", Resource: "storageclasses", Verb: "list"},
			},
		},
		{
			name: "ingressClass — list networking.k8s.io/ingressclasses",
			got:  check.NewIngressClass(nil, nil).RequiredPermissions(),
			want: []domain.PermissionRequest{
				{Group: "networking.k8s.io", Resource: "ingressclasses", Verb: "list"},
			},
		},
		{
			name: "clusterResources — list core/nodes",
			got:  check.NewClusterResources(nil, nil).RequiredPermissions(),
			want: []domain.PermissionRequest{
				{Group: "", Resource: "nodes", Verb: "list"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !reflect.DeepEqual(tc.got, tc.want) {
				t.Errorf("RequiredPermissions: got %+v, want %+v", tc.got, tc.want)
			}
		})
	}
}
