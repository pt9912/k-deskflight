/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package k8s_test

import (
	"context"
	"errors"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/pt9912/k-deskflight/internal/adapter/k8s"
)

func newFakeDiscovery(resources []*metav1.APIResourceList) *fakediscovery.FakeDiscovery {
	// Resources lebt auf der embedded *testing.Fake — nicht direkt
	// auf FakeDiscovery (client-go v0.36.0).
	return &fakediscovery.FakeDiscovery{
		Fake: &ktesting.Fake{Resources: resources},
	}
}

func TestAPIGroupAdapterHasAPIGroupPresent(t *testing.T) {
	fd := newFakeDiscovery([]*metav1.APIResourceList{
		{GroupVersion: "cert-manager.io/v1"},
		{GroupVersion: "v1"},
	})

	adapter := k8s.NewAPIGroupAdapter(fd)
	present, err := adapter.HasAPIGroup(context.Background(), "cert-manager.io")
	if err != nil {
		t.Fatalf("HasAPIGroup: %v", err)
	}
	if !present {
		t.Errorf("HasAPIGroup(cert-manager.io): got false, want true")
	}
}

func TestAPIGroupAdapterHasAPIGroupAbsent(t *testing.T) {
	fd := newFakeDiscovery([]*metav1.APIResourceList{
		{GroupVersion: "storage.k8s.io/v1"},
	})

	adapter := k8s.NewAPIGroupAdapter(fd)
	present, err := adapter.HasAPIGroup(context.Background(), "cert-manager.io")
	if err != nil {
		t.Fatalf("HasAPIGroup: %v", err)
	}
	if present {
		t.Errorf("HasAPIGroup(cert-manager.io): got true, want false")
	}
}

func TestAPIGroupAdapterServerGroupsError(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("get", "group", func(_ ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("forbidden")
	})

	// Discovery aus fake-clientset zieht Reactor-Pfade — auf get/group
	// reagieren wir mit Fehler.
	adapter := k8s.NewAPIGroupAdapter(cs.Discovery())
	if _, err := adapter.HasAPIGroup(context.Background(), "cert-manager.io"); err == nil {
		t.Errorf("expected error, got nil")
	}
}
