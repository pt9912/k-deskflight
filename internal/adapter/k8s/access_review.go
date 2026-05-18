/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package k8s

import (
	"context"
	"fmt"

	authzv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

// AccessReviewAdapter implementiert `port.AccessReviewer` gegen
// `authorization.k8s.io/v1.SelfSubjectAccessReview` (architecture.md
// AR-018, slice-M5 §2.1). Genutzt vom Reconciler-Pre-Execution-Pfad,
// um vor jedem Check zu prüfen, ob der Operator-ServiceAccount die
// nötigen Cluster-Rechte hat.
type AccessReviewAdapter struct {
	client kubernetes.Interface
}

// NewAccessReviewAdapter baut den Adapter aus einem geteilten
// Clientset (`ClusterClients.Clientset`).
func NewAccessReviewAdapter(c kubernetes.Interface) *AccessReviewAdapter {
	return &AccessReviewAdapter{client: c}
}

// CanI führt eine SelfSubjectAccessReview gegen den API-Server aus.
//
// Drei-Outcome-Kontrakt (slice-M5 §2.3, deckungsgleich mit
// `port.AccessReviewer`):
//   - `(true, nil)` — Permission gewährt.
//   - `(false, nil)` — Permission verweigert.
//   - `(false, err)` — SAR-Subsystem-Aufruf gescheitert (Netzwerk,
//     Auth-Webhook, Encoding-Fehler etc.).
//
// `PermissionRequest`-Felder werden 1:1 auf
// `authzv1.ResourceAttributes` übertragen — der Domain-Layer bleibt
// k8s-frei (AR-005 `domain-isolation`).
func (a *AccessReviewAdapter) CanI(ctx context.Context, req domain.PermissionRequest) (bool, error) {
	sar := &authzv1.SelfSubjectAccessReview{
		Spec: authzv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authzv1.ResourceAttributes{
				Namespace:   req.Namespace,
				Verb:        req.Verb,
				Group:       req.Group,
				Resource:    req.Resource,
				Subresource: req.Subresource,
			},
		},
	}
	out, err := a.client.AuthorizationV1().
		SelfSubjectAccessReviews().
		Create(ctx, sar, metav1.CreateOptions{})
	if err != nil {
		return false, fmt.Errorf("self subject access review %s: %w", req.CanonicalString(), err)
	}
	return out.Status.Allowed, nil
}
