# Roadmap — MVP v0.1

**Status:** In Progress (Roadmap aktiv; Slices noch nicht gestartet)
**Eröffnet:** 2026-05-16
**Bezug:** [Lastenheft `LH-MVP-002`, `LH-PRI-001`, `LH-REL-001`, `LH-VM-004`, `LH-QG-001..011`](../../../../spec/lastenheft.md),
[ADR 0001](../../adr/0001-dokumentations-und-planungsstruktur.md),
[ADRs 0004 – 0012](../../adr/) (alle MVP-relevanten Architekturentscheidungen inkl. Quality-Gates)

---

## 0. Verhältnis zu Pflichtenheft und Architektur

Diese Roadmap legt die **Slice-Reihenfolge** für den MVP fest — also das
**Was** in welcher Reihenfolge geliefert wird, nicht das **Wie** im
Detail.

- **`spec/architecture.md`** ist angelegt (`AR-*`-Kennungsraum,
  Paket-Layout-Hybrid, Modulpfad `github.com/pt9912/k-deskflight`,
  Layer-Modell, konkrete `depguard`-Regeln gemäß `LH-QG-004`,
  Reconciler-/Check-/RBAC-Skizze, Build-Pipeline-Anker). Damit ist
  `ADR 0001 §3` („spätestens vor der ersten Implementierungs-Slice")
  für die strukturellen Architekturentscheidungen erfüllt; M1 ist
  aktivierbar. Das **Pflichtenheft** (`LH-VM-002`) im engeren
  V-Modell-Sinn — exakte CRD-Spec-Feld-Typen, Test-Mock-Strategie,
  konkrete Schwellen, Probe-Timeouts — entsteht mit dem M1-Slice-Plan
  bzw. in einer eigenen `spec/pflichtenheft.md`; offene Punkte sind
  in `architecture.md` als `AR-OP-*` geführt.
- Pro Slice entsteht beim Aktivieren ein eigener
  `in-progress/slice-MX-…md`-Plan mit Detail-Lieferzielen,
  Abnahmekriterien und Test-Schritten. Diese Roadmap wird nicht
  duplikativ.
- Wenn alle M1 – M7 Slices in `done/` liegen, wandert diese Roadmap als
  Closure-Notiz ebenfalls nach `done/`.

---

## 1. Lieferziel der Roadmap

MVP v0.1 (`LH-REL-001`) gemäß `LH-MVP-002` und `LH-PRI-001`:

- CRD `OpenDeskPreflightCheck` (API-Gruppe `k-deskflight.geo-terrain.net/v1alpha1`,
  namespaced, gemäß `ADR 0006`).
- Controller, ausschließlich lesend (`LH-F-035`).
- MVP-Prüfungen: Kubernetes-Version, StorageClass, IngressClass,
  cert-manager-Vorhandensein, Cluster-Ressourcen, RBAC-Selbstprüfung
  (`LH-F-024`).
- Status-Conditions, Gesamtphase, Zusammenfassung.
- Prometheus-`/metrics`-Endpoint mit Framework-Defaults (`ADR 0007`).
- Raw-Manifeste in `deploy/manifests/` (kein Helm Chart, `ADR 0005`).
- Container-Image, Beispielmanifest, README-Stand passend zum Release.
- Quality-Gates aktiv gemäß `ADR 0012` und `LH-QG-001..011`
  (Linting, Tests, Coverage 90 %, Architektur-Boundary,
  Generated-Drift, Vulnerability- und Image-Scan, Doc-Refs).

---

## 2. Slice-Übersicht

| Slice | Titel | Vorgänger | Status |
| ----- | ----- | --------- | ------ |
| M1 | Repo & Build-Skeleton | — | Done (Closure: [`done/slice-M1-repo-skeleton.md`](../done/slice-M1-repo-skeleton.md)) |
| M2 | CRD + Controller-Skeleton | M1 | Pending |
| M3 | Erste Prüfung — Kubernetes-Version | M2 | Pending |
| M4 | Cluster-State-Prüfungen (Storage, Ingress, cert-manager, Ressourcen) | M2 (kann mit M3 parallel) | Pending |
| M5 | RBAC-Selbstprüfung & Robustheit | M3 + M4 | Pending |
| M6 | Metrics-Endpoint, Tests, Doku | M5 | Pending |
| M7 | Beispielmanifest, Release-Tag v0.1.0 | M6 | Pending |

Abhängigkeitsgraph: `M1 → M2 → {M3, M4} → M5 → M6 → M7`.

---

## 3. Slices im Detail

### M1 — Repo & Build-Skeleton

**Lieferziel:** Go-Projektskelett im Repository, lokale Bau- und
Test-Befehle, Container-Image baut leer, CI-Pipeline-Stub,
Pflicht-Quality-Gates auf Skelett-Ebene aktiv.

- Go-Modulpfad final entscheiden (Folge zu `ADR 0004 §4`),
  `go.mod`, Verzeichnis-Layout (`cmd/`, `internal/`, `api/`, `deploy/`).
- `Makefile` mit Targets `build`/`lint`/`test`/`image-build` plus
  Bündel-Targets `gates` und `security-gates` (`LH-QG-009`,
  m-trace-Pattern als Vorlage — siehe Memory-Hinweis unten).
- `golangci-lint`-Konfiguration `.golangci.yml` mit den fünf
  Default-Lintern und dem 24-er SOLID-nahen Profil gemäß
  `ADR 0012 §2.2` und `LH-QG-001`.
- `scripts/verify-doc-refs.sh` (adaptiert von m-trace, gemäß
  `ADR 0012 §2.10` und `LH-QG-008`) als Pflicht-Gate für
  Markdown-Querverweise — Geltungsbereich: `docs/`, `spec/`,
  `README.md`, `README.de.md`, `CONTRIBUTING.md`,
  `CODE_OF_CONDUCT.md`, `SECURITY.md`. Schließt zugleich den
  `ADR 0003 §5`-Folge-Trigger.
- Multi-Stage `Dockerfile` (`distroless` oder vergleichbares Base);
  eigene `lint`-Stage analog `m-trace/apps/api/Dockerfile`.
- CI-Workflow-Skelett (GitHub Actions; passt zu `ADR 0011` GHSA-Pfad)
  mit zwei Jobs: `gates` und `security-gates` parallel.
- DCO-Bot-Aktivierung als Folgepflicht (`ADR 0011 §2.4`).
- `deploy/manifests/`-Verzeichnis angelegt (initial leer oder
  CRD-Stub als Platzhalter; siehe M2).
- README-Stand bleibt zur M1-Aktivierung der aktuelle `main`-Stand;
  Roadmap- und ADR-Querverweise im README sind konsistent zu halten,
  legitime spätere Edits (Tippfehler, neue ADR-Aufnahmen) zwingen
  zu keiner Roadmap-Änderung.

**Memory-Hinweis (intern, für Maintainer):** Die m-trace-Quelldateien
(`/Development/m-trace/Makefile`, `apps/api/.golangci.yml`,
`apps/api/Dockerfile`, `scripts/verify-doc-refs.sh`, `docs/user/quality.md`)
sind die Vorlagen. Für externe Mitwirkende sind die Vorlagen nicht
sichtbar; das M1-Slice-Plan-Dokument (entsteht beim Aktivieren) führt
die adaptierten Inhalte direkt im k-deskflight-Repository.

**Lastenheft-Kennungen:** `LH-NF-001` (Go), `LH-NF-014` (lokale
Entwicklung), `LH-NF-015` (Container-Image), `LH-NF-021` (Sprachregel),
`LH-NF-019` (Ressourcenverbrauch — Konzept), `LH-PROD-001` (Naming),
`LH-QG-001` (Linting), `LH-QG-008` (Doc-Refs), `LH-QG-009`
(Gate-Bündelung), `LH-QG-010` (Suppressions-Verbot).

**Architekturartefakte:** Pflichtenheft entsteht parallel oder voraus;
mindestens Paketstruktur und Go-Modulpfad müssen festliegen, bevor
Slice abgeschlossen wird.

**Verifikation:**

- `make build` baut ohne Fehler.
- `make image-build` produziert ein laufendes Image (Init/Help-Pfad
  startet).
- `make lint` und `make test` laufen grün (auch wenn noch nichts
  testet — Skelett-Smoketests).
- CI-Workflow-Run auf einem Beispiel-PR ist grün.

**Verifikationspfad:** kein dediziertes `LH-AK-*` zu M1 selbst;
M1 ist die Voraussetzung für alle weiteren `LH-AK-*` ab M2.

---

### M2 — CRD + Controller-Skeleton

**Lieferziel:** CRD `OpenDeskPreflightCheck` (`v1alpha1`, namespaced,
gemäß `ADR 0006`), Controller-Reconciler-Stub, Status-Schema mit
Conditions und Phase, kein Prüflogik-Inhalt; Generated-Drift- und
depguard-Boundary-Gates aktiv.

- CRD-Schema mit `spec.profile`, `spec.checks.kubernetesVersion`
  (Platzhalter), `status.phase`, `status.summary`,
  `status.conditions` (`LH-F-005`/`LH-F-006`/`LH-F-007`).
- **Profile-Default-Vorbelegung** im CRD-OpenAPI-Schema:
  `spec.profile` Default `production`, `kubernetesVersion.min`
  Default gemäß `ADR 0009 §2.2` (heute `"1.34"`). `LH-PROF-002`/
  `LH-PROF-003` werden durch die Schema-Defaults vorbereitet, die
  fachliche Auswertung der Defaults pro Profile passiert in M4.
- ServiceAccount, ClusterRole, RoleBinding (lesend) im
  `deploy/manifests/`-Set, gemäß `LH-AK-015`, `LH-NF-006` und
  `ADR 0005`.
- Reconciler reagiert auf CR-Anlage, schreibt `status.phase = Pending`
  → `Running` → `Passed` (mit leerer Summary, weil noch keine
  Prüfung), Conditions: leer.
- **Generated-Drift-Gate aktiv**: `controller-gen`-Output (CRD-YAML,
  DeepCopy) gegen Git-Stand prüfen (`LH-QG-005`).
- **depguard-Regeln im Linter-Profil** (Stub): erste Layer-
  Definition gemäß Pflichtenheft/`architecture.md` (`LH-QG-004`).
  Strikte Durchsetzung kommt in M6.
- **Coverage-Range-Selektor und `scripts/coverage-gate.sh` anlegen**
  (`LH-QG-003`, Adaption von `m-trace/apps/api/scripts/
  coverage-gate.sh`): Selektor schließt `cmd/`-Wiring aus und
  zielt auf `internal/`-Pakete (konkrete Pfade gemäß Pflichtenheft).
  Skript liest das Total-Line-Cover-Ergebnis und exited mit 1 bei
  Unterschreitung der Schwelle. In M2 ist das Skript bereits
  vorhanden und mit niedriger Schwelle (z. B. 0 %) als
  Smoketest-Pfad aktiv; M6 hebt die Schwelle auf 90 % und macht
  die Verletzung PR-blockierend.
- Beispielmanifest minimal (CR ohne Inhalt außer Name).

**Lastenheft-Kennungen:** `LH-F-001`, `LH-F-002`, `LH-F-003`,
`LH-F-004`, `LH-F-005`, `LH-F-006`, `LH-F-007`, `LH-F-009`
(API-Erreichbarkeit), `LH-F-035` (lesender Betrieb),
`LH-NF-002` (Kubernetes-Konventionen), `LH-NF-004` (Stabilität),
`LH-NF-006` (Minimalrechte-Konzept), `LH-PROD-002` (API-Gruppe),
`LH-PROF-002`/`LH-PROF-003` (Schema-Defaults), `LH-AK-015` (RBAC),
`LH-DAT-002` (Status-Speicherung), `LH-QG-003` (Coverage-Skript +
Range-Selektor, Schwelle in M6), `LH-QG-004` (Boundary-Stub),
`LH-QG-005` (Generated-Drift).

**Verifikation:**

- `LH-AK-001` — CRD installierbar.
- `LH-AK-002` — Operator startbar.
- `LH-AK-003` — Ressource verarbeitbar.
- `LH-AK-004` — Status sichtbar.
- `LH-AK-011` — Conditions vorhanden (auch wenn leer ist die Struktur
  da).

---

### M3 — Erste Prüfung: Kubernetes-Version

**Lieferziel:** Erste konkrete Check-Implementierung — Kubernetes-Version
gegen die in `spec.checks.kubernetesVersion.min` konfigurierte
Mindestversion (`ADR 0009`).

- Versions-Discovery via `discovery.ServerVersion()`.
- Vergleich gegen konfigurierte Mindestversion (Default-Vorbelegung
  per Profile, `ADR 0009 §2.2`).
- Condition `KubernetesVersionReady` (true/false), Phase, Schweregrad
  `critical` bei Fail.
- `status.summary.passed`/`failed`/`warning`/`lastChecked`.

**Lastenheft-Kennungen:** `LH-F-008`, `LH-F-031` (Schweregrad),
`LH-F-032` (Ergebnis-Inhalt), `LH-NF-003` (Nachvollziehbarkeit),
`LH-DAT-003` (Zeitstempel), `LH-QA-001` (verständliche Fehlermeldungen).

**Verifikation:**

- `LH-AK-005` — K8s-Version prüfbar.
- Tests: passed-Case auf aktueller Version, failed-Case mit
  konfigurierter Min `99.99` (synthetisch).

---

### M4 — Cluster-State-Prüfungen

**Lieferziel:** Vier weitere Prüfungen — StorageClass, IngressClass,
cert-manager, Cluster-Ressourcen.

- StorageClass (`LH-F-010`, `LH-F-011`): konfigurierte Klassen
  vorhanden? Default-StorageClass erkennbar?
- IngressClass (`LH-F-012`): konfigurierte Klasse vorhanden?
- cert-manager (`LH-F-013`): API-Gruppe `cert-manager.io` vorhanden,
  mindestens ein `ClusterIssuer` erreichbar? (Vorhandensein, nicht
  Detail-Validierung — die kommt v0.2 mit `LH-F-014`.)
- Cluster-Ressourcen (`LH-F-015`): allocatable CPU/Memory aller
  Ready-Nodes summieren, gegen konfigurierte Mindestwerte prüfen
  (`LH-AK-009`).
- Jede Prüfung mit eigener Condition und Severity.

**Lastenheft-Kennungen:** `LH-F-010`, `LH-F-011`, `LH-F-012`,
`LH-F-013`, `LH-F-015`, `LH-NF-005` (Fehlertoleranz —
Einzelausfall darf andere nicht stoppen), `LH-PROF-002`/`-003`
(Profile bestimmen Default-Werte).

**Verifikation:**

- `LH-AK-006` — StorageClass.
- `LH-AK-007` — IngressClass.
- `LH-AK-008` — cert-manager.
- `LH-AK-009` — Ressourcen.

---

### M5 — RBAC-Selbstprüfung & Robustheit

**Lieferziel:** Operator prüft eigene Berechtigungen und bleibt bei
Einzelfehlern stabil.

- `SelfSubjectAccessReview`/`SelfSubjectRulesReview` pro aktivierter
  Prüfung (`LH-F-024`).
- Condition `RBACInsufficient` falls Recht fehlt; betroffene
  Einzelprüfung wird `Unknown` (`LH-AK-016`).
- Fehlertoleranz: panic-Rückfänger im Reconcile-Loop, einzelne
  Check-Fehler erzeugen `Unknown`, nicht Abbruch (`LH-NF-005`).
- Secret-Output-Filter aktiv (`LH-SEC-002`, `LH-NF-007`) — Tests
  prüfen, dass kein Secret-Inhalt in Logs/Events/Status landet.
  Im MVP gibt es noch keine externen Secrets (`ADR 0010`),
  aber der Filter ist als Pflicht-Konvention verankert.
- Keine destruktiven Aktionen (`LH-SEC-005`).

**Lastenheft-Kennungen:** `LH-F-024`, `LH-F-031`, `LH-NF-004`,
`LH-NF-005`, `LH-NF-006` (minimal notwendige Berechtigungen —
operative Verankerung), `LH-SEC-001`, `LH-SEC-002`, `LH-SEC-005`,
`LH-DAT-007` (Konvention vorbereitet, auch ohne aktiven Use),
`LH-NF-007` (Datenschutz).

**Verifikation:**

- `LH-AK-010` — Fehlerfall robust.
- `LH-AK-012` — Keine Secret-Leaks (im MVP trivial, weil keine
  externen Secrets; Tests prüfen den Filter trotzdem).
- `LH-AK-015` — Minimalrechte dokumentiert.
- `LH-AK-016` — RBAC-Selbstprüfung wirksam.

---

### M6 — Metrics-Endpoint, Tests, Doku

**Lieferziel:** Prometheus-`/metrics`-Endpoint mit
controller-runtime-Defaults (`ADR 0007`), Integrationstests gegen
einen lokalen kind-/envtest-Cluster, vollständige Anwender-Doku,
alle MVP-Pflicht-Quality-Gates strikt grün.

- `/metrics`-Endpoint exposed, ServiceAccount-RBAC für Scrape
  passend, Endpoint im Smoketest erreichbar.
- Integrationstests (kind oder envtest): jeder MVP-Check hat einen
  passed- und einen failed-Case.
- Anwender-Doku in `docs/user/` ausarbeiten:
  - Installation (raw manifests).
  - CR-Beispiele für `evaluation` und `production`.
  - Conditions-Katalog mit Reason/Severity.
  - Troubleshooting (typische Fehlerbilder).
- **Coverage-Gate strikt** auf 90 % Line-Coverage über produktive
  Pakete (`LH-QG-003`); `make coverage-gate` blockt PRs.
- **`govulncheck` strikt** als Pflicht-Gate (`LH-QG-006`); funktions-
  basiertes Scanning gegen Go-Vulnerability-Datenbank.
- **Architektur-Boundary strikt**: alle `depguard`-Regeln aus M2
  sind aktiv und brechen den Build bei Layer-Verletzung
  (`LH-QG-004`).

**Lastenheft-Kennungen:** `LH-SST-004` (Prometheus-Format),
`LH-NF-008` (`/metrics` als Endpoint, eigene Domänen-Metriken
folgen v0.2), `LH-NF-010` (Testbarkeit), `LH-NF-013`
(Dokumentation), `LH-QA-002` (reproduzierbare Ergebnisse),
`LH-QA-004` (transparente Bewertung), `LH-QG-002` (Tests),
`LH-QG-003` (Coverage), `LH-QG-004` (Boundary strikt),
`LH-QG-006` (Vulnerability-Scan).

**Verifikation:**

- `LH-AK-013` — Dokumentation vorhanden.
- Coverage-Gate grün bei Default-Threshold 90 % (`ADR 0012 §2.5`).
- `govulncheck`-Lauf ohne Treffer in aufgerufenen Funktionen.
- Architektur-Boundary-Check ohne `depguard`-Verletzung.
- Smoketest `/metrics`-Endpoint liefert HTTP 200 mit
  Prometheus-Format.

---

### M7 — Beispielmanifest, Release-Tag v0.1.0

**Lieferziel:** Vollständige Beispielmanifeste, Release-Notes,
v0.1.0-Tag, Container-Image-Publish auf GHCR; Image-Scan-Gate als
letzte Release-Pflicht.

- Beispielmanifeste konsistent mit `LH-PROD-003a` (MVP-Profil).
- Release-Notes pro `ADR 0011 §2.5`: SemVer-Tag, Inhalt verlinkt.
- CHANGELOG-Entscheidung umsetzen (`planning/open/changelog.md`)
  vor dem Tag.
- Container-Image-Publish via `make image-publish`-Pattern aus
  m-trace (Approval-Gate, `ADR 0011 §2.5`).
- **Trivy Image-Scan** vor Tag (`LH-QG-007`): `CRITICAL`/`HIGH`
  brechen Release; `MEDIUM` wird in den Release-Notes berichtet.
  Vulnignore-Einträge (falls vorhanden) tragen `expires`-Datum.
- v0.1.0-Tag setzen.
- DCO-Compliance-Check vor Merge der Release-PR.

**Lastenheft-Kennungen:** `LH-REL-001` (Version 0.1), `LH-MVP-002`
(Vollständigkeit), `LH-AK-014` (Open-Source-Veröffentlichung
möglich — schließt jetzt vollständig, inkl. README/CONTRIBUTING/
CODE_OF_CONDUCT/SECURITY und CHANGELOG), `LH-QG-007` (Image-Scan),
`LH-QG-009` (`make security-gates`).

**Verifikation:**

- v0.1.0-Tag existiert, GHSA-Pfad ist aktiv (Repo öffentlich).
- Alle `LH-AK-001..016` erfüllt (Traceability-Matrix §20 grün).
- Trivy-Scan zeigt keinen ungeklärten `CRITICAL`/`HIGH`-Fund.
- Container-Image `ghcr.io/<owner>/k-deskflight:v0.1.0` läuft auf
  einer der drei aktuellen K8s-Versionen aus `ADR 0009`.

---

## 4. Was nicht im MVP

Diese Roadmap deckt **ausschließlich** den MVP-Scope. Folgende Inhalte
gehören nicht in M1–M7 und kommen in späteren Roadmaps:

- DNS / TLS / Netzwerk-Reachability (`LH-F-018`, `LH-F-019`,
  `LH-F-022`) — v0.2 (`ADR 0010`).
- ClusterIssuer-Detailprüfung (`LH-F-014`) — v0.2.
- Node-Anzahl- und Zustandsprüfung (`LH-F-016`, `LH-F-017`) — v0.2.
- Events (`LH-F-027`) — v0.2.
- ConfigMap-Report (`LH-F-028`) — v0.2.
- Eigene Domänen-Metriken (`LH-NF-008` voll) — v0.2.
- Helm Chart (`LH-NF-016`, `LH-SST-010`) — v0.2 (`ADR 0005`).
- PostgreSQL- und S3-Erreichbarkeit (`LH-F-020`, `LH-F-021`) — v0.3+
  (`ADR 0010` + Folge-ADR, Trigger
  `docs/plan/planning/open/external-services-v03-activation.md`).
- HTML-Report, kubectl-Plugin, Plattformprofile (`k3s`/`scs`/
  `airgapped`/`custom`) — spätere Versionen (`LH-PRI-003`,
  `LH-PROF-001`/`-004`).
- Mandantenverwaltung, Backup-Orchestrierung, Upgrade-Orchestrierung
  — nicht im Produktscope (`LH-MVP-003`, `LH-SYS-002..005`).

---

## 5. Meilenstein-Marker für `LH-VM-006`-Traceability

`LH-VM-006` fordert: „Jede wesentliche Implementierungsfunktion soll auf
mindestens eine Lastenheftkennung zurückführbar sein." Die Slices stellen
diese Rückführbarkeit her:

| Marker | Slice | Bedeutung |
| ------ | ----- | --------- |
| M1 | Repo bereit | Build- und CI-Skelett, kein `LH-AK-*` direkt |
| M2 | CRD installierbar | `LH-AK-001`, `-002`, `-003`, `-004`, `-011` |
| M3 | Erste Prüfung lauffähig | `LH-AK-005` |
| M4 | Alle Cluster-State-Prüfungen | `LH-AK-006`, `-007`, `-008`, `-009` |
| M5 | Robustheit & RBAC | `LH-AK-010`, `-012`, `-015`, `-016` |
| M6 | Doku & Tests | `LH-AK-013` |
| M7 | v0.1.0-Release | `LH-AK-014`; Traceability-Matrix vollständig grün |

---

## 6. Status-Tracking

Slice-Status werden in dieser Datei aktualisiert, sobald ein Slice von
`Pending` nach `In Progress` und schließlich nach `Done` wandert. Pro
abgeschlossenem Slice entsteht eine Closure-Notiz in
`docs/plan/planning/done/slice-MX-…md` mit Lieferzielen-Abgleich und
Verifikations-Ergebnis. Mit M7-Abschluss wandert auch diese Roadmap als
Sammel-Closure-Notiz nach `done/`.

---

## 7. Status

In Progress. M1 geschlossen am 2026-05-17 (Closure-Notiz unter
[`done/slice-M1-repo-skeleton.md`](../done/slice-M1-repo-skeleton.md));
M2–M7 weiterhin Pending. Aktivierung von M2 erfolgt durch einen eigenen
`slice-M2-…md` in `in-progress/`.
