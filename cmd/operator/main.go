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
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	preflightv1alpha1 "github.com/pt9912/k-deskflight/api/v1alpha1"
	"github.com/pt9912/k-deskflight/internal/adapter/check"
	"github.com/pt9912/k-deskflight/internal/adapter/k8s"
	"github.com/pt9912/k-deskflight/internal/hexagon/application"
)

const productName = "k-deskflight"

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

	mgr, err := ctrl.NewManager(cfg, manager.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	// Adapter-Wiring (architecture.md AR-013, AR-009 Phase 1).
	discoveryAdapter, err := k8s.NewDiscoveryAdapter(cfg)
	if err != nil {
		return fmt.Errorf("create discovery adapter: %w", err)
	}

	registry := check.NewRegistry()
	registry.Register(check.NewKubernetesVersion(discoveryAdapter, nil))

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
