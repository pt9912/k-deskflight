# ADR 0012 — Quality-Gates

**Status:** Accepted
**Datum:** 2026-05-16
**Bezug:** [Lastenheft](../../../spec/lastenheft.md),
[ADR 0001](0001-dokumentations-und-planungsstruktur.md),
[ADR 0002](0002-adr-lifecycle.md),
[ADR 0003](0003-kennungsbasierte-querverweise.md),
[ADR 0005](0005-helm-chart-nicht-im-mvp.md),
[ADR 0006](0006-api-gruppe-und-crd-scope.md),
[ADR 0007](0007-prometheus-metrik-scope-im-mvp.md),
[ADR 0011](0011-governance-und-beitragskonventionen.md)

---

## 1. Kontext

Bisher war die Quality-Gate-Politik im Projekt nicht festgelegt:
`LH-NF-010` (Testbarkeit), `LH-NF-011` (Erweiterbarkeit) und
`LH-NF-012` (Wartbarkeit / modular) sowie `LH-QA-001..006` benennen
Qualitätsanforderungen, operationalisieren sie aber nicht.
`LH-MVP-002` zählt Tests und Doku unter den MVP-Funktionen auf, ohne
einen Linter- oder Coverage-Vertrag. Die Roadmap (`docs/plan/planning/
in-progress/roadmap.md`) erwähnt unter M6 lediglich „Coverage-Gate
für Unit- und Integrations-Tests grün (Konkrete Schwelle entsteht im
Pflichtenheft)".

Das Schwesterprojekt `/Development/m-trace` (siehe Memory
`reference_m_trace_patterns.md`) hat einen ausgereiften
Quality-Gate-Stack: `lint`, `test`, `coverage-gate`, `arch-check`,
`generated-drift-check`, `vuln-check`, `image-scan`, `docs-check`,
plus opt-in `mutation-report`, `fuzz-check`, `benchmark-smoke`. Die
`golangci-lint`-Konfiguration in `m-trace/apps/api/.golangci.yml`
verbindet fünf Default-Linter mit einem 24-er SOLID-nahen Profil und
schließt `//nolint`-Suppressions aus.

`SOLID` ist eine **Design-Linie**, kein automatisch messbares Gate.
Diese ADR nähert SOLID-Compliance über drei orthogonale Mechaniken
an, die zusammen das fachliche Ziel tragen:

1. Architektur-Boundary-Check (`depguard`) — bedient *Dependency
   Inversion* und *Single Responsibility* auf Paket-Ebene.
2. Komplexitäts- und Größenlinter (`cyclop`, `gocognit`, `funlen`,
   `nestif`, `maintidx`, `dupl`) — approximieren *Single
   Responsibility* und *Open/Closed* auf Funktions-/Modul-Ebene.
3. Code-Review-Pflicht aus `ADR 0011 §2.7` — einziger Ort, an dem
   *Liskov Substitution* und *Interface Segregation* fundiert
   geprüft werden; keine Statik-Analyse leistet das.

Diese ADR adaptiert das m-trace-Pattern weitgehend 1:1 für die
Go-Seite des k-deskflight-Operators. TypeScript-/Svelte-Anteile aus
m-trace entfallen mangels Workspace.

---

## 2. Entscheidung

### 2.1 Gate-Kategorien-Übersicht

| Kategorie | Tool / Target | Aktivierungs-Slice | Pflicht? | LH-Anker |
| --------- | ------------- | ------------------ | -------- | -------- |
| Statisches Linting | `golangci-lint` v2 mit SOLID-nahem Profil (§2.2) | M1 | ja | `LH-QG-001`, `LH-NF-012` |
| Unit + Integrationstests | `go test`, `envtest`/`kind` | M2 (Skelett) → M6 (vollständig) | ja | `LH-QG-002`, `LH-NF-010` |
| Coverage-Gate | `go test -cover` mit Schwelle (§2.5) | M6 | ja | `LH-QG-003`, `LH-NF-010` |
| Architektur-Boundary | `depguard` als Linter-Regel im `golangci-lint`-Profil | M2 (Regeln angelegt) → M6 (strikt) | ja | `LH-QG-004`, `LH-NF-011`, `LH-NF-012` |
| Generated-Drift | `controller-gen` (CRD-YAML, DeepCopy) und git-diff-Check | M2 | ja | `LH-QG-005`, `LH-NF-002` |
| Vulnerability-Scan | `govulncheck` (Go-Vuln-DB, function-based) | M6 | ja | `LH-QG-006`, `LH-NF-006`, `LH-SEC-001` |
| Image-Scan | Trivy gegen Container-Image (CRITICAL+HIGH brechen) | M7 | ja | `LH-QG-007`, `LH-NF-006` |
| Doc-Refs-Linter | `verify-doc-refs.sh` (m-trace-Adaption) | M1 | ja | `LH-QG-008`, `ADR 0003 §5` |
| Gate-Bündelung | `make gates` (Pflicht) und `make security-gates` (Pflicht, parallel im CI) | M1 (Targets angelegt) → M6 (vollständig) | ja | `LH-QG-009` |
| Suppressions-Verbot | Konfigurations-Konvention in `.golangci.yml issues.exclude-rules` (`//nolint`-Pragmas verboten, §2.4) | M1 (Konvention etabliert) | ja | `LH-QG-010` |
| Mutation-Tests | `gremlins` | später, opt-in | nein | `LH-QG-011a` |
| Benchmarks | `go test -bench` | später, opt-in | nein | `LH-QG-011b`, `LH-NF-019` |
| Fuzz-Tests | `go test -fuzz` | später, opt-in | nein | `LH-QG-011c`, `LH-NF-004` |

Pflicht-Gates brechen den CI-Build und werden vor Merge in `main`
erzwungen (`ADR 0011 §2.7`). Opt-in-Gates laufen ggf. als Nightly
oder On-Demand und produzieren Reports, blockieren aber nicht.

### 2.2 Statisches Linting: golangci-lint-Profil

Verbindlich für allen produktiven Go-Code unter dem Operator-Modul:

**5 Default-Linter (`golangci-lint`-Defaults):**

- `govet` — semantische Korrektheit (printf-Argumente etc.).
- `errcheck` — Fehlerwerte werden nicht ignoriert.
- `staticcheck` — klassische Bug-Patterns + Style.
- `unused` — toter Code.
- `ineffassign` — unwirksame Zuweisungen.

**24 SOLID-nahe Linter (Pflicht-Profil):**

| Linter | Kurzbeschreibung | SOLID-Beitrag |
| ------ | ---------------- | ------------- |
| `containedctx` | `context.Context` nicht in Structs speichern | SRP |
| `contextcheck` | Context korrekt weiterreichen | SRP |
| `cyclop` | Zyklomatische Komplexität, max 15 | SRP |
| `depguard` | Import-Regeln / Layer-Grenzen | DIP |
| `dupl` | Code-Duplikate, threshold 150 | OCP |
| `fatcontext` | Context in Loops/Closures | SRP |
| `forbidigo` | Verbotene Identifier/APIs (`fmt.Print*` → `log/slog` etc.) | SRP |
| `funlen` | Funktionslänge, Default-Schwelle aus golangci-lint | SRP |
| `gochecknoglobals` | Keine globalen Variablen | DIP |
| `gochecknoinits` | Keine `init()`-Funktionen | DIP |
| `gocognit` | Kognitive Komplexität | SRP |
| `gocyclo` | Zyklomatische Komplexität (älteres, einfacheres Maß; ergänzt `cyclop` mit pragmatischen Schwellen) | SRP |
| `gomodguard_v2` | Modul-Allow-/Blocklist (golangci-lint-v2-API-konforme Variante; ersetzt das ältere `gomodguard`-Plugin) | DIP |
| `iface` | Interface-Pollution vermeiden | ISP |
| `inamedparam` | Interface-Parameter benennen | ISP |
| `interfacebloat` | Zu große Interfaces | ISP |
| `ireturn` | Interfaces annehmen, konkrete Typen zurückgeben | DIP |
| `maintidx` | Maintainability Index | SRP |
| `nestif` | Tiefe `if`-Verschachtelung | SRP |
| `noctx` | HTTP-Aufrufe ohne Context | SRP |
| `reassign` | Package-Variablen nicht neu zuweisen | DIP |
| `revive` | Konfigurierbarer Stil-/Design-Linter | SRP |
| `testpackage` | Externe `_test`-Packages | ISP |
| `unparam` | Ungenutzte Parameter | ISP |

Konkrete Schwellen (Komplexität, Duplikat-Threshold, `funlen`-Werte)
werden in `.golangci.yml` zentral gepflegt; Startwerte folgen
m-trace (z. B. `cyclop.max-complexity: 15`, `dupl.threshold: 150`).
Senkungen der Schwellen sind frei; Hebungen (z. B. `max-complexity`
auf 20) sind ADR-pflichtig.

### 2.3 `forbidigo`-Verbotsliste (Startwert)

Mindest-Verbote im Pflicht-Profil:

- `fmt.Print*` — Pflicht: `log/slog` mit Kontext (`LH-NF-009`).
- `panic(...)` außerhalb von `init()`-Stoßzeit — Reconcile-Pfade
  müssen Fehler weitergeben, nicht abstürzen (`LH-NF-004`,
  `LH-AK-010`).

Erweiterungen entstehen mit dem Pflichtenheft (`LH-VM-002`).

### 2.4 Suppressions-Verbot

`//nolint`-Pragmas im Code sind verboten. Verstöße werden entweder
durch Code-Änderung behoben oder durch eine zentrale, kommentierte
Konfigurationsregel in `.golangci.yml issues.exclude-rules`
(Pflicht-Format: vorangestellter `Why:`-Kommentar mit Begründung
und Geltungsbereich). Test-Code, Generated-Code und
`scripts/`-Helfer dürfen pro-Pfad anders gewichtet sein, aber jede
Ausnahme ist namentlich dokumentiert.

### 2.5 Coverage-Gate

- **Default-Threshold:** 90 % Line-Coverage über produktive Pakete.
- **Ziel:** ≥ 95 %.
- **Senkung des Defaults** ist ADR-pflichtig (eigene Folge-ADR mit
  Begründung); Anheben pro Operator-Release ist frei.
- **Scope:** Coverage-Range schließt das `cmd/`-Wiring (Signal-
  Handling, OTel-Setup) bewusst aus und konzentriert sich auf die
  fachliche Logik (Reconciler, Check-Implementierungen,
  Discovery-Adapter). Konkreter Range-Selector entsteht mit der
  Paketstruktur im Pflichtenheft.
- **Mechanik:** `go test -cover -coverprofile=...` plus ein
  Schwellen-Check-Skript (Pendant zu `m-trace/apps/api/scripts/
  coverage-gate.sh`).

### 2.6 Architektur-Boundary

Verbindlich, dass das `golangci-lint`-Profil eine `depguard`-Regel
enthält, die die Architekturgrenzen des Operators erzwingt. Die
**konkreten Regeln** und das **Paket-Layout** (kubebuilder vs.
Hexagonal) entstehen mit dem Pflichtenheft (`LH-VM-002`) bzw. mit
einer eigenen Architektur-ADR (`spec/architecture.md`-Stub aus
`ADR 0001 §3`). Diese ADR bindet nur das **Prinzip**: Boundary muss
maschinell prüfbar sein.

### 2.7 Generated-Drift

CRD-YAML-Manifeste und DeepCopy-Methoden werden aus den
Go-Type-Definitionen über `controller-gen` (`kubebuilder`-Standard)
erzeugt. Ein CI-Gate regeneriert beide bei jedem Lauf und vergleicht
mit dem committeten Stand via `git diff --exit-code`; Abweichung
bricht den Build (Pendant zu `m-trace/Makefile generated-drift-check`).
Damit ist die Quelle der Wahrheit (Go-Types) und das Konsumformat
(YAML / generierter Go-Code) konsistent gehalten.

### 2.8 Vulnerability-Scan

`govulncheck` läuft im CI-Job `security-gates` separat von
`make gates`. Funktionsbasiertes Scanning: nur tatsächlich
aufgerufene Vulnerable-Funktionen brechen den Build. Findings ohne
Fix-Plan werden über eine zentral kommentierte Vulnignore-Konfig
zeitlich befristet (`expires`-Datum); abgelaufene Einträge brechen
den Generator (Pendant zu `m-trace/scripts/render-trivyignore.sh`).

Versionspin: bei ADR-Erstellung `v1.1.4` (analog m-trace). Pin-Hebung
ist Routine und wird im Pflichtenheft (`LH-VM-002`) bzw. im
`Makefile`/CI-Workflow gepflegt, **ohne ADR**. Eine inhaltliche
Änderung der Scan-Politik (z. B. fail-open statt fail-closed, andere
Vuln-DB) wäre weiterhin ADR-pflichtig.

### 2.9 Image-Scan

Trivy scannt das Operator-Container-Image. Policy: `CRITICAL` und
`HIGH` brechen den Build, `MEDIUM` wird berichtet.
Pro-Image-Vulnignore-Liste mit `expires`-Pflicht analog §2.8.

Versionspin: bei ADR-Erstellung `aquasec/trivy:0.59.1` (analog
m-trace). Pin-Hebung ist Routine analog §2.8 — ohne ADR. Politik-
Änderungen (z. B. Schwellen-Anpassung von `HIGH` auf `MEDIUM` als
Build-Brecher) bleiben ADR-pflichtig.

### 2.10 Doc-Refs-Linter

`scripts/verify-doc-refs.sh` wird als Adaption des m-trace-Skripts
`/Development/m-trace/scripts/verify-doc-refs.sh` eingeführt
(93-Zeilen-Bash; `set -euo pipefail`; Exit-Codes 0/1/2 für
passed/broken/env-error).

**Geltungsbereich** für k-deskflight:

- Alle `*.md`-Dateien rekursiv unter `docs/` und `spec/`.
- Top-Level-Dokumente: `README.md`, `README.de.md`, `CONTRIBUTING.md`,
  `CODE_OF_CONDUCT.md`, `SECURITY.md`, `CHANGELOG.md` (sobald vorhanden,
  siehe Trigger `docs/plan/planning/open/changelog.md`).

**Was geprüft wird:**

- Lokale Markdown-Linktargets `[text](path)` und `[text](<path>)`-
  Notation. Relative Pfade werden gegen den Speicherort der
  enthaltenden MD-Datei aufgelöst, absolute Pfade gegen Repository-
  Root.
- Image-Links `![…](…)` werden übersprungen (m-trace-Konvention).
- Externe Links mit URL-Schema (`http://`, `https://`, `mailto:`, …)
  werden ignoriert.
- Fragment-Anker (`#section`) werden vor der Auflösung abgeschnitten;
  die Datei muss existieren, nicht der Anker.

**CI-Verankerung:** Teil von `make gates` (siehe §2.11), brechend bei
einem oder mehreren Treffern.

Damit schließt diese ADR den unter `ADR 0003 §5` benannten
Folge-Trigger („`tools/check_refs`-Skript als Routinearbeit"); der
Trigger braucht keinen eigenen `planning/open/`-Eintrag.

### 2.11 Gate-Bündelung

- `make gates` führt die Pflicht-Gates der Inner-Loop zusammen:
  Lint, Tests, Coverage, Architektur-Boundary, Generated-Drift,
  Doc-Refs. Läuft auf dem Entwickler-Rechner in vertretbarer Zeit
  (Sekunden bis wenige Minuten); ist PR-blockierend.
- `make security-gates` führt die externen Gates zusammen:
  `govulncheck`, Image-Scan. Läuft separat im CI (parallel zu
  `make gates`), weil Vulnerability-DB-Download initial mehrere
  Minuten dauern kann. Ebenfalls PR-blockierend.

Beide Bündel werden bei `LH-VM-005` (Integrationstest-Phase)
operativ in CI-Workflows verankert.

### 2.12 Opt-in-Gates

Mutation-Tests, Fuzz-Tests und Benchmarks sind als Pattern
verankert (`gremlins`, `go test -fuzz`, `go test -bench`), aber
nicht MVP-pflichtig. Aktivierung pro Release entweder als Nightly
oder als bewusste Schwellen-Hebung. Sie produzieren Reports, die in
Release-Notes auswertbar sind, blockieren aber den normalen PR-Flow
nicht.

---

## 3. Konsequenzen

- **Lastenheft** bekommt einen neuen Abschnitt `§16a Quality-Gates`
  mit Kennungen `LH-QG-001` bis `LH-QG-010` sowie Sub-Kennungen
  `LH-QG-011a` (Mutation), `LH-QG-011b` (Benchmarks), `LH-QG-011c`
  (Fuzz) gemäß §2.1-Tabelle und
  §2.12 für Opt-in). Die Kennungen sind ab Acceptance dieser ADR
  normativ.
- **Roadmap** `docs/plan/planning/in-progress/roadmap.md` wird
  schärft:
  - M1 erweitert um `make lint`, `make gates`-Skelett, Doc-Refs-Gate.
  - M2 erweitert um Generated-Drift-Gate und Architektur-Boundary-
    Regeln (Stub).
  - M6 erweitert um Coverage-90 %-Gate, `govulncheck`,
    `arch-check`-strikt.
  - M7 erweitert um Image-Scan vor Release-Tag.
- **`ADR 0003 §5` Folge-Trigger** (`tools/check_refs`-Skript) ist
  durch §2.10 dieser ADR adressiert; kein separater
  `planning/open/`-Eintrag erforderlich.
- **`golangci-lint`-Konfiguration** und Linter-Schwellen entstehen
  in einer Datei `.golangci.yml` im Repository-Root oder im
  Operator-Modul-Pfad — Verortung wird im Pflichtenheft
  konkretisiert.
- **Pflichtenheft (`LH-VM-002`)** ist die Heimat für: konkrete
  `depguard`-Regeln, Paket-Layout, Coverage-Range, Probe-Timeouts,
  CI-Job-Layout, Vulnignore-Konvention.
- **Schwesterprojekt-Konsistenz:** k-deskflight nutzt damit dieselbe
  Quality-Gate-Linie wie m-trace; spätere Cross-Reviews oder
  geteilte Tooling-Verbesserungen sind möglich.
- **`LH-RISK-002`** (Zu großer Projektumfang): Die hohe
  Coverage-Schwelle und die SOLID-nahe Linter-Liste erhöhen den
  Aufwand pro PR; das ist bewusst akzeptiert, weil sie die spätere
  Wartbarkeit hebt.

---

## 4. Nicht Gegenstand dieser ADR

- **Konkrete `depguard`-Regeln** und das **Paket-Layout** (kubebuilder
  vs. Hexagonal) — Pflichtenheft (`LH-VM-002`) bzw.
  `spec/architecture.md`.
- **Coverage-Range-Selector** (welche Pakete in den Nenner gehen,
  welche bewusst nicht) — Pflichtenheft.
- **Konkrete `.golangci.yml`-Inhalte** (Schwellen-Werte, Pfad-
  Carveouts mit `Why:`-Kommentar) — entstehen mit M1.
- **TypeScript-/Svelte-Linter-Profile** — k-deskflight hat kein
  TS-Workspace; entfällt.
- **Mutation-Test-/Fuzz-Schwellen** (Mutation-Score-Ziel, Fuzz-Budget)
  — falls die Opt-in-Gates aktiviert werden, eigene ADR mit den
  konkreten Schwellen.
- **CI-Plattform-Details** (GitHub-Actions-Workflow-YAML, Reusable
  Workflows) — `ADR 0011` bindet GitHub als Hoster, konkrete
  Workflow-Definition entsteht mit M1 und der CI-Implementierung.
- **Release-Approval-Mechanismus** (Pendant zu m-trace
  `MTRACE_RELEASE_APPROVED=1`, `image-publish-guard`) —
  Pflichtenheft.
- **OpenAPI-/CRD-Schema-Validierung** (Pendant zu
  m-trace `schema-validate`) — entsteht mit der CRD-Auslieferung in M2;
  diese ADR bindet nur das Drift-Gate, nicht das Schema-Lint-Gate
  selbst.
- **Performance-Budgets** für den Operator (Speicher, CPU,
  Reconcile-Latenz) — `LH-NF-019` als Anker, konkretes Budget mit
  späterem Benchmark-Slice.
