/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package k8s_test

import (
	"context"
	"errors"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"

	"github.com/pt9912/k-deskflight/internal/adapter/k8s"
)

func TestIngressClassAdapterListIngressClasses(t *testing.T) {
	cs := fake.NewSimpleClientset(
		&networkingv1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{Name: "nginx"},
			Spec:       networkingv1.IngressClassSpec{Controller: "k8s.io/ingress-nginx"},
		},
		&networkingv1.IngressClass{
			ObjectMeta: metav1.ObjectMeta{Name: "traefik"},
			Spec:       networkingv1.IngressClassSpec{Controller: "traefik.io/ingress-controller"},
		},
	)

	adapter := k8s.NewIngressClassAdapter(cs)
	infos, err := adapter.ListIngressClasses(context.Background())
	if err != nil {
		t.Fatalf("ListIngressClasses: %v", err)
	}
	if got, want := len(infos), 2; got != want {
		t.Fatalf("count: got %d, want %d", got, want)
	}

	byName := make(map[string]string, len(infos))
	for _, info := range infos {
		byName[info.Name] = info.Controller
	}
	if byName["nginx"] != "k8s.io/ingress-nginx" {
		t.Errorf("nginx controller: got %q", byName["nginx"])
	}
	if byName["traefik"] != "traefik.io/ingress-controller" {
		t.Errorf("traefik controller: got %q", byName["traefik"])
	}
}

func TestIngressClassAdapterListError(t *testing.T) {
	cs := fake.NewSimpleClientset()
	cs.PrependReactor("list", "ingressclasses", func(_ ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("forbidden")
	})

	adapter := k8s.NewIngressClassAdapter(cs)
	if _, err := adapter.ListIngressClasses(context.Background()); err == nil {
		t.Errorf("expected error, got nil")
	}
}
