/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package application

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

// SinglePodTopologyMismatchReason is the AR-026 reason marker that
// guard violations carry in their error message. Tests assert against
// this string instead of the full error format so that wording changes
// in the surrounding sentence do not break the contract.
const SinglePodTopologyMismatchReason = "SinglePodTopologyMismatch"

// EnforceSinglePodTopology guards against `--leader-elect=false` being
// run while more than one operator pod is active in the operator
// namespace (architecture.md AR-026; slice-M7 §2.8). It is intended to
// be called once during operator startup before the manager starts.
//
// Behaviour matrix:
//
//   - leaderElect=true                                          → no-op (controller-runtime coordinates multi-pod via leases).
//   - leaderElect=false, expectedReplicas<1                     → normalise to 1, then proceed.
//   - leaderElect=false, expectedReplicas>1                     → SinglePodTopologyMismatch (hard config error per AR-026).
//   - leaderElect=false, expectedReplicas==1, labelSelector=="" → SinglePodTopologyMismatch. This branch is a library-API safety net; the CLI wiring in cmd/operator/main.go always passes a non-empty operatorPodLabelSelector constant. Tests cover the branch so that future callers cannot accidentally drop the selector.
//   - leaderElect=false, expectedReplicas==1, list error        → SinglePodTopologyMismatch (wraps the API error).
//   - leaderElect=false, expectedReplicas==1, active pods >1    → SinglePodTopologyMismatch.
//   - leaderElect=false, expectedReplicas==1, active pods ≤1    → ok.
//
// "Active" means Pod.Status.Phase ∈ {Running, Pending}. Succeeded,
// Failed and Unknown phases do not count toward the topology guard
// because they cannot serve reconciles.
//
// Parameters mirror the slice-M7 §2.8 signature exactly:
//
//	EnforceSinglePodTopology(ctx, podsAPI, namespace, labelSelector,
//	                        expectedReplicas, leaderElect) error
func EnforceSinglePodTopology(
	ctx context.Context,
	podsAPI corev1client.PodsGetter,
	namespace string,
	labelSelector string,
	expectedReplicas int,
	leaderElect bool,
) error {
	if leaderElect {
		return nil
	}
	if expectedReplicas < 1 {
		expectedReplicas = 1
	}
	if expectedReplicas > 1 {
		return fmt.Errorf(
			"%s: --leader-elect=false with --expected-replica-count=%d (>1) is a configuration error; either enable leader election or set --expected-replica-count=1",
			SinglePodTopologyMismatchReason, expectedReplicas,
		)
	}
	if labelSelector == "" {
		return fmt.Errorf(
			"%s: empty label selector — refusing to count pods in namespace %q without a selector",
			SinglePodTopologyMismatchReason, namespace,
		)
	}
	pods, err := podsAPI.Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return fmt.Errorf(
			"%s: list pods in %q: %w",
			SinglePodTopologyMismatchReason, namespace, err,
		)
	}
	active := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending {
			active++
		}
	}
	if active > 1 {
		return fmt.Errorf(
			"%s: --leader-elect=false expects 1 operator pod but %d are active in namespace %q (label=%q)",
			SinglePodTopologyMismatchReason, active, namespace, labelSelector,
		)
	}
	return nil
}
