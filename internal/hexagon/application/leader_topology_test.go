/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/pt9912/k-deskflight/internal/hexagon/application"
)

const (
	testNS       = "k-deskflight-system"
	testSelector = "app.kubernetes.io/name=k-deskflight,app.kubernetes.io/component=operator"
)

func makeOperatorPod(name string, phase corev1.PodPhase) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNS,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "k-deskflight",
				"app.kubernetes.io/component": "operator",
			},
		},
		Status: corev1.PodStatus{Phase: phase},
	}
}

// TestEnforceSinglePodTopology_LeaderElectTrueIsNoOp verifies that the
// guard short-circuits when leader election is enabled — controller-
// runtime handles multi-pod coordination via leases (AR-026).
func TestEnforceSinglePodTopology_LeaderElectTrueIsNoOp(t *testing.T) {
	// A clientset that would explode if called: any List/Get triggers a
	// reactor error. Passing it through with leaderElect=true must not
	// touch the API.
	client := fake.NewSimpleClientset()
	client.PrependReactor("list", "pods", func(_ ktesting.Action) (bool, runtime.Object, error) {
		t.Fatal("guard must not call API when leaderElect=true")
		return false, nil, nil
	})
	if err := application.EnforceSinglePodTopology(
		context.Background(), client.CoreV1(), testNS, testSelector, 1, true,
	); err != nil {
		t.Fatalf("expected nil error for leaderElect=true, got %v", err)
	}
}

// TestEnforceSinglePodTopology_NoPodsOK covers the typical startup
// path: leader-elect=false, single replica, no other operator pods in
// the namespace.
func TestEnforceSinglePodTopology_NoPodsOK(t *testing.T) {
	client := fake.NewSimpleClientset()
	if err := application.EnforceSinglePodTopology(
		context.Background(), client.CoreV1(), testNS, testSelector, 1, false,
	); err != nil {
		t.Fatalf("expected nil error for empty namespace, got %v", err)
	}
}

// TestEnforceSinglePodTopology_OneRunningPodOK is the "self only" path
// — the operator pod itself is running, no other replica exists.
func TestEnforceSinglePodTopology_OneRunningPodOK(t *testing.T) {
	client := fake.NewSimpleClientset(
		makeOperatorPod("k-deskflight-operator-self", corev1.PodRunning),
	)
	if err := application.EnforceSinglePodTopology(
		context.Background(), client.CoreV1(), testNS, testSelector, 1, false,
	); err != nil {
		t.Fatalf("expected nil error for single running pod, got %v", err)
	}
}

// TestEnforceSinglePodTopology_TwoRunningPodsRejected is the safety
// path: two concurrently active operator pods under --leader-elect=false
// would double-reconcile (AR-026). Guard must reject.
func TestEnforceSinglePodTopology_TwoRunningPodsRejected(t *testing.T) {
	client := fake.NewSimpleClientset(
		makeOperatorPod("k-deskflight-operator-a", corev1.PodRunning),
		makeOperatorPod("k-deskflight-operator-b", corev1.PodRunning),
	)
	err := application.EnforceSinglePodTopology(
		context.Background(), client.CoreV1(), testNS, testSelector, 1, false,
	)
	if err == nil {
		t.Fatal("expected SinglePodTopologyMismatch for 2 running pods, got nil")
	}
	if !strings.Contains(err.Error(), application.SinglePodTopologyMismatchReason) {
		t.Fatalf("expected reason marker %q in error, got %q",
			application.SinglePodTopologyMismatchReason, err.Error())
	}
}

// TestEnforceSinglePodTopology_RunningPlusPendingRejected checks that
// Pending is counted as active alongside Running — the second pod is
// already scheduled and will start reconciling momentarily.
func TestEnforceSinglePodTopology_RunningPlusPendingRejected(t *testing.T) {
	client := fake.NewSimpleClientset(
		makeOperatorPod("k-deskflight-operator-running", corev1.PodRunning),
		makeOperatorPod("k-deskflight-operator-pending", corev1.PodPending),
	)
	err := application.EnforceSinglePodTopology(
		context.Background(), client.CoreV1(), testNS, testSelector, 1, false,
	)
	if err == nil {
		t.Fatal("expected SinglePodTopologyMismatch for running+pending, got nil")
	}
}

// TestEnforceSinglePodTopology_SucceededFailedSkipped verifies that
// terminal phases (Succeeded, Failed, Unknown) do not count toward the
// active-pod tally — only Running and Pending pods can serve
// reconciles.
func TestEnforceSinglePodTopology_SucceededFailedSkipped(t *testing.T) {
	client := fake.NewSimpleClientset(
		makeOperatorPod("k-deskflight-operator-active", corev1.PodRunning),
		makeOperatorPod("k-deskflight-operator-old-1", corev1.PodSucceeded),
		makeOperatorPod("k-deskflight-operator-old-2", corev1.PodFailed),
		makeOperatorPod("k-deskflight-operator-old-3", corev1.PodUnknown),
	)
	if err := application.EnforceSinglePodTopology(
		context.Background(), client.CoreV1(), testNS, testSelector, 1, false,
	); err != nil {
		t.Fatalf("expected nil error (terminal phases skipped), got %v", err)
	}
}

// TestEnforceSinglePodTopology_ExpectedReplicasGreaterThanOneRejected
// checks the hard configuration error: --leader-elect=false combined
// with --expected-replica-count > 1 is a self-inconsistent CLI
// invocation (AR-026).
func TestEnforceSinglePodTopology_ExpectedReplicasGreaterThanOneRejected(t *testing.T) {
	client := fake.NewSimpleClientset()
	err := application.EnforceSinglePodTopology(
		context.Background(), client.CoreV1(), testNS, testSelector, 3, false,
	)
	if err == nil {
		t.Fatal("expected SinglePodTopologyMismatch for expectedReplicas=3, got nil")
	}
	if !strings.Contains(err.Error(), "expected-replica-count=3") {
		t.Fatalf("expected error to mention expectedReplicas value, got %q", err.Error())
	}
}

// TestEnforceSinglePodTopology_ExpectedReplicasZeroNormalisedToOne
// verifies the AR-026 normalisation: values <1 are clamped to 1.
func TestEnforceSinglePodTopology_ExpectedReplicasZeroNormalisedToOne(t *testing.T) {
	client := fake.NewSimpleClientset(
		makeOperatorPod("k-deskflight-operator-self", corev1.PodRunning),
	)
	if err := application.EnforceSinglePodTopology(
		context.Background(), client.CoreV1(), testNS, testSelector, 0, false,
	); err != nil {
		t.Fatalf("expected nil error after normalisation to 1, got %v", err)
	}
}

// TestEnforceSinglePodTopology_EmptySelectorRejected ensures the guard
// refuses to count without a selector — listing all pods in the
// namespace would risk including unrelated workloads.
func TestEnforceSinglePodTopology_EmptySelectorRejected(t *testing.T) {
	client := fake.NewSimpleClientset()
	err := application.EnforceSinglePodTopology(
		context.Background(), client.CoreV1(), testNS, "", 1, false,
	)
	if err == nil {
		t.Fatal("expected SinglePodTopologyMismatch for empty selector, got nil")
	}
	if !strings.Contains(err.Error(), "empty label selector") {
		t.Fatalf("expected error to mention empty selector, got %q", err.Error())
	}
}

// TestEnforceSinglePodTopology_ListErrorWrapped attests that an API
// list error is reported with the AR-026 reason marker so that startup
// logs unambiguously flag the configuration root cause.
func TestEnforceSinglePodTopology_ListErrorWrapped(t *testing.T) {
	client := fake.NewSimpleClientset()
	listErr := errors.New("synthetic forbidden")
	client.PrependReactor("list", "pods", func(_ ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, listErr
	})
	err := application.EnforceSinglePodTopology(
		context.Background(), client.CoreV1(), testNS, testSelector, 1, false,
	)
	if err == nil {
		t.Fatal("expected wrapped error from list failure, got nil")
	}
	if !strings.Contains(err.Error(), application.SinglePodTopologyMismatchReason) {
		t.Fatalf("expected reason marker in error, got %q", err.Error())
	}
	if !errors.Is(err, listErr) {
		t.Fatalf("expected errors.Is to find wrapped listErr, got %v", err)
	}
}
