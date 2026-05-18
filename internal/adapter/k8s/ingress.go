/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

// IngressClassAdapter implementiert `port.IngressClassDiscovery` gegen
// `networking.k8s.io/v1`. K8s-Floor 1.34 (ADR 0009) macht den v1beta1-
// Fallback unnötig (siehe slice-M4 §9). M4-untested.
type IngressClassAdapter struct {
	client kubernetes.Interface
}

// NewIngressClassAdapter baut den Adapter aus einem geteilten Clientset.
func NewIngressClassAdapter(c kubernetes.Interface) *IngressClassAdapter {
	return &IngressClassAdapter{client: c}
}

// ListIngressClasses liefert alle IngressClasses des Clusters samt
// `Spec.Controller`-String.
func (a *IngressClassAdapter) ListIngressClasses(ctx context.Context) ([]port.IngressClassInfo, error) {
	list, err := a.client.NetworkingV1().IngressClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list ingress classes: %w", err)
	}
	out := make([]port.IngressClassInfo, 0, len(list.Items))
	for _, ic := range list.Items {
		out = append(out, port.IngressClassInfo{
			Name:       ic.Name,
			Controller: ic.Spec.Controller,
		})
	}
	return out, nil
}
