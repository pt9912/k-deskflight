/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package k8s_test

import (
	"context"
	"errors"
	"testing"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/pt9912/k-deskflight/internal/adapter/k8s"
)

func TestStorageClassAdapterListStorageClasses(t *testing.T) {
	cs := fake.NewSimpleClientset(
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "standard",
				Annotations: map[string]string{
					"storageclass.kubernetes.io/is-default-class": "true",
				},
			},
			Provisioner: "rancher.io/local-path",
		},
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{Name: "fast"},
			Provisioner: "ebs.csi.aws.com",
		},
		&storagev1.StorageClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "legacy-default",
				Annotations: map[string]string{
					"storageclass.beta.kubernetes.io/is-default-class": "true",
				},
			},
		},
	)

	adapter := k8s.NewStorageClassAdapter(cs)
	infos, err := adapter.ListStorageClasses(context.Background())
	if err != nil {
		t.Fatalf("ListStorageClasses: %v", err)
	}

	if got, want := len(infos), 3; got != want {
		t.Fatalf("count: got %d, want %d", got, want)
	}

	byName := make(map[string]bool, len(infos))
	for _, info := range infos {
		byName[info.Name] = info.IsDefault
	}

	if !byName["standard"] {
		t.Errorf("standard: expected IsDefault=true (GA annotation)")
	}
	if !byName["legacy-default"] {
		t.Errorf("legacy-default: expected IsDefault=true (legacy annotation)")
	}
	if byName["fast"] {
		t.Errorf("fast: expected IsDefault=false (no annotation)")
	}
}

func TestStorageClassAdapterListError(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("list", "storageclasses", func(_ ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("forbidden")
	})

	adapter := k8s.NewStorageClassAdapter(cs)
	if _, err := adapter.ListStorageClasses(context.Background()); err == nil {
		t.Errorf("expected error, got nil")
	}
}
