/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

// Package k8s enthält die Kubernetes-API-Adapter-Implementierungen
// (architecture.md AR-004 Adapter-Layer). In M3 ist nur der
// Discovery-Adapter belegt; M4 ergänzt StorageClass-/IngressClass-/
// cert-manager-Adapter, M5 die SelfSubjectAccessReview-Anbindung.
package k8s

import (
	"context"
	"fmt"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

// DiscoveryAdapter implementiert `port.KubernetesAPI` (M3:
// ausschließlich `ServerVersion`) gegen den Discovery-Endpoint des
// Kubernetes-API-Servers.
type DiscoveryAdapter struct {
	client discovery.DiscoveryInterface
}

// NewDiscoveryAdapter baut den Adapter aus einer rest.Config (typisch:
// `mgr.GetConfig()`). In-Cluster läuft das mit dem ServiceAccount-Token,
// out-of-cluster via kubeconfig.
func NewDiscoveryAdapter(cfg *rest.Config) (*DiscoveryAdapter, error) {
	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create discovery client: %w", err)
	}
	return NewDiscoveryAdapterWithClient(client), nil
}

// NewDiscoveryAdapterWithClient baut den Adapter direkt aus einem
// existierenden Discovery-Client. Genutzt von Tests (fake clientset)
// und vom Wiring in Step 5 (`ClusterClients.Discovery`).
func NewDiscoveryAdapterWithClient(client discovery.DiscoveryInterface) *DiscoveryAdapter {
	return &DiscoveryAdapter{client: client}
}

// ServerVersion liefert die rohe `GitVersion`-Zeichenkette des
// `/version`-Endpoints (z. B. "v1.34.2"). Die client-go-Discovery-API
// nimmt aktuell keinen Context entgegen — Cancellation läuft
// transparent über die rest.Config-Timeouts. M5-Härtung kann das mit
// einem Goroutine-Select-Pattern explizit cancelbar machen.
func (a *DiscoveryAdapter) ServerVersion(_ context.Context) (string, error) {
	info, err := a.client.ServerVersion()
	if err != nil {
		return "", fmt.Errorf("discovery ServerVersion: %w", err)
	}
	return info.GitVersion, nil
}
