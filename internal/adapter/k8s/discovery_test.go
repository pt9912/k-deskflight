/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package k8s_test

import (
	"context"
	"errors"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	ktesting "k8s.io/client-go/testing"

	"github.com/pt9912/k-deskflight/internal/adapter/k8s"
)

func TestDiscoveryAdapterServerVersion(t *testing.T) {
	cs := fake.NewSimpleClientset()
	fd := cs.Discovery().(*fakediscovery.FakeDiscovery)
	fd.FakedServerVersion = &version.Info{GitVersion: "v1.34.2"}

	adapter := k8s.NewDiscoveryAdapterWithClient(cs.Discovery())
	got, err := adapter.ServerVersion(context.Background())
	if err != nil {
		t.Fatalf("ServerVersion: %v", err)
	}
	if got != "v1.34.2" {
		t.Errorf("ServerVersion: got %q, want %q", got, "v1.34.2")
	}
}

func TestDiscoveryAdapterServerVersionError(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("get", "version", func(_ ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("connection refused")
	})

	adapter := k8s.NewDiscoveryAdapterWithClient(cs.Discovery())
	if _, err := adapter.ServerVersion(context.Background()); err == nil {
		t.Errorf("expected error, got nil")
	}
}

// TestNewDiscoveryAdapterFromConfigInvalid deckt den Fehlerpfad in
// `NewDiscoveryAdapter(*rest.Config)`. Eine rest.Config mit
// inkompatiblen Auth-Feldern lässt `discovery.NewDiscoveryClientForConfig`
// fehlschlagen.
func TestNewDiscoveryAdapterFromConfigInvalid(t *testing.T) {
	cfg := &rest.Config{
		Host:        "http://localhost:1",
		BearerToken: "x",
		Username:    "y",
		Password:    "z",
	}
	if _, err := k8s.NewDiscoveryAdapter(cfg); err == nil {
		t.Errorf("expected error from invalid rest.Config, got nil")
	}
}
