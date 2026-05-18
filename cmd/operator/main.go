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
	"fmt"
	"log/slog"
	"os"

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
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("operator starting", slog.String("product", productName))

	if err := run(logger); err != nil {
		logger.Error("operator exited with error", slog.String("err", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctrl.SetLogger(zap.New(zap.UseDevMode(false)))

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("register client-go scheme: %w", err)
	}
	if err := preflightv1alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("register v1alpha1 scheme: %w", err)
	}

	cfg, err := ctrl.GetConfig()
	if err != nil {
		return fmt.Errorf("load kubeconfig: %w", err)
	}

	mgr, err := ctrl.NewManager(cfg, manager.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: healthzAddr,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
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
	clients, err := k8s.NewClusterClients(cfg)
	if err != nil {
		return fmt.Errorf("create cluster clients: %w", err)
	}

	discoveryAdapter := k8s.NewDiscoveryAdapterWithClient(clients.Discovery)
	storageAdapter := k8s.NewStorageClassAdapter(clients.Clientset)
	ingressAdapter := k8s.NewIngressClassAdapter(clients.Clientset)
	apiGroupAdapter := k8s.NewAPIGroupAdapter(clients.Discovery)
	nodeAdapter := k8s.NewNodeAdapter(clients.Clientset)

	registry := check.NewRegistry()
	registry.Register(check.NewKubernetesVersion(discoveryAdapter, nil))
	registry.Register(check.NewStorageClass(storageAdapter, nil))
	registry.Register(check.NewIngressClass(ingressAdapter, nil))
	registry.Register(check.NewCertManager(apiGroupAdapter, nil))
	registry.Register(check.NewClusterResources(nodeAdapter, nil))

	reconciler := &application.Reconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Registry: registry,
	}
	if err := reconciler.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("setup reconciler: %w", err)
	}

	logger.Info("manager starting — blocking on signal handler")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		return fmt.Errorf("manager start: %w", err)
	}
	return nil
}
