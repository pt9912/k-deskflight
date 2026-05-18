/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package check_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pt9912/k-deskflight/internal/adapter/check"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
	"github.com/pt9912/k-deskflight/internal/hexagon/port"
)

type fakeIngressClassDiscovery struct {
	classes []port.IngressClassInfo
	err     error
}

func (f fakeIngressClassDiscovery) ListIngressClasses(_ context.Context) ([]port.IngressClassInfo, error) {
	return f.classes, f.err
}

func TestIngressClassPassed(t *testing.T) {
	t.Parallel()
	disc := fakeIngressClassDiscovery{classes: []port.IngressClassInfo{
		{Name: "nginx", Controller: "k8s.io/ingress-nginx"},
	}}
	chk := check.NewIngressClass(disc, fixedClock())

	res := chk.Run(context.Background(), domain.IngressClassSpec{Names: []string{"nginx"}})

	if res.Status != domain.StatusTrue {
		t.Errorf("Status: got %q, want %q (message=%q)", res.Status, domain.StatusTrue, res.Message)
	}
	if res.Reason != "IngressClassReady" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "IngressClassReady")
	}
	if res.Severity != domain.SeverityInfo {
		t.Errorf("Severity: got %q, want %q", res.Severity, domain.SeverityInfo)
	}
}

func TestIngressClassFailedMissing(t *testing.T) {
	t.Parallel()
	disc := fakeIngressClassDiscovery{classes: []port.IngressClassInfo{
		{Name: "traefik"},
	}}
	chk := check.NewIngressClass(disc, fixedClock())

	res := chk.Run(context.Background(), domain.IngressClassSpec{Names: []string{"nginx", "traefik"}})

	if res.Status != domain.StatusFalse {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusFalse)
	}
	if res.Reason != "IngressClassMissing" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "IngressClassMissing")
	}
	if res.Severity != domain.SeverityCritical {
		t.Errorf("Severity: got %q, want %q", res.Severity, domain.SeverityCritical)
	}
	if !strings.Contains(res.Message, "nginx") {
		t.Errorf("Message should name the missing class: got %q", res.Message)
	}
}

func TestIngressClassLookupFailed(t *testing.T) {
	t.Parallel()
	disc := fakeIngressClassDiscovery{err: errors.New("connection refused")}
	chk := check.NewIngressClass(disc, fixedClock())

	res := chk.Run(context.Background(), domain.IngressClassSpec{Names: []string{"nginx"}})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "IngressClassLookupFailed" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "IngressClassLookupFailed")
	}
}

func TestIngressClassInvalidSpec(t *testing.T) {
	t.Parallel()
	disc := fakeIngressClassDiscovery{}
	chk := check.NewIngressClass(disc, fixedClock())

	res := chk.Run(context.Background(), otherSpec{})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "InvalidSpec" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "InvalidSpec")
	}
}
