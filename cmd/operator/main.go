/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

// Package main wires the k-deskflight operator (architecture.md AR-003,
// AR-009). Der Entry-Point bleibt dünn: Scheme registrieren, Manager
// bauen, Adapter + Checks + Registry konstruieren, Reconciler injizieren,
// Signal-Handler setzen, blockieren. Alle fachliche Logik liegt unter
// internal/hexagon/.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	preflightv1alpha1 "github.com/pt9912/k-deskflight/api/v1alpha1"
	"github.com/pt9912/k-deskflight/internal/adapter/check"
	"github.com/pt9912/k-deskflight/internal/adapter/k8s"
	"github.com/pt9912/k-deskflight/internal/hexagon/application"
)

const (
	productName = "k-deskflight"

	// healthzAddr + metricsAddr exposen die controller-runtime-
	// Default-Endpoints (Healthz/Readyz auf :8081; Prometheus-
	// /metrics auf :8080) für Liveness-/Readiness-Probes und für
	// das HTTP-Smoke-Script (scripts/operator-http-smoke.sh).
	// /metrics ist in M3 unauthentisiert exposed — der vollständige
	// Prometheus-Scrape-Pfad (RBAC, ServiceMonitor, Auth-Filter)
	// bleibt M6 (Roadmap §3 M6, AR-024).
	healthzAddr = ":8081"
	metricsAddr = ":8080"

	// defaultLeaderElectionID is the coordination.k8s.io/lease name
	// used by the controller-runtime Manager. Stable across operator
	// restarts (AR-026, slice-M7 §2.8).
	defaultLeaderElectionID = "k-deskflight-operator"

	// operatorPodLabelSelector matches the operator Deployment template
	// labels in deploy/manifests/deployment.yaml. Used by the single-pod
	// topology guard (AR-026) when --leader-elect=false.
	operatorPodLabelSelector = "app.kubernetes.io/name=k-deskflight,app.kubernetes.io/component=operator"
)

// runConfig bundles the CLI-flag-derived knobs that run() consumes.
// Kept as a struct so that adding a flag does not change the
// signature on every call site (AR-026 wiring is the first user;
// future v0.2 will likely add more knobs).
type runConfig struct {
	leaderElect             bool
	expectedReplicaCount    int
	leaderElectionID        string
	leaderElectionNamespace string
}

func main() {
	cfg := runConfig{}
	flag.BoolVar(&cfg.leaderElect, "leader-elect", true,
		"Enable leader election for controller manager. Default true (AR-026 production mode); set to false only for debug/isolated single-pod runs (requires pods list permission in the operator namespace).")
	flag.IntVar(&cfg.expectedReplicaCount, "expected-replica-count", 1,
		"Expected operator replica count. Only consulted when --leader-elect=false to enforce the single-pod topology guard (AR-026). Values <1 are normalised to 1.")
	flag.StringVar(&cfg.leaderElectionID, "leader-election-id", defaultLeaderElectionID,
		"Name of the coordination.k8s.io/lease that holds the leader election state.")
	flag.StringVar(&cfg.leaderElectionNamespace, "leader-election-namespace", "",
		"Namespace for the leader-election lease. Default: POD_NAMESPACE env (via Downward-API). Fallback to 'default' is debug-only — production deployments MUST set POD_NAMESPACE via Downward-API (see deploy/manifests/deployment.yaml).")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("operator starting",
		slog.String("product", productName),
		slog.Bool("leader_elect", cfg.leaderElect),
		slog.Int("expected_replica_count", cfg.expectedReplicaCount),
		slog.String("leader_election_id", cfg.leaderElectionID),
	)

	if err := run(logger, cfg); err != nil {
		logger.Error("operator exited with error", slog.String("err", err.Error()))
		os.Exit(1)
	}
}

// resolveLeaderElectionNamespace picks the leader-election namespace
// in this priority order: explicit --leader-election-namespace flag →
// POD_NAMESPACE env (injected via Downward-API in
// deploy/manifests/deployment.yaml) → metav1.NamespaceDefault.
// The "fallback default" branch warns loudly because AR-026 expects
// the lease in the operator's own namespace; landing in 'default' is
// almost always a deploy-time misconfiguration in production.
func resolveLeaderElectionNamespace(flagValue string, logger *slog.Logger) (string, string) {
	if flagValue != "" {
		return flagValue, "flag"
	}
	if env := os.Getenv("POD_NAMESPACE"); env != "" {
		return env, "POD_NAMESPACE env"
	}
	logger.Warn(
		"leader-election namespace fell back to 'default' — production deployments MUST inject POD_NAMESPACE via Downward-API",
		slog.String("hint", "see deploy/manifests/deployment.yaml env block"),
	)
	return metav1.NamespaceDefault, "fallback default"
}

func run(logger *slog.Logger, cfg runConfig) error {
	ctrl.SetLogger(zap.New(zap.UseDevMode(false)))

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("register client-go scheme: %w", err)
	}
	if err := preflightv1alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("register v1alpha1 scheme: %w", err)
	}

	restCfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("load kubeconfig: %w", err)
	}

	leNamespace, leNamespaceSource := resolveLeaderElectionNamespace(cfg.leaderElectionNamespace, logger)

	mgr, err := ctrl.NewManager(restCfg, manager.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: healthzAddr,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		// AR-026: production-mode leader election via coordination.k8s.io
		// leases. ReleaseOnCancel speeds up failover when the leader pod
		// receives SIGTERM — the lease is released immediately rather than
		// waiting for the renew deadline.
		LeaderElection:                cfg.leaderElect,
		LeaderElectionID:              cfg.leaderElectionID,
		LeaderElectionNamespace:       leNamespace,
		LeaderElectionResourceLock:    "leases",
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		return fmt.Errorf("register healthz check: %w", err)
	}
	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		return fmt.Errorf("register readyz check: %w", err)
	}

	// Adapter-Wiring (architecture.md AR-013, AR-009 Phase 1, slice-M4 §2.1).
	// NewClusterClients bündelt Clientset + Discovery-Client, damit alle
	// Discovery-Adapter denselben rest.Config-Pfad teilen.
	clients, err := k8s.NewClusterClients(restCfg)
	if err != nil {
		return fmt.Errorf("create cluster clients: %w", err)
	}

	// AR-026 single-pod topology guard. Only active when leader election
	// is disabled — the production path (--leader-elect=true) relies on
	// the lease-based coordinator and skips the pod-list check.
	//
	// 15s timeout matches the AR-026 lease duration: if the API server
	// is unreachable for longer than that, the operator startup is
	// considered failed and the container restarts (rather than hanging
	// in pod-list indefinitely, which would only be caught by the
	// livenessProbe after another 10–30s).
	guardCtx, guardCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer guardCancel()
	if err := application.EnforceSinglePodTopology(
		guardCtx,
		clients.Clientset.CoreV1(),
		leNamespace,
		operatorPodLabelSelector,
		cfg.expectedReplicaCount,
		cfg.leaderElect,
	); err != nil {
		return fmt.Errorf("single-pod topology guard: %w", err)
	}

	discoveryAdapter := k8s.NewDiscoveryAdapterWithClient(clients.Discovery)
	storageAdapter := k8s.NewStorageClassAdapter(clients.Clientset)
	ingressAdapter := k8s.NewIngressClassAdapter(clients.Clientset)
	apiGroupAdapter := k8s.NewAPIGroupAdapter(clients.Discovery)
	nodeAdapter := k8s.NewNodeAdapter(clients.Clientset)
	accessReviewer := k8s.NewAccessReviewAdapter(clients.Clientset)

	registry := check.NewRegistry()
	registry.Register(check.NewKubernetesVersion(discoveryAdapter, nil))
	registry.Register(check.NewStorageClass(storageAdapter, nil))
	registry.Register(check.NewIngressClass(ingressAdapter, nil))
	registry.Register(check.NewCertManager(apiGroupAdapter, nil))
	registry.Register(check.NewClusterResources(nodeAdapter, nil))

	reconciler := &application.Reconciler{
		Client:         mgr.GetClient(),
		Scheme:         mgr.GetScheme(),
		Registry:       registry,
		AccessReviewer: accessReviewer,
		Logger:         logger,
	}
	if err := reconciler.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setup reconciler: %w", err)
	}

	logger.Info("manager starting — blocking on signal handler",
		slog.String("leader_election_namespace", leNamespace),
		slog.String("leader_election_namespace_source", leNamespaceSource))
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("manager start: %w", err)
	}
	return nil
}
