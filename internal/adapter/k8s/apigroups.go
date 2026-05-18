/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package k8s

import (
	"context"
	"fmt"

	"k8s.io/client-go/discovery"
)

// APIGroupAdapter implementiert `port.APIGroupDiscovery` gegen den
// Discovery-Endpoint. Genutzt vom cert-manager-Existence-Check
// (`LH-F-013`). M4-untested.
type APIGroupAdapter struct {
	client discovery.DiscoveryInterface
}

// NewAPIGroupAdapter baut den Adapter aus einem geteilten
// Discovery-Client (`ClusterClients.Discovery`).
func NewAPIGroupAdapter(c discovery.DiscoveryInterface) *APIGroupAdapter {
	return &APIGroupAdapter{client: c}
}

// HasAPIGroup prüft, ob die angegebene API-Gruppe vom Server gemeldet
// wird. Die client-go-`ServerGroups`-API nimmt aktuell keinen Context
// — Cancellation läuft transparent über die rest.Config-Timeouts
// (analog zu `DiscoveryAdapter.ServerVersion`).
func (a *APIGroupAdapter) HasAPIGroup(_ context.Context, name string) (bool, error) {
	groups, err := a.client.ServerGroups()
	if err != nil {
		return false, fmt.Errorf("discovery ServerGroups: %w", err)
	}
	for _, g := range groups.Groups {
		if g.Name == name {
			return true, nil
		}
	}
	return false, nil
}
