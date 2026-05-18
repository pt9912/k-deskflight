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
)

type fakeAPIGroupDiscovery struct {
	present bool
	err     error
}

func (f fakeAPIGroupDiscovery) HasAPIGroup(_ context.Context, _ string) (bool, error) {
	return f.present, f.err
}

func TestCertManagerInstalled(t *testing.T) {
	t.Parallel()
	disc := fakeAPIGroupDiscovery{present: true}
	chk := check.NewCertManager(disc, fixedClock())

	res := chk.Run(context.Background(), domain.CertManagerSpec{})

	if res.Status != domain.StatusTrue {
		t.Errorf("Status: got %q, want %q (message=%q)", res.Status, domain.StatusTrue, res.Message)
	}
	if res.Reason != "CertManagerInstalled" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "CertManagerInstalled")
	}
	if res.Severity != domain.SeverityInfo {
		t.Errorf("Severity: got %q, want %q", res.Severity, domain.SeverityInfo)
	}
}

// TestCertManagerMissingSeverityIsWarning fixiert slice-M4 §9 — bei
// fehlendem cert-manager bleibt die Severity bewusst `warning`, NICHT
// `critical`. Bricht der Test, ist es eine bewusste Produkt-
// entscheidung und muss im Plan reflektiert werden.
func TestCertManagerMissingSeverityIsWarning(t *testing.T) {
	t.Parallel()
	disc := fakeAPIGroupDiscovery{present: false}
	chk := check.NewCertManager(disc, fixedClock())

	res := chk.Run(context.Background(), domain.CertManagerSpec{})

	if res.Status != domain.StatusFalse {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusFalse)
	}
	if res.Reason != "CertManagerMissing" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "CertManagerMissing")
	}
	if res.Severity != domain.SeverityWarning {
		t.Errorf("Severity: got %q, want %q (slice-M4 §9 — Warning ist bewusste Entscheidung)",
			res.Severity, domain.SeverityWarning)
	}
}

// TestCertManagerMissingMessageMentionsAlternatives fixiert slice-M4
// §3.1 / §7 #11 — der Pflicht-Message-Inhalt überbrückt die M6-Doku-
// Verbindlichkeit aus §9. Anwender sehen die zwei legitimen
// Alternativen (Install / externe TLS-Terminierung) direkt im
// CR-Status.
func TestCertManagerMissingMessageMentionsAlternatives(t *testing.T) {
	t.Parallel()
	disc := fakeAPIGroupDiscovery{present: false}
	chk := check.NewCertManager(disc, fixedClock())

	res := chk.Run(context.Background(), domain.CertManagerSpec{})

	if !strings.Contains(res.Message, "external TLS termination") {
		t.Errorf("Message must mention external TLS termination (slice-M4 §3.1 message bridge); got %q", res.Message)
	}
	if !strings.Contains(res.Message, "install cert-manager") {
		t.Errorf("Message must name cert-manager-install alternative; got %q", res.Message)
	}
}

func TestCertManagerLookupFailed(t *testing.T) {
	t.Parallel()
	disc := fakeAPIGroupDiscovery{err: errors.New("forbidden")}
	chk := check.NewCertManager(disc, fixedClock())

	res := chk.Run(context.Background(), domain.CertManagerSpec{})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "CertManagerLookupFailed" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "CertManagerLookupFailed")
	}
}

func TestCertManagerInvalidSpec(t *testing.T) {
	t.Parallel()
	disc := fakeAPIGroupDiscovery{present: true}
	chk := check.NewCertManager(disc, fixedClock())

	res := chk.Run(context.Background(), otherSpec{})

	if res.Status != domain.StatusUnknown {
		t.Errorf("Status: got %q, want %q", res.Status, domain.StatusUnknown)
	}
	if res.Reason != "InvalidSpec" {
		t.Errorf("Reason: got %q, want %q", res.Reason, "InvalidSpec")
	}
}
