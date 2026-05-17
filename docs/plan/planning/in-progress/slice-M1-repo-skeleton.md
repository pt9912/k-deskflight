# Slice M1 — Repo & Build-Skeleton

**Status:** In Progress
**Eröffnet:** 2026-05-17
**Vorgänger:** keiner (Eintritts-Slice)
**Nachfolger:** [M2 — CRD + Controller-Skeleton](./roadmap.md#m2--crd--controller-skeleton)
**Bezug:** [Roadmap §3 M1](./roadmap.md#m1--repo--build-skeleton),
[`spec/architecture.md` §3 (AR-001–AR-005) und §AR-019](../../../../spec/architecture.md),
[ADR 0004 §2](../../adr/0004-projektname.md),
[ADR 0011](../../adr/0011-governance-und-beitragskonventionen.md),
[ADR 0012 §2.1–§2.11](../../adr/0012-quality-gates.md)

---

## 1. Lieferziel

Go-Projektskelett im Repository-Root mit lauffähigen Bau-, Test- und
Lint-Targets, einem Multi-Stage-`Dockerfile`, einem CI-Workflow-Skelett
(zwei parallele Jobs `gates` und `security-gates`) und allen
Pflicht-Quality-Gates auf Skelett-Ebene. **Kein Fachcode**, nur ein
minimaler `cmd/operator/main.go`, der ohne Argumente startet, eine
Selbst-Identifikation auf `stdout` schreibt und sauber terminiert.

---

## 2. Slice-Entscheidungen

### 2.1 Build-Workflow

**Docker-only**, analog `m-trace/apps/api`. Konsequenzen:

- Repository enthält **keine lokale Go-Toolchain-Anforderung** in den
  Pflicht-Targets.
- `Makefile`-Targets rufen ausschließlich
  `docker build --target <stage>` (lint/test/build) bzw.
  `docker run` (scripts).
- BuildKit-Cache (`GOMODCACHE`, `GOCACHE`) wird über den `deps`-Stage
  geteilt; `--no-cache-filter <stage>` zwingt Stale-Layer-Reset bei
  `lint`/`test`/`coverage` (Pattern aus `m-trace/apps/api/Makefile`).
- Lokale Go-Installation ist erlaubt, aber **nicht** Voraussetzung für
  PR-grünen Pfad.

### 2.2 CI-Workflow

**Zwei parallele Jobs** gemäß [`ADR 0012 §2.11`](../../adr/0012-quality-gates.md):

- `gates` — `make gates` (lint + test + doc-refs; coverage-gate als
  Smoketest mit Schwelle 0 %, Architektur-Boundary lax bis M2,
  Generated-Drift hat noch kein Eingabematerial bis M2).
- `security-gates` — `make security-gates` (`govulncheck` als Stub;
  funktionsbasiertes Scanning gegen `go.mod`-Standard, M6 hebt strikt;
  Image-Scan via Trivy erst in M7 vor dem Release-Tag).

Beide Jobs sind PR-blockierend. DCO-Check wird über die GitHub-Org-Bot-
Aktivierung gepflegt, **nicht** als Workflow-Step ([`ADR 0011 §2.4`](../../adr/0011-governance-und-beitragskonventionen.md)).

### 2.3 Linter-Profil

**Volles Pflicht-Profil** aus [`ADR 0012 §2.2`](../../adr/0012-quality-gates.md):
5 Default-Linter (`govet`, `errcheck`, `staticcheck`, `unused`,
`ineffassign`) + 24 SOLID-nahe Linter. `depguard` ist im Profil
aktiviert, aber **ohne Regeln** — die fünf konkreten Regelblöcke
([`architecture.md` §AR-005](../../../../spec/architecture.md))
werden in M2 ergänzt, sobald die Paketstruktur Inhalte hat. Startwerte
für Schwellen (`cyclop.max-complexity: 15`, `dupl.threshold: 150`)
übernehmen die m-trace-Werte ([`ADR 0012 §2.2`](../../adr/0012-quality-gates.md)).

### 2.4 Versionspins

| Komponente | Pin in M1 | Quelle |
| ---------- | --------- | ------ |
| Go-Toolchain | `golang:1.26.3` | analog `m-trace/apps/api/Dockerfile` |
| `golangci-lint` | `v2.12.1-alpine` | analog `m-trace/apps/api/Dockerfile` |
| `govulncheck` | `v1.1.4` (per `go install`) | [`ADR 0012 §2.8`](../../adr/0012-quality-gates.md) |
| Runtime-Base | `gcr.io/distroless/static:nonroot` | [`architecture.md` §AR-019](../../../../spec/architecture.md), Step 6 |

Pin-Hebungen sind Routine ohne ADR ([`ADR 0012 §2.8`/`§2.9`](../../adr/0012-quality-gates.md)); Begründung im Commit-Body genügt.

---

## 3. Datei-Inventar

Neu im Repository:

| Pfad | Zweck | Vorlage |
| ---- | ----- | ------- |
| `go.mod` | Modulpfad `github.com/pt9912/k-deskflight`, `go 1.26` | [`architecture.md` §AR-001](../../../../spec/architecture.md) |
| `cmd/operator/main.go` | Smoke-Entry: Logging-Setup-Stub, Print Build-Info, return 0 | [`architecture.md` §AR-003](../../../../spec/architecture.md) |
| `internal/hexagon/{domain,application,port}/.gitkeep` | Layer-Verzeichnisse pro AR-003 | — |
| `internal/adapter/{k8s,check}/.gitkeep` | Adapter-Verzeichnisse | — |
| `api/v1alpha1/.gitkeep` | CRD-Typen-Heimat (Inhalt in M2) | — |
| `config/{crd,rbac,samples}/.gitkeep` | `controller-gen`-Output-Ziel (M2+) | — |
| `deploy/manifests/.gitkeep` | Roh-Manifeste ([`ADR 0005`](../../adr/0005-helm-chart-nicht-im-mvp.md)) | — |
| `Dockerfile` | Multi-Stage: `deps`/`lint`/`test`/`coverage`/`compile`/`build`/`runtime` | `m-trace/apps/api/Dockerfile` |
| `Makefile` | Targets `build`/`lint`/`test`/`coverage-gate`/`doc-refs`/`image-build`/`gates`/`security-gates`/`govulncheck`/`clean` | `m-trace/apps/api/Makefile` + `m-trace/Makefile` (gates-Bündelung) |
| `.golangci.yml` | 5 Defaults + 24 SOLID-nahe Linter; `depguard` aktiviert ohne Regeln; `forbidigo`-Startverbote (`fmt.Print*`, `panic(...)`) | `m-trace/apps/api/.golangci.yml` |
| `.dockerignore` | Build-Kontext-Filter | `m-trace/apps/api/.dockerignore` (falls vorhanden, sonst minimal) |
| `scripts/verify-doc-refs.sh` | Doc-Refs-Linter ([`ADR 0012 §2.10`](../../adr/0012-quality-gates.md)) — Adaption: Geltungsbereich um `README.de.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md` erweitern | `m-trace/scripts/verify-doc-refs.sh` |
| `scripts/coverage-gate.sh` | Schwellen-Check ([`ADR 0012 §2.5`](../../adr/0012-quality-gates.md)); Default-Schwelle in M1 = 0 (Smoketest), M6 hebt auf 90 | `m-trace/apps/api/scripts/coverage-gate.sh` |
| `.github/workflows/ci.yml` | Zwei Jobs `gates` und `security-gates`, beide auf `ubuntu-latest` mit BuildKit aktiv | [`ADR 0012 §2.11`](../../adr/0012-quality-gates.md), [`ADR 0011 §2.5–§2.6`](../../adr/0011-governance-und-beitragskonventionen.md) |

**Keine Anpassung** in dieser Slice: `README.md`/`README.de.md`,
`CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `LICENSE` —
Stand `main` bleibt. Folge-Edits (Tippfehler, ADR-Aufnahmen) sind
zulässig, ohne diesen Slice-Plan anzupassen.

---

## 4. Reihenfolge der Umsetzung

1. **`go.mod`** mit `module github.com/pt9912/k-deskflight` und `go 1.26`.
2. **Layout** anlegen (`cmd/operator/`, `internal/hexagon/{domain,application,port}/`,
   `internal/adapter/{k8s,check}/`, `api/v1alpha1/`, `config/{crd,rbac,samples}/`,
   `deploy/manifests/`, `scripts/`) jeweils mit `.gitkeep`.
3. **`cmd/operator/main.go`** als 15-Zeilen-Smoke-Binary (kein
   `controller-runtime`-Wiring — das kommt in M2, sonst hätten wir
   schon Dependencies, die der Generated-Drift-Gate in M2 anfasst).
4. **`Dockerfile`** mit allen Stages (`deps`/`lint`/`test`/`coverage`/`compile`/`build`/`runtime`).
5. **`Makefile`** mit Pflicht-Targets plus `gates`- und
   `security-gates`-Bündel.
6. **`.golangci.yml`** mit vollem Profil; `depguard`-Block leer im
   `rules:`-Map, dokumentiert via `Why:`-Kommentar.
7. **Scripts** (`scripts/verify-doc-refs.sh`, `scripts/coverage-gate.sh`) mit
   den k-deskflight-spezifischen Pfaden.
8. **`.github/workflows/ci.yml`** mit beiden Jobs.
9. **Roadmap-Eintrag** in [`roadmap.md` §2 Tabelle und §7 Status](./roadmap.md):
   M1 von „Pending" auf „In Progress" und nach Abschluss auf „Done"
   ziehen; Closure-Notiz in `done/slice-M1-repo-skeleton.md`.

Jeder Schritt = ein eigener Commit; Commit-Botschaften nach Convention
([`ADR 0011 §2.7`](../../adr/0011-governance-und-beitragskonventionen.md)).

---

## 5. Lastenheft-Kennungen

`LH-NF-001` (Go als Sprache),
`LH-NF-014` (lokale Entwicklung — hier: Docker-only-Path),
`LH-NF-015` (Container-Image),
`LH-NF-021` (Sprachregel — Englisch im Code, Deutsch in Doku ist außerhalb dieser Slice),
`LH-PROD-001` (Naming `k-deskflight`),
`LH-QG-001` (Linting aktiv),
`LH-QG-008` (Doc-Refs aktiv),
`LH-QG-009` (Gate-Bündelung `make gates` / `make security-gates`),
`LH-QG-010` (Suppressions-Verbot: `//nolint`-Pragmas verboten).

---

## 6. Architekturartefakte

Erfüllt aus `spec/architecture.md`:

- `AR-001` (Go-Modulpfad)
- `AR-002` (Hybrid kubebuilder/Hexagonal — Verzeichnis-Stub)
- `AR-003` (Verzeichnis-Layout — Stub mit `.gitkeep`)
- `AR-019` (Dockerfile-Stages)

Vorbereitet, aktiv ab späterer Slice:

- `AR-004` (Layer-Modell) — wird mit `depguard`-Regeln in M2 maschinell.
- `AR-005` (`depguard`-Regeln) — Block in `.golangci.yml` vorbereitet,
  Regeln in M2.
- `AR-007` ff. — Reconciler/Check-Inhalte ab M2/M3.

---

## 7. Verifikation (Abnahmekriterien)

1. **`make build`** läuft grün; Image `k-deskflight:go` existiert.
2. **`make lint`** läuft grün (oder mit dokumentiertem Carveout im
   `Why:`-Kommentar) — d. h. das Pflicht-Profil bewertet den
   Smoke-`main.go` als sauber.
3. **`make test`** läuft grün, auch wenn nur ein Smoke-Test
   (`TestMainReturnsZero` oder vergleichbar) existiert.
4. **`make coverage-gate`** läuft grün mit Schwelle 0 (Smoketest-
   Schwelle für M1; M6 hebt auf 90).
5. **`make doc-refs`** läuft grün auf dem aktuellen `docs/`/`spec/`-
   Stand.
6. **`make gates`** ruft (1)–(5) gebündelt und ist grün.
7. **`make security-gates`** ruft `govulncheck` und ist grün (es gibt
   noch keine Dependencies außer Go-Stdlib).
8. **`make image-build`** produziert ein lauffähiges Image; ein lokaler
   `docker run --rm k-deskflight:go` schreibt eine Selbst-
   Identifikation auf `stdout` und exited mit 0.
9. **CI-Workflow-Run** auf einem Beispiel-PR ist grün (beide Jobs).

**Kein dediziertes `LH-AK-*`** für M1 selbst — M1 ist die Voraussetzung
für alle weiteren `LH-AK-*` ab M2 ([`Roadmap §3 M1`](./roadmap.md#m1--repo--build-skeleton)).

---

## 8. Out-of-Scope (geht in M2 oder später)

- **CRD-Typen** und `controller-gen`-Lauf — M2.
- **Generated-Drift-Gate strikt** — M2 (ab dann existiert
  `config/crd/`-Inhalt).
- **`depguard`-Regelblöcke** — M2 ([`architecture.md` §AR-005](../../../../spec/architecture.md)).
- **Coverage-Schwelle 90 %** — M6 ([`ADR 0012 §2.5`](../../adr/0012-quality-gates.md)).
- **`govulncheck` strikt** — M6.
- **Trivy Image-Scan** — M7.
- **Beispielmanifeste mit CR-Inhalt** — M2 (leerer CR), M7 (vollständig).
- **`controller-runtime`-Wiring im `main.go`** — M2.

---

## 9. Risiken und Mitigation

- **Linter-Profil zu streng auf Skelett-Code:** Smoke-`main.go` muss
  vom Pflicht-Profil als sauber bewertet werden. Falls nicht: kein
  `//nolint`-Pragma (verboten, [`LH-QG-010`](../../../../spec/lastenheft.md)),
  sondern dokumentierter Carveout in `.golangci.yml issues.exclude-rules`
  mit `Why:`-Kommentar — und dabei darauf achten, dass die Ausnahme so
  eng wie möglich gefasst ist.
- **`golangci-lint v2.12.1` evtl. inkompatibel mit `go 1.26`:** m-trace
  läuft auf dieser Kombination grün; sollte sich das ändern, Pin im
  `Dockerfile` heben (Routine-Operation, kein ADR).
- **CI-Workflow `gates` und `security-gates` teilen den Build-Cache
  nicht** zwischen Jobs: bewusst akzeptiert, weil Trennung den
  PR-Pfad stabiler macht (Vuln-DB-Download isoliert). Build-Zeit-
  Impact wird in M7 bewertet.
