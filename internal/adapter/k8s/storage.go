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

// Annotationen, die eine StorageClass als Cluster-Default markieren.
// Adapter toleriert beide gleichwertig (slice-M4 §2.1 / §9): der
// GA-Schlüssel auf modernen Clustern, der beta-Schlüssel auf
// historisch gewachsenen Installationen.
const (
	defaultStorageClassAnnotationGA     = "storageclass.kubernetes.io/is-default-class"
	defaultStorageClassAnnotationLegacy = "storageclass.beta.kubernetes.io/is-default-class"
)

// StorageClassAdapter implementiert `port.StorageClassDiscovery` gegen
// die typed clientset-API (`storage.k8s.io/v1`). M4-untested
// (envtest-Pflicht M6).
type StorageClassAdapter struct {
	client kubernetes.Interface
}

// NewStorageClassAdapter baut den Adapter aus einem geteilten
// Clientset (`ClusterClients.Clientset`).
func NewStorageClassAdapter(c kubernetes.Interface) *StorageClassAdapter {
	return &StorageClassAdapter{client: c}
}

// ListStorageClasses liefert alle StorageClasses des Clusters samt
// Default-Markierung und Provisioner-String.
func (a *StorageClassAdapter) ListStorageClasses(ctx context.Context) ([]port.StorageClassInfo, error) {
	list, err := a.client.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list storage classes: %w", err)
	}
	out := make([]port.StorageClassInfo, 0, len(list.Items))
	for _, sc := range list.Items {
		out = append(out, port.StorageClassInfo{
			Name:        sc.Name,
			IsDefault:   isDefaultStorageClass(sc.Annotations),
			Provisioner: sc.Provisioner,
		})
	}
	return out, nil
}

func isDefaultStorageClass(annotations map[string]string) bool {
	if annotations[defaultStorageClassAnnotationGA] == "true" {
		return true
	}
	if annotations[defaultStorageClassAnnotationLegacy] == "true" {
		return true
	}
	return false
}
