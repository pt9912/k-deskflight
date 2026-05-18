/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package port

import (
	"context"

	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

// AccessReviewer prüft, ob der Operator-ServiceAccount eine
// angeforderte Permission gegen den aktuellen Cluster gewährt bekommt
// (architecture.md AR-018, slice-M5 §2.1). Implementiert vom
// `internal/adapter/k8s/access_review.go`-Adapter via
// `authorization.k8s.io/v1.SelfSubjectAccessReview`.
//
// Kontrakt für den Caller (slice-M5 §2.3):
//
//   - `(true, nil)` — Permission wird gewährt, Check darf laufen.
//   - `(false, nil)` — Permission wird verweigert, Check fällt mit
//     Reason `RBACInsufficient` aus.
//   - `(_, err)` — SAR-Subsystem-Aufruf gescheitert (transient,
//     Auth-Webhook-Fehler, Netzwerk); Check fällt mit Reason
//     `RBACCheckFailed` aus. Der `bool`-Rückgabewert ist in diesem
//     Fall nicht zu interpretieren; Konvention: `false`, aber der
//     Caller MUSS `err != nil` zuerst prüfen.
type AccessReviewer interface {
	CanI(ctx context.Context, req domain.PermissionRequest) (bool, error)
}
