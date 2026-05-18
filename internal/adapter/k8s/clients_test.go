/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package k8s_test

import (
	"testing"

	"k8s.io/client-go/rest"

	"github.com/pt9912/k-deskflight/internal/adapter/k8s"
)

func TestNewClusterClientsHappyPath(t *testing.T) {
	clients, err := k8s.NewClusterClients(&rest.Config{Host: "http://localhost:1"})
	if err != nil {
		t.Fatalf("NewClusterClients: %v", err)
	}
	if clients.Clientset == nil {
		t.Errorf("Clientset is nil")
	}
	if clients.Discovery == nil {
		t.Errorf("Discovery is nil")
	}
}

func TestNewClusterClientsInvalidConfig(t *testing.T) {
	// rest.Config mit konfligierender Auth — `kubernetes.NewForConfig`
	// lehnt das ab.
	cfg := &rest.Config{
		Host:        "http://localhost:1",
		BearerToken: "x",
		Username:    "y",
		Password:    "z",
	}
	if _, err := k8s.NewClusterClients(cfg); err == nil {
		t.Errorf("expected error from invalid rest.Config, got nil")
	}
}
