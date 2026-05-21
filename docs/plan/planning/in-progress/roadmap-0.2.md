# Roadmap — v0.2

**Status:** In Progress (Roadmap aktiviert 2026-05-21; M8 in Arbeit)
**Eröffnet:** 2026-05-21
**Aktiviert:** 2026-05-21
**Bezug:** [Lastenheft `LH-PRI-002`, `LH-REL-002`, `LH-VM-004`, `LH-VM-006`](../../../../spec/lastenheft.md),
[Architektur](../../../../spec/architecture.md),
[ADR 0001](../../adr/0001-dokumentations-und-planungsstruktur.md),
[ADR 0005](../../adr/0005-helm-chart-nicht-im-mvp.md),
[ADR 0007](../../adr/0007-prometheus-metrik-scope-im-mvp.md),
[ADR 0008](../../adr/0008-report-format-stack.md),
[ADR 0010](../../adr/0010-externe-dienstpruefungen-und-secret-mechanik.md),
[ADR 0014](../../adr/0014-v0.2-scope-schnitt.md)

---

## 0. Verhältnis zu Pflichtenheft und Architektur

Diese Roadmap legt die **Slice-Reihenfolge** für v0.2 fest — also das
**Was** in welcher Reihenfolge geliefert wird, nicht das **Wie** im
Detail.

- **`ADR 0014`** fixiert den v0.2-Scope-Schnitt (Voll-`LH-PRI-002` +
  `AR-OP-006`/OTel; `AR-OP-007`/Conversion-Webhook und `AR-OP-008`/
  Tenant-Isolation bleiben offen). Diese Roadmap löst den Scope-Schnitt
  operativ in Slices auf.
- **`spec/architecture.md`** trägt die strukturellen v0.2-Entscheidungen
  bereits (Layer-Modell, depguard-Regeln, RBAC-Erweiterungen für
  `core/events` und `core/configmaps` aus §10). Slice-spezifische
  CRD-Spec-Felder, Probe-Implementierungen und Test-Doubles entstehen
  pro Slice in `in-progress/slice-MX-…md`.
- Pro Slice entsteht beim Aktivieren ein eigener
  `in-progress/slice-MX-…md`-Plan mit Detail-Lieferzielen,
  Abnahmekriterien und Test-Schritten. Diese Roadmap wird nicht
  duplikativ.
- **Bewegungspfad dieser Datei:**
  `next/` (jetzt, Review-Phase) → `in-progress/` (mit Aktivierung des
  ersten Slices `M8`) → `done/` (mit
  `done/roadmap-0.2.md` als Closure, wenn alle M8–M16
  in `done/` liegen, analog zur abgeschlossenen v0.1-Roadmap).

---

## 1. Lieferziel der Roadmap

v0.2 (`LH-REL-002`) gemäß `LH-PRI-002` und `ADR 0014`:

- **Helm-Chart** als zusätzlicher Distributions-Pfad (`LH-NF-016`,
  `LH-SST-010`, `ADR 0005`).
- **Kubernetes-Events** bei Phasen-Übergängen (`LH-F-027`, `ADR 0008`).
- **ConfigMap-Report** mit YAML- und Markdown-Key (`LH-F-028`,
  `ADR 0008`).
- **Eigene Domänen-Metriken** über Prometheus-Format hinaus
  (`LH-NF-008`, `LH-SST-004`, `ADR 0007`).
- **OpenTelemetry-Tracing-Spans** im Reconcile-Pfad
  (`AR-OP-006`, `ADR 0007 §4`, `ADR 0014 §2.3`).
- **Node-Anzahl-/-Zustands-Prüfung** (`LH-F-016`, `LH-F-017`).
- **ClusterIssuer-Prüfung** über die cert-manager-Existenz-Prüfung
  aus v0.1 hinaus (`LH-F-014`).
- **DNS-Prüfung** (`LH-F-018`).
- **TLS-Reachability + Zertifikatsgültigkeit** (`LH-F-019`).
- **TCP-Network-Reachability ohne Auth** (`LH-F-022`, `ADR 0010 §2.2`).
- **Release-Tag `v0.2.0`** mit Trivy-Scan, Release-Guard und GHCR-
  Image — analog zur M7-Mechanik aus v0.1.

`AR-OP-007` (Conversion-Webhook), `AR-OP-008` (Tenant-Isolation), der
"mit-Auth"-Block aus `ADR 0010 §2.3` (PostgreSQL/S3) und alle
`LH-PRI-003`-Punkte (Plattformprofile, HTML-Report, kubectl-Plugin)
sind **nicht** Teil von v0.2 — siehe §4.

---

## 2. Slice-Übersicht

| Slice | Titel | Hauptlieferziel | Lastenheft-/AR-Kennungen |
| ----- | ----- | --------------- | ------------------------ |
| **M8** | Helm-Chart als Distributions-Pfad | `Chart.yaml`, `values.yaml`, Templates aus `deploy/manifests/`, Helm-Lint-Gate, Smoke-Install via `helm install` im Cluster-Smoke | `LH-NF-016`, `LH-SST-010`, `ADR 0005` |
| **M9** | Kubernetes-Events bei Phasen-Übergängen | Event-Recorder im Reconciler, Pflichtmessage-Snippets pro Phase, RBAC-Erweiterung `core/events` (`create`+`patch`), Sanitizer-Pass auf Event-Messages | `LH-F-027`, `ADR 0008` |
| **M10** | ConfigMap-Report | `status.reportRef` im CRD, ConfigMap mit YAML- und Markdown-Key, RBAC-Erweiterung `core/configmaps` (`get`/`list`/`create`/`update`/`patch`), Größen-/Truncation-Strategie | `LH-F-028`, `ADR 0008` |
| **M11** | Eigene Domänen-Metriken | Check-Counter, Phase-Gauge, Reconcile-Duration-Histogram unter `k_deskflight_*`-Namespace, OpenTelemetry-Instrumentations-Basis (Provider-Wiring) | `LH-NF-008`, `LH-SST-004`, `ADR 0007 §3` |
| **M12** | OpenTelemetry-Tracing-Spans | Tracer-Provider mit OTLP-Export, Spans für `Reconcile` und pro Check, Span-Attribute (CR-Name, Profile, Check-Name), Header-Propagation-Pflicht (aus Span-Context) | `AR-OP-006`, `ADR 0007 §4`, `ADR 0014 §2.3` |
| **M13** | Node + ClusterIssuer-Check | Node-Anzahl- und `Ready`-Status-Aggregation; ClusterIssuer-Existenz und `Ready`-Status (cert-manager-API-Gruppe) | `LH-F-014`, `LH-F-016`, `LH-F-017` |
| **M14** | DNS- und TLS-Check | DNS-Resolution-Check (A/AAAA, optional CNAME-Pfad); TLS-Handshake + Zertifikatsgültigkeit/-Ablauf; Trust-Bundle-Auswahl | `LH-F-018`, `LH-F-019` |
| **M15** | Network-Reachability ohne Auth | TCP-Connect-Check, Per-Endpoint-Timeout, Profil-Default-Endpoints; egress-NetworkPolicy-Annotation in Doku | `LH-F-022`, `ADR 0010 §2.2` |
| **M16** | Release-Tag `v0.2.0` | CHANGELOG-`[0.2.0]`-Section, `make release-guard VER=v0.2.0`, Trivy-Scan, GHCR-Image `ghcr.io/pt9912/k-deskflight:v0.2.0`, Helm-Chart-Publish (Distributions-Form aus M8-Slice), Roadmap-Sammel-Closure nach `done/` | `LH-REL-002`, `LH-AK-013..016` analog |

---

## 3. Slices im Detail

> Diese Detail-Sektion ist absichtlich schlanker gehalten als die
> M1–M7-Roadmap. Vollständige Slice-Pläne entstehen mit der Aktivierung
> des jeweiligen Slices unter `in-progress/slice-MX-…md`.

### M8 — Helm-Chart als Distributions-Pfad

**Lieferziel:** Helm-Chart unter `deploy/charts/k-deskflight/` mit
`Chart.yaml`, `values.yaml`, `values.schema.json` und Templates, die
aus den bestehenden `deploy/manifests/`-Manifesten abgeleitet sind.
Helm-Lint und Chart-Testing als zusätzliches Quality-Gate. Cluster-
Smoke wird um einen `helm install`-Pfad erweitert (parallel zum
bestehenden `kubectl apply -f deploy/manifests/`-Pfad).

**Eingangsabhängigkeit:** keine — Helm-Chart ist Distributions-Vehikel
für alle nachfolgenden Slices.

**Out-of-scope-Notizen:** Distributions-Form (Helm-Repository vs. OCI-
Registry über GHCR) entscheidet eine Folge-ADR im Slice-Plan
(`ADR 0005 §4` bereits angemerkt). Subchart-Pattern und Helm-Hook-
Pattern werden in M8 nicht verwendet, weil v0.2 keine Abhängigkeits-
charts und keine Pre-/Post-Install-Logik braucht.

### M9 — Kubernetes-Events bei Phasen-Übergängen

**Lieferziel:** `events.k8s.io/v1`-Event-Recorder (oder `corev1`-
Recorder via controller-runtime) wird im Reconciler-Wiring aktiviert.
Events feuern bei jedem Phasen-Übergang
(`Pending→Running→Passed|Warning|Failed|Unknown`), mit
sanitisierten Messages aus dem `Sanitize*`-Pfad von M5. RBAC wird um
`core/events` mit `create`+`patch` ergänzt (`spec/architecture.md §10`).

**Eingangsabhängigkeit:** keine. Events sind unabhängig vom Report.

**Out-of-scope-Notizen:** Event-Aggregation/Deduplication ist
Kubernetes-Server-seitig (Event-Pattern), nicht im Operator
zu lösen. Custom-Event-Reasons folgen dem
`Reason<Feature><State>`-Pattern aus `AR-OP-010`.

### M10 — ConfigMap-Report mit YAML- und Markdown-Key

**Lieferziel:** Neuer CRD-Status-Slot `status.reportRef` (NamespacedName
auf eine ConfigMap), `spec.report.enabled` als Opt-in-Flag. Der
Reconciler schreibt das `Status`-Snapshot serialisiert als YAML-Key
und gerendert als Markdown-Key in eine ConfigMap im Operator-
Namespace. RBAC wird um `core/configmaps` mit `get`/`list`/`create`/
`update`/`patch` ergänzt. Größen-Limit (1 MiB ConfigMap-Grenze) wird
durch ein Truncation-Pattern abgesichert.

**Eingangsabhängigkeit:** keine — ConfigMap-Report nutzt den Status-
Aggregator (`AR-014`) als Eingabe; der existiert seit M3.

**Out-of-scope-Notizen:** HTML-Report ist `LH-PRI-003` (v0.3+).
Report-Schema-Versionierung wird im Slice-Plan entschieden.

### M11 — Eigene Domänen-Metriken

**Lieferziel:** Eigene Prometheus-Metriken über die controller-runtime-
Defaults aus v0.1 hinaus:
- `k_deskflight_check_total{check, severity}` (Counter pro Check-Lauf)
- `k_deskflight_check_duration_seconds{check}` (Histogram)
- `k_deskflight_cr_phase{name, namespace, phase}` (Gauge mit
  One-Hot-Encoding der Phase)
- `k_deskflight_reconcile_total{result}` (Counter)

Implementation als OpenTelemetry-Meter-Provider; Prometheus-Exporter
exportiert über denselben `/metrics`-Endpoint wie heute. Damit wird
gleichzeitig die OTel-Instrumentations-Basis aufgebaut, auf die M12
aufsetzt.

**Eingangsabhängigkeit:** keine. M11 bereitet aber M12 vor.

**Out-of-scope-Notizen:** Custom-Histogram-Buckets werden im Slice-
Plan festgelegt. Auth-Filter für `/metrics`-Endpoint (v0.1: explizit
unauthenticated) wird in einer eigenen Folge-ADR adressiert
([`slice-M6-metrics-tests-doku.md`](../done/slice-M6-metrics-tests-doku.md) deferral).

### M12 — OpenTelemetry-Tracing-Spans im Reconcile-Pfad

**Lieferziel:** OTel-Tracer-Provider mit OTLP-gRPC-Exporter (Endpoint
über Env-Var `OTEL_EXPORTER_OTLP_ENDPOINT`, deaktiviert wenn leer).
Spans für `Reconcile`-Aufruf und pro Check-Ausführung; Standard-
Attribute (`k-deskflight.cr.name`, `k-deskflight.cr.namespace`,
`k-deskflight.profile`, `k-deskflight.check.name`,
`k-deskflight.check.severity`). Header-Propagation für
ausgehende Netzwerk-Calls (DNS/TLS/Reachability — M14/M15) folgt
der W3C-Trace-Context-Konvention.

**Eingangsabhängigkeit:** M11 (OTel-Provider-Basis).

**Out-of-scope-Notizen:** OTel-Backend-Wahl (Jaeger / Tempo / Grafana /
generischer OTLP-Collector) ist Anwender-Entscheidung; der Operator
wählt nicht. Sampling-Strategie (Always / Ratio / Parent-Based)
wird im Slice-Plan festgelegt.

### M13 — Node + ClusterIssuer-Check

**Lieferziel:** Zwei zusätzliche Checks unter `internal/adapter/check/`:
- `NodeCheck` — aggregiert `nodes.status.conditions[type=Ready]`
  über alle Nodes; konfigurierbare `minReady` und `minTotal`
  Mindestwerte über `spec.checks.nodes.{minReady,minTotal}`.
- `ClusterIssuerCheck` — listet `cert-manager.io/v1/ClusterIssuers`,
  prüft pro Eintrag `status.conditions[type=Ready]`. Konfigurierbar
  über `spec.checks.certManager.clusterIssuers[]` (Namens-Liste oder
  „alle vorhandenen").

Beide Checks erweitern die bestehende Check-Adapter-/Registry-Pattern-
Familie (M4) um zwei neue Interfaces. SAR-Selbstprüfung (`AR-018`)
ist obligatorisch.

**Eingangsabhängigkeit:** keine.

**Out-of-scope-Notizen:** Node-Taints/Labels-Filter (z. B. Control-
Plane-Ausschluss) ist Slice-Plan-Detail. Issuer-Typ-spezifische
Validierung (CA-Issuer, ACME-Issuer, …) wird nicht implementiert —
M13 prüft nur die generische `Ready`-Condition.

### M14 — DNS- und TLS-Check

**Lieferziel:** Zwei Checks unter `internal/adapter/check/`:
- `DNSCheck` — Resolution A/AAAA für eine Liste von Hostnamen aus
  `spec.checks.dns.hosts[]`; optionaler CNAME-Pfad-Trace; pro Host
  konfigurierbares Timeout.
- `TLSCheck` — TLS-Handshake gegen `host:port`-Endpunkte aus
  `spec.checks.tls.endpoints[]`; prüft Zertifikatsgültigkeit
  (NotBefore/NotAfter), Chain-Verifikation, optional SAN-Match.

Beide Checks laufen aus dem Operator-Pod heraus — egress-
NetworkPolicy-Annotation in `docs/user/`-Doku.

**Eingangsabhängigkeit:** keine. Optional M12 (Span-Propagation für
ausgehende DNS-/TLS-Calls).

**Out-of-scope-Notizen:** TLS-Mutual-Auth (Client-Zertifikate) ist
`mit-Auth`-Block (`ADR 0010 §2.3`) und damit v0.3+. DNSSEC-
Validierung ist Slice-Plan-Detail; Default: aus.

### M15 — Network-Reachability ohne Auth

**Lieferziel:** `ReachabilityCheck` unter `internal/adapter/check/` —
TCP-Connect gegen `host:port`-Endpunkte aus
`spec.checks.network.endpoints[]`; Per-Endpoint-Timeout und
konfigurierbare Retry-Anzahl. Profile-Defaults für Production
(z. B. `kubernetes.default.svc:443`, Default-Ingress-Controller-Endpoint)
und Evaluation (leer, vollständig deklarativ).

**Eingangsabhängigkeit:** keine. Egress-NetworkPolicy-Anmerkung
analog M14.

**Out-of-scope-Notizen:** HTTP-Reachability (volle HTTP-Response-
Validierung) ist Slice-Plan-Detail; M15 macht reine
TCP-Reachability. UDP-Reachability ist nicht im v0.2-Scope.
PostgreSQL- und S3-Reachability sind `mit-Auth`-Block (v0.3+).

### M16 — Release-Tag `v0.2.0`

**Lieferziel:** Analog zu Slice M7 aus v0.1:
- `CHANGELOG.md` erhält eine `[0.2.0]`-Section unterhalb
  `[Unreleased]`, gepflegt schrittweise aus den M8–M15-Closures.
- `make image-build VER=v0.2.0`, `make image-publish-dry-run VER=v0.2.0`,
  `make image-publish VER=v0.2.0`, `make image-scan VER=v0.2.0`,
  `make security-gates VER=v0.2.0`, `make release-guard VER=v0.2.0` —
  vollständig geerbt aus M7.
- Trivy-Scan gegen das v0.2.0-Image (CRITICAL/HIGH blockierend).
- GHCR-Publish des Images.
- Helm-Chart-Publish (Distributions-Form aus M8-Slice).
- Sammel-Closure dieser Roadmap-Datei: Move nach `done/`.

**Eingangsabhängigkeit:** M8–M15 alle in `done/`.

**Out-of-scope-Notizen:** Release-Announcement-Format (Blog/Mailingliste)
ist out-of-scope. ServiceMonitor-Stack-Integration für
Prometheus-Operator ist v0.3 (separate Folge-ADR).

---

## 4. Was nicht in v0.2

- **`AR-OP-007` Conversion-Webhook** — CRD bleibt `v1alpha1`
  (`ADR 0014 §2.4`). Sprung auf `v1alpha2`/`v1beta1` erst, wenn
  schemabreaking changes nicht mehr in `v1alpha1` additive ergänzt
  werden können (vermutlich v1.0 / `LH-REL-004`).
- **`AR-OP-008` Tenant-Isolation** — verschoben nach v0.3+
  (`ADR 0014 §2.5`). Kein konkreter Multi-Mandanten-Use-Case im
  aktuellen Lastenheft.
- **`ADR 0010 §2.3` mit-Auth-Block** — PostgreSQL- und S3-
  Reachability mit Credentials aus Secrets. Aktivierung in v0.3+
  via Folge-ADR (`ADR 0014 §2.6`).
- **`LH-PRI-003`-Punkte** — vordefinierte Plattformprofile,
  HTML-Report, kubectl-Plugin, OpenDesk-Integration.
- **Helm-Chart-Subchart-Pattern / Helm-Hooks** — M8 erbringt nur die
  Basis-Chart-Struktur.
- **OTel-Backend-Implementation** — der Operator exportiert OTLP, die
  Backend-Wahl ist Anwender-Sache (Jaeger/Tempo/Grafana/generic
  Collector).
- **HTTP-Response-Validierung** und **UDP-Reachability** — über
  TCP-Reachability hinausgehend; nicht in v0.2 (M15 §Out-of-scope).
- **DNSSEC-Validierung** — über generische DNS-Resolution hinausgehend;
  nicht in v0.2 (M14 §Out-of-scope).
- **Auth-Filter für `/metrics`-Endpoint** — eigene Folge-ADR zu
  `ADR 0007`, deferred aus M6.

---

## 5. Meilenstein-Marker für `LH-VM-006`-Traceability

| `LH-PRI-002`-Punkt | Slice |
| ------------------ | ----- |
| `LH-F-018` (DNS) | M14 |
| `LH-F-019` (TLS) | M14 |
| `LH-F-022` (Reachability ohne Auth) | M15 |
| `LH-F-016` (Node-Anzahl) | M13 |
| `LH-F-017` (Node-Zustand) | M13 |
| `LH-F-014` (ClusterIssuer) | M13 |
| `LH-F-027` (Events) | M9 |
| `LH-F-028` (ConfigMap-Report) | M10 |
| `LH-NF-008` / `LH-SST-004` (Domänen-Metriken) | M11 |
| `LH-NF-016` / `LH-SST-010` (Helm-Chart) | M8 |
| `AR-OP-006` (OTel-Tracing) | M12 (via `ADR 0014 §2.3`) |
| `LH-REL-002` (Release-Tag v0.2.0) | M16 |

Neue `LH-AK-*`-Abnahmekriterien für v0.2 (z. B. ein Helm-Install-
Abnahme-Item, ConfigMap-Report-Abnahme-Item, OTel-Tracing-Abnahme-Item)
entstehen mit dem jeweiligen Slice-Plan und werden im selben Commit
ins Lastenheft eingepflegt.

---

## 6. Status-Tracking

Pro Slice eine Statuszeile im Slice-Plan-Header (`**Status:** Geplant
| In Arbeit | Done`). Diese Roadmap-Tabelle in §2 wird beim
Slice-Abschluss mit einem Verweis auf die Closure-Notiz unter `done/`
angereichert (analog zur abgeschlossenen v0.1-Roadmap).

---

## 7. Status

**Aktueller Stand (2026-05-21):** **In Progress.** Slice-Schnitte
M8–M16 nach Review aktiviert; die Datei ist von `planning/next/` nach
`planning/in-progress/` gewandert. Der erste Slice
[`slice-M8-helm-chart.md`](slice-M8-helm-chart.md) ist eröffnet und
in Arbeit. Folge-Slices M9–M16 öffnen sich der Reihe nach mit der
M8-Closure bzw. abhängigkeitsorientiert (siehe §3 pro Slice
„Eingangsabhängigkeit").
