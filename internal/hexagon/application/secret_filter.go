/*
Copyright 2026 The k-deskflight Authors.

Licensed under the MIT License (see LICENSE at the repository root).
*/

package application

import (
	"context"
	"log/slog"

	"github.com/pt9912/k-deskflight/internal/hexagon/domain"
)

// SanitizeMessage ist der String-Pfad-Hook für vertrauliche
// Klartext-Inhalte in Status/Event/Log-Messages (slice-M5 §2.6).
//
// In M5 ist die Implementierung die Identitätsfunktion — der MVP
// kennt noch keine externen Secrets (`ADR 0010`). v0.2 ersetzt die
// Identität durch eine echte Pattern-Maskierung (Bearer-Token,
// Base64-Blobs, …); das Aufruf-Pattern muss bis dahin in allen
// Status-/Event-/Log-Pfaden lückenlos verankert sein.
//
// **Pflicht-Aufrufstellen** (slice-M5 §2.6-Tabelle):
//   - vor jedem `Status().Update` auf `Condition.Message` und
//     `Summary`-Feldern;
//   - vor jedem K8s-Event (post-MVP).
func SanitizeMessage(msg string) string {
	return msg
}

// SanitizeAttrs ist der Attribut-Pfad-Hook für strukturierte
// `slog`-Aufrufe (slice-M5 §2.6, Folge-Review-Befund 3). Reine
// String-Sanitization über `SanitizeMessage` greift nicht bei
// `slog.Any("key", val)`-Attrs — vertrauliche Werte könnten am
// Filter vorbei in den strukturierten Logger-Output landen.
//
// In M5 ist die Implementierung die Identitätsfunktion; das
// Aufruf-Pattern verankert die Stelle für v0.2.
//
// **Pflicht-Aufrufstellen**:
//   - Panic-Recovery-`slog.Any("recover", r)` / `slog.Any("stack", …)`
//     in `runner.go` und `Reconcile`;
//   - SAR-Error-Logger in `runner.go` (RBACCheckFailed-Pfad);
//   - Per-Result-Summary-Log am Ende von `Reconcile` über `LogResult`.
func SanitizeAttrs(attrs ...slog.Attr) []slog.Attr {
	out := make([]slog.Attr, len(attrs))
	copy(out, attrs)
	return out
}

// LogResult ist der Pflicht-Wrapper für jeden Log-Call, der ein
// `domain.Result` oder daraus abgeleitete Daten ans Logging weiterreicht
// (slice-M5 §2.6). Kombiniert beide Sanitize-Hooks: `msg` wird
// sanitisiert, Result-Felder werden als `slog.Attr` gebaut und durch
// `SanitizeAttrs` geschickt, danach ruft die Funktion
// `logger.LogAttrs`.
//
// Caller muss einen `ctx` übergeben (`context.Background()` oder
// `context.TODO()` sind explizit erlaubt); ein `nil`-Logger wird
// stillschweigend ignoriert, damit frühe Reconcile-Pfade ohne
// instantiierten Logger nicht panicken.
//
// Nutzung: in `Reconciler.Reconcile` für die Per-Result-Summary, in
// `runner.go` für RBAC-/Timeout-/Panic-Diagnose-Logs.
func LogResult(
	ctx context.Context,
	logger *slog.Logger,
	level slog.Level,
	msg string,
	result domain.Result,
	extra ...slog.Attr,
) {
	if logger == nil {
		return
	}
	resultAttrs := []slog.Attr{
		slog.String("check", result.Name),
		slog.String("status", string(result.Status)),
		slog.String("reason", result.Reason),
		slog.String("severity", string(result.Severity)),
		slog.String("message", SanitizeMessage(result.Message)),
	}
	resultAttrs = append(resultAttrs, extra...)
	logger.LogAttrs(ctx, level, SanitizeMessage(msg), SanitizeAttrs(resultAttrs...)...)
}
