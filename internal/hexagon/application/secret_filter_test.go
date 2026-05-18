/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/pt9912/k-deskflight/internal/hexagon/application"
	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

// TestSanitizeMessageIdentity fixiert das M5-Verhalten: Identität.
// v0.2 ersetzt das durch echte Pattern-Maskierung; bricht dieser Test,
// muss der Plan §2.6 explizit re-amend werden.
func TestSanitizeMessageIdentity(t *testing.T) {
	t.Parallel()
	in := "server version 1.34.2 satisfies minimum 1.34"
	if got := application.SanitizeMessage(in); got != in {
		t.Errorf("SanitizeMessage: got %q, want identity %q", got, in)
	}
	if got := application.SanitizeMessage(""); got != "" {
		t.Errorf("SanitizeMessage(empty): got %q, want empty", got)
	}
}

// TestSanitizeAttrsIdentity fixiert M5-Verhalten für den Attribut-Pfad.
func TestSanitizeAttrsIdentity(t *testing.T) {
	t.Parallel()
	in := []slog.Attr{
		slog.String("foo", "bar"),
		slog.Int("count", 3),
	}
	got := application.SanitizeAttrs(in...)
	if len(got) != 2 || got[0].Key != "foo" || got[1].Key != "count" {
		t.Errorf("SanitizeAttrs identity broken: got %+v", got)
	}
}

// TestLogResultBuildsExpectedAttrs verifiziert das Pflicht-Pattern aus
// slice-M5 §2.6: LogResult baut die Standard-Attrs (check, status,
// reason, severity, message) und hängt extras hinten an. Sanitize-Hooks
// laufen im selben Pfad — wir verifizieren das indirekt, weil sie in
// M5 Identitäts-Funktionen sind, aber das Aufruf-Pattern ist verankert.
func TestLogResultBuildsExpectedAttrs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	result := domain.Result{
		Name:     "StorageClassReady",
		Status:   domain.StatusFalse,
		Reason:   "StorageClassMissing",
		Severity: domain.SeverityCritical,
		Message:  "missing storage classes: fast",
	}
	application.LogResult(context.Background(), logger, slog.LevelInfo, "result emitted", result,
		slog.String("trace_id", "abc-123"),
	)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log entry: %v\n%s", err, buf.String())
	}
	want := map[string]string{
		"msg":      "result emitted",
		"check":    "StorageClassReady",
		"status":   "False",
		"reason":   "StorageClassMissing",
		"severity": "critical",
		"message":  "missing storage classes: fast",
		"trace_id": "abc-123",
	}
	for k, v := range want {
		if got, _ := entry[k].(string); got != v {
			t.Errorf("entry[%q]: got %q, want %q", k, got, v)
		}
	}
}

// TestLogResultHandlesNilLogger verifiziert, dass nil-Logger keine
// Panik auslöst — defensive Programmierung für Pfade ohne Logger
// (z. B. frühe Reconcile-Phase).
func TestLogResultHandlesNilLogger(t *testing.T) {
	t.Parallel()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("LogResult panicked on nil logger: %v", r)
		}
	}()
	application.LogResult(context.Background(), nil, slog.LevelInfo, "msg", domain.Result{})
}
