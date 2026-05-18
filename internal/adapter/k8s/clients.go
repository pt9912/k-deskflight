/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package k8s

import (
	"fmt"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ClusterClients bündelt die Kubernetes-Clients, die der Operator zur
// Cluster-Inspection braucht (architecture.md AR-009, slice-M4 §3.2).
// Wiring in cmd/operator/main.go nutzt diesen Konstruktor, damit alle
// Discovery-Adapter denselben Clientset und Discovery-Client teilen.
type ClusterClients struct {
	Clientset kubernetes.Interface
	Discovery discovery.DiscoveryInterface
}

// NewClusterClients baut das ClusterClients-Bundle aus einer rest.Config
// (typisch: `mgr.GetConfig()`). In-Cluster läuft das mit dem
// ServiceAccount-Token, out-of-cluster via kubeconfig.
func NewClusterClients(cfg *rest.Config) (*ClusterClients, error) {
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes clientset: %w", err)
	}
	d, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create discovery client: %w", err)
	}
	return &ClusterClients{Clientset: cs, Discovery: d}, nil
}
