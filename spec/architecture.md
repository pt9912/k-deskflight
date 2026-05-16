# Architekturbeschreibung: k-deskflight (OpenDesk Preflight Operator)

**Dokument-ID:** AR-OPD-PFO-001
**Bezug:** [Lastenheft `LH-OPD-PFO-001`](lastenheft.md)
**Artefakt:** Architekturbeschreibung (`ADR 0001 §3`)
**Version:** 0.1.0
**Status:** Entwurf
**Autor:** Dietmar Burkard
**Sprache:** Deutsch (`LH-NF-021`)

---

## 1. Zweck und Geltungsbereich

Diese Architekturbeschreibung ergänzt das Lastenheft (`LH-OPD-PFO-001`)
und konkretisiert die strukturellen Entscheidungen für die
Implementierung des `k-deskflight`-Operators. Sie schließt die in
`ADR 0001 §3` als „spätere `spec/architecture.md`" angekündigte
Lücke und liefert insbesondere die in `ADR 0012 §2.6` ans
Pflichtenheft delegierten `depguard`-Regeln.

**Was diese Datei festlegt:**

- Modulpfad, Verzeichnis-Layout und Layer-Modell.
- Architektur-Boundary-Regeln (`depguard`-Pflicht aus `ADR 0012`).
- Skizze der CRD-API-Schicht, des Controller-Reconciler-Pfads, der
  Check-Plugin-Architektur und des RBAC-Konzepts.
- Build- und Release-Pipeline-Anker für `make`-Targets und CI-Job-
  Struktur.

**Was diese Datei NICHT festlegt:**

- Exakte CRD-Spec-Feld-Typen, kubebuilder-Marker im Detail,
  OpenAPI-Validierung pro Feld — Pflichtenheft (`LH-VM-002`) bzw.
  Slice-Pläne in `docs/plan/planning/in-progress/`.
- Konkrete Probe-Implementierungen, Test-Mock-Strukturen, exakte
  Timeouts und Retry-Budgets — Pflichtenheft und Slice-Pläne.
- CI-Workflow-YAML im Detail, Release-Approval-Skript-Layout —
  M1-Slice-Plan.
- Operative Tool-Versionspins (govulncheck, Trivy, gremlins) —
  `ADR 0012 §2.8/§2.9` markiert sie als nicht-ADR-pflichtig;
  konkrete Werte leben im `Makefile`.

---

## 2. Kennungsraum `AR-*`

Architekturartefakte führen den eigenen Kennungsraum `AR-*`
(Architecture). Die Wahl folgt der in `ADR 0003 §1` offen gelassenen
Option „eigene Präfixfamilie".

| Sub-Präfix | Bedeutung |
| ---------- | --------- |
| `AR-001..` | normative Architekturentscheidungen mit langfristiger Wirkung |
| `AR-OP-001..` | offene Architektur-Punkte (analog `LH-OP-*`) |

`AR-*`-Kennungen sind positionsunabhängig und stabil
(`ADR 0003 §2.1`). Cross-Verweise zwischen `LH-*` und `AR-*` nutzen
die Kennungen direkt, nicht §-Nummern (`ADR 0003 §2.4`).

---

## 3. Modulstruktur

### AR-001 — Go-Modulpfad

```text
github.com/pt9912/k-deskflight
```

Halterung GitHub-Org/-Account `pt9912` (konsistent mit
`ADR 0011 §2.6` GitHub-Hosting und mit dem Halter von
`/Development/m-trace`).

### AR-002 — Paket-Layout-Schule

**Hybrid: außen `kubebuilder`, innen Hexagonal.**

- **Außen kubebuilder** für Tooling-Kompatibilität: `cmd/`-Entry,
  `api/v1alpha1/`-Typen mit kubebuilder-Markern, `config/`-Manifeste
  via `controller-gen`. Damit funktionieren `kubebuilder create`,
  `controller-gen object`, `controller-gen crd` direkt.
- **Innen Hexagonal** unter `internal/` für fachliche Logik-
  Trennung: `domain/`, `application/`, `port/`, `adapter/`. Damit
  bleibt die Domänenlogik unabhängig von der Kubernetes-API und
  testbar ohne Cluster.

Das Pattern ist im Kubernetes-Operator-Ökosystem etabliert (z. B.
ähnlich `metrics-server`-Schichtenung) und bewahrt die Kohärenz mit
dem m-trace-Schwesterprojekt.

### AR-003 — Verzeichnis-Layout

```text
github.com/pt9912/k-deskflight
├── cmd/
│   └── operator/
│       └── main.go              # Entry-Point: controller-runtime-Setup,
│                                # OTel-/Metrics-Wiring, Signal-Handling
├── api/
│   └── v1alpha1/
│       ├── opendeskpreflightcheck_types.go   # CRD-Spec/-Status-Typen
│       ├── groupversion_info.go              # Group/Version-Registrierung
│       └── zz_generated.deepcopy.go          # controller-gen-Output
├── internal/
│   ├── domain/                  # Reine Domänentypen, keine k8s-Abhängigkeit
│   │   ├── check.go             # Check-Interface, Result, Severity
│   │   └── profile.go           # Profile-Konstanten, Default-Werte
│   ├── application/             # Use-Cases / Reconciler-Orchestrierung
│   │   ├── reconciler.go        # Reconcile-Loop (high-level)
│   │   └── aggregator.go        # Status-Aggregation per LH-F-031
│   ├── port/                    # Interfaces, von application konsumiert
│   │   ├── kubernetes.go        # KubernetesAPI-Interface
│   │   ├── checkregistry.go     # CheckRegistry-Interface
│   │   └── clock.go             # Clock-Interface (Test-Inversion)
│   └── adapter/
│       ├── k8s/                 # Kubernetes-API-Adapter (implementiert port)
│       │   ├── discovery.go     # ServerVersion-Lookup für LH-F-008
│       │   ├── storage.go       # StorageClass-Lookup für LH-F-010
│       │   ├── ingress.go       # IngressClass-Lookup für LH-F-012
│       │   ├── certmanager.go   # cert-manager-API-Discovery für LH-F-013
│       │   ├── nodes.go         # Node-Listing + Allocatable-Summe für LH-F-015
│       │   └── rbac.go          # SelfSubjectAccessReview für LH-F-024
│       └── check/               # Konkrete Check-Implementierungen
│           ├── kubernetesversion.go
│           ├── storageclass.go
│           ├── ingressclass.go
│           ├── certmanager.go
│           ├── resources.go
│           └── rbac.go
├── config/                      # kubebuilder-Manifeste (controller-gen)
│   ├── crd/                     # Generierte CRD-YAMLs
│   ├── rbac/                    # Generierte Role/ClusterRole/Binding
│   └── samples/                 # Beispiel-CRs
├── deploy/
│   └── manifests/               # Roh-Manifeste für `kubectl apply -k`
│                                # (ADR 0005 — kein Helm Chart im MVP)
├── scripts/
│   ├── coverage-gate.sh         # LH-QG-003 (Adaption von m-trace)
│   └── verify-doc-refs.sh       # LH-QG-008 (Adaption von m-trace)
├── docs/
├── spec/                        # Lastenheft + diese Architektur
├── Dockerfile                   # Multi-Stage (deps/lint/test/build/runtime)
├── Makefile                     # build, lint, test, gates, security-gates,
│                                # image-build, image-publish, release-guard
├── .golangci.yml                # 5 Defaults + 24 SOLID-nahe + depguard-Regeln
├── README.md / README.de.md
├── CONTRIBUTING.md / CODE_OF_CONDUCT.md / SECURITY.md
├── LICENSE
├── go.mod
└── go.sum
```

### AR-004 — Layer-Modell

Abhängigkeitspfeile von innen nach außen:

```text
domain  ← application  ↔  port
                            ↑
                          adapter
```

- `domain` hat **keine** Imports aus `application`, `port`,
  `adapter`. Auch keine Kubernetes-Libraries.
- `application` importiert `domain` und `port`. Keine `adapter`.
- `port` importiert `domain`. Keine `application` (wäre Zyklus),
  keine `adapter`.
- `adapter` importiert `port` (zur Implementierung) und `domain`
  (für Result/Severity-Typen). Importiert **nicht** `application`.
- `cmd/operator` ist die Wiring-Schicht: importiert `application`
  und `adapter`, instanziiert konkrete Adapter und injiziert sie in
  den Reconciler. Diese Schicht ist nicht testpflichtig
  (Coverage-Range-Selektor schließt sie aus, `LH-QG-003`).
- `api/v1alpha1` ist Spec-Definition; importiert nur Kubernetes-
  Standard-Typen und kubebuilder-Markers.

### AR-005 — `depguard`-Regeln

Pflicht aus `LH-QG-004` und `ADR 0012 §2.6`. Im `.golangci.yml` als
Linter-Regel im Pflicht-Profil aktiv. Verstöße brechen den Build.

```yaml
depguard:
  rules:
    domain-isolation:
      list-mode: lax
      files:
        - '**/internal/domain/**'
      deny:
        - pkg: k8s.io
          desc: domain layer must not depend on Kubernetes libraries (use port)
        - pkg: sigs.k8s.io
          desc: domain layer must not depend on controller-runtime
        - pkg: github.com/pt9912/k-deskflight/internal/application
          desc: domain must not depend on application (AR-004)
        - pkg: github.com/pt9912/k-deskflight/internal/port
          desc: domain must not depend on port (AR-004)
        - pkg: github.com/pt9912/k-deskflight/internal/adapter
          desc: domain must not depend on adapter (AR-004)
    application-no-adapter:
      list-mode: lax
      files:
        - '**/internal/application/**'
      deny:
        - pkg: github.com/pt9912/k-deskflight/internal/adapter
          desc: application must depend on ports, not on adapter implementations (AR-004)
    port-no-application:
      list-mode: lax
      files:
        - '**/internal/port/**'
      deny:
        - pkg: github.com/pt9912/k-deskflight/internal/application
          desc: ports are abstractions, must not depend on application (AR-004)
        - pkg: github.com/pt9912/k-deskflight/internal/adapter
          desc: ports define abstractions, must not depend on adapter implementations (AR-004)
    adapter-no-application:
      list-mode: lax
      files:
        - '**/internal/adapter/**'
      deny:
        - pkg: github.com/pt9912/k-deskflight/internal/application
          desc: adapter implements port, must not call into application directly (AR-004)
```

Konkrete Schwellen für die übrigen Linter (`cyclop.max-complexity`,
`funlen`, `dupl.threshold` etc.) folgen der m-trace-Vorlage und sind
mit dem M1-Slice-Plan zu fixieren.

---

## 4. CRD-API-Schicht (`api/v1alpha1/`)

### AR-006 — Type-Layout

Datei `opendeskpreflightcheck_types.go` enthält die Spec/Status-
Strukturen für `OpenDeskPreflightCheck` (`LH-PROD-002`, `ADR 0006`).
Die exakten Feld-Namen, Typen, OpenAPI-Constraints und kubebuilder-
Marker entstehen mit der M2-Slice-Implementierung und sind
Pflichtenheft-Inhalt (`LH-VM-002`). Diese Architektur bindet nur
die **Struktur-Schichten**:

- `Spec.Profile` (`LH-PROF-001..004`): String-Enum mit Default
  `production`.
- `Spec.Checks`: verschachtelte Map auf Check-Typ-spezifische
  Sub-Strukturen (kubernetesVersion, ingress, certManager, storage,
  resources). MVP-Set siehe `LH-PRI-001`.
- `Status.Phase`: Enum (`LH-F-006`: `Pending`/`Running`/`Passed`/
  `Warning`/`Failed`/`Unknown`).
- `Status.Summary`: Aggregat (`LH-F-007`: passed/warning/failed/
  lastChecked).
- `Status.Conditions`: Standard-Kubernetes-Conditions-Liste
  (`LH-F-005`); Reason/Severity/Message gemäß `LH-F-031`/`LH-F-032`.

### AR-007 — Generated-Drift-Mechanik

`controller-gen` (kubebuilder-Tooling) erzeugt:

- `zz_generated.deepcopy.go` im `api/v1alpha1/`-Verzeichnis.
- CRD-YAML unter `config/crd/`.
- RBAC-Skelette unter `config/rbac/` (basierend auf
  kubebuilder-Markern im Controller-Code).

Diese Artefakte sind committet, aber von Hand nicht editierbar.
`LH-QG-005` (Generated-Drift-Gate, `ADR 0012 §2.7`) regeneriert sie
bei jedem CI-Lauf und vergleicht via `git diff --exit-code`.

### AR-008 — Versioning-Strategie

Initial `v1alpha1` (`ADR 0006 §2.3`). Schema-Brüche zwischen
MVP-Releases sind nach Kubernetes-Konvention für `v1alpha1`
zulässig. Migration auf `v1alpha2` oder `v1beta1` ist Folge-ADR-Stoff
(siehe `ADR 0006 §4`).

Conversion-Webhooks sind **nicht** Bestandteil des MVP. Ein
Versionssprung im MVP-Zeitfenster wird über CR-Re-Apply gelöst, nicht
über Server-seitige Conversion.

---

## 5. Controller-Reconciler

### AR-009 — Reconcile-Pfad

Der Reconciler lebt in `internal/application/reconciler.go` und
folgt einem deterministischen sechs-Phasen-Pfad pro Reconcile-Lauf:

1. **Fetch** — CR über `client.Get` lesen. Bei `NotFound`: kein
   Requeue (CR gelöscht).
2. **Validate** — Spec-Konsistenz prüfen (Profile-Wert gültig,
   keine widersprüchlichen Felder). Bei Validierungsfehler:
   Phase `Failed`, Condition `SpecInvalid`.
3. **Determine Active Checks** — basierend auf Profile und
   Spec.Checks-Map die zu aktivierenden Check-Instanzen aus der
   `CheckRegistry` (`AR-013`) auflösen.
4. **Execute Checks** — pro aktivem Check eine Goroutine starten
   (Limit per Worker-Pool — Pflichtenheft-Detail) oder sequenziell,
   je nach Pflichtenheft-Wahl. Pro Check: `Run(ctx, spec) Result`
   ohne Cluster-Mutation.
5. **Aggregate** — Schweregrade auf die Gesamtphase mappen
   (`LH-F-031`/`ADR 0010 §2.3`).
6. **Update Status** — `client.Status().Update` mit
   `Phase`/`Summary`/`Conditions`. Konflikte (resourceVersion)
   werden mit Re-Fetch + Retry behandelt.

### AR-010 — Wiederholintervall (`LH-F-025`)

`Spec.Interval` (Default Pflichtenheft, vorgeschlagener Startwert
`5m`) steuert `RequeueAfter` am Ende des Reconcile-Laufs. Manuelle
Auslösung (`LH-F-026`) erfolgt durch Anwender-Edit am CR
(Annotation-Bump oder Spec-Änderung) — automatisches Re-Reconcile
durch das controller-runtime-Watch.

### AR-011 — Error-Handling und Fehlertoleranz

Pro `LH-NF-005`: ein Fehler in einer Einzel-Check-Ausführung darf
andere Checks nicht stoppen. Konkrete Mechanik:

- Jede `Check.Run`-Implementierung fängt `panic` mittels
  `defer/recover` und gibt einen `Result` mit `Status: Unknown`,
  `Reason: InternalError` zurück.
- Reconciler aggregiert auch `Unknown`-Results in die Gesamtphase
  (`Unknown` per `LH-F-031`).
- Logger schreibt jedes Internal-Error mit Stack-Trace, aber kein
  Stack-Trace im CR-Status (`LH-SEC-002` / `LH-NF-007`).
- Reconciler selbst hat einen äußeren `defer/recover` für
  unerwartete Panics; bei einem solchen wird Phase `Unknown` mit
  Condition `ReconcileError` gesetzt.

### AR-026 — Leader-Election und Replica-Modell

Der `controller-runtime`-Manager wird mit **Leader-Election
aktiviert** gestartet. Konkret: `Manager.Options.LeaderElection =
true` mit `LeaderElectionID = "k-deskflight-operator"` (oder
äquivalent), `LeaderElectionNamespace = <Operator-Namespace>`
(siehe `AR-016`).

Damit gilt:

- **Im MVP** läuft genau **eine aktive Replica** des Operators.
  Mehrere Replicas im selben Deployment sind technisch zulässig,
  führen aber zum Standby-Verhalten der Nicht-Leader-Replicas — kein
  Doppel-Reconcile, keine Status-Konflikte (`AR-009 §6`
  resourceVersion-Konflikte sind damit auf legitime Konfliktszenarien
  beschränkt, nicht auf Self-Conflict).
- **Failover** erfolgt über das Standard-Lease-Renewal-Schema
  (`coordination.k8s.io/leases`, Default-Lease-Duration 15 s,
  Renew-Deadline 10 s, Retry-Period 2 s — konkrete Werte gehören
  ins Pflichtenheft, controller-runtime-Defaults sind akzeptabel).
- **RBAC** für Leases ist in `AR-015` eingebunden
  (`coordination.k8s.io/leases` mit `get/list/watch/create/update/
  patch`).
- **Hochverfügbarkeit** mit echtem Active-Active oder fortgeschritteneren
  Modellen ist Folge-ADR-Stoff für v0.2+ (`LH-NF-019`-Ressourcenkonzept
  und Replica-Skalierung).

Begründung: Ohne Leader-Election produziert ein versehentliches
Hochskalieren des Deployments auf `replicas: 2` doppelte
Reconcile-Loops, was sich in unkontrollierten Status-Flips und
Event-Lawinen äußert. Die Mehrkosten der Leader-Election (eine
zusätzliche `Lease`-Ressource im Operator-Namespace, ein paar
Watch-Verbindungen) sind vernachlässigbar; die Risiko-Mitigation ist
substantiell.

---

## 6. Check-Plugin-Architektur

### AR-012 — Check-Interface

Definiert in `internal/domain/check.go`:

```go
package domain

import "context"

type Severity string

const (
    SeverityInfo     Severity = "info"
    SeverityWarning  Severity = "warning"
    SeverityCritical Severity = "critical"
)

type ConditionStatus string

const (
    StatusTrue    ConditionStatus = "True"
    StatusFalse   ConditionStatus = "False"
    StatusUnknown ConditionStatus = "Unknown"
)

type Result struct {
    Name           string             // Conditiontype, z. B. KubernetesVersionReady
    Status         ConditionStatus
    Reason         string             // Maschinenlesbar, CamelCase
    Severity       Severity
    Message        string             // Menschenlesbar
    LastTransition time.Time
}

type Check interface {
    Name() string                                  // Stable ID, used as ConditionType
    Run(ctx context.Context, spec CheckSpec) Result
}

type CheckSpec interface{}   // Marker; konkrete Checks casten auf eigene Sub-Spec
```

### AR-013 — Check-Registry

Definiert in `internal/port/checkregistry.go` (Interface) und
implementiert in `internal/adapter/check/registry.go`.

```go
package port

import "github.com/pt9912/k-deskflight/internal/domain"

type CheckRegistry interface {
    Register(c domain.Check)
    Resolve(name string) (domain.Check, bool)
    ListByProfile(profile string, spec map[string]any) []domain.Check
}
```

`cmd/operator/main.go` registriert beim Start alle MVP-Checks
(KubernetesVersion, StorageClass, IngressClass, certManager,
Resources, RBAC).

### AR-014 — Schweregrad-Aggregation

Implementierung in `internal/application/aggregator.go`. Mappt eine
Liste von `Result` auf die Gesamtphase nach `LH-F-031`-Tabelle.
Reihenfolge: höchster Schweregrad eines Failed-Results bestimmt die
Phase. `Unknown`-Results aus `ConnectivityUnknown`-artigen Checks
(`ADR 0010 §2.3`) führen zu Gesamtphase `Unknown`, sofern kein
`critical`/`warning`-Fail vorliegt.

---

## 7. RBAC-Konzept

### AR-015 — ClusterRole MVP-Minimum

Für die MVP-Prüfungen werden ausschließlich **lesende** cluster-weite
Rechte benötigt (`LH-F-035`, `LH-AK-015`, `LH-NF-006`):

| API-Gruppe | Ressourcen | Verben | Begründung |
| ---------- | ---------- | ------ | ---------- |
| `""` (core) | `nodes` | `get`, `list`, `watch` | Allocatable CPU/Memory, Ready-Status (`LH-F-015`) |
| `storage.k8s.io` | `storageclasses` | `get`, `list`, `watch` | `LH-F-010`/`LH-F-011` |
| `networking.k8s.io` | `ingressclasses` | `get`, `list`, `watch` | `LH-F-012` |
| `apiextensions.k8s.io` | `customresourcedefinitions` | `get`, `list` | `cert-manager.io`-Discovery (`LH-F-013`) |
| `authorization.k8s.io` | `selfsubjectaccessreviews`, `selfsubjectrulesreviews` | `create` | `LH-F-024` |
| `coordination.k8s.io` | `leases` | `get`, `list`, `watch`, `create`, `update`, `patch` | Leader-Election (`AR-026`) |

Strikte Minimal-Rechte-Linie (`LH-SEC-001`, `LH-NF-006`): keine
weiteren Rechte. Insbesondere keine `persistentvolumeclaims`-Rechte
im MVP (PVC-Inspektion ist nicht Teil der MVP-Prüfungen aus
`LH-PRI-001`).

### AR-016 — Role im Operator-Namespace

| API-Gruppe | Ressourcen | Verben | Begründung |
| ---------- | ---------- | ------ | ---------- |
| `k-deskflight.geo-terrain.net` | `opendeskpreflightchecks` | `get`, `list`, `watch`, `update`, `patch` | CR-Verarbeitung |
| `k-deskflight.geo-terrain.net` | `opendeskpreflightchecks/status` | `get`, `update`, `patch` | Status-Updates (`LH-F-004`) |

Strikte Minimal-Rechte-Linie: keine `events.create`/`events.patch`
(`LH-F-027` ist v0.2) und keine `configmaps`-Rechte (`LH-F-028` ist
v0.2). Diese Rechte kommen mit den jeweiligen v0.2-Slices in das
RBAC-Set — siehe `§10` Verhältnis zu späteren Phasen.

### AR-017 — ServiceAccount

`k-deskflight-operator` im Operator-Namespace. Bindings:

- ClusterRoleBinding `k-deskflight-operator` → ClusterRole
  `k-deskflight-operator-cluster` (AR-015).
- RoleBinding `k-deskflight-operator` im Operator-Namespace → Role
  `k-deskflight-operator-namespace` (AR-016).
- Optional: zusätzliche RoleBindings in CR-Namespaces, wenn CRs
  außerhalb des Operator-Namespace liegen. Konkretisierung im
  Pflichtenheft.

### AR-018 — SelfSubjectAccessReview-Right

`authorization.k8s.io/selfsubjectaccessreviews` und
`/selfsubjectrulesreviews` im ClusterRole (AR-015) — beide nur mit
`create`-Verb (Standard für SAR-APIs). Damit erfüllt `LH-F-024`/
`LH-AK-016`: der Operator kann seine eigenen Rechte gegen die
aktivierten Prüfungen abgleichen, ohne weitere Rechte zu erhalten.

---

## 8. Build und Release

### AR-019 — Dockerfile-Stages

Multi-Stage-`Dockerfile` analog `m-trace/apps/api/Dockerfile`:

1. `deps` — `golang:1.26`-Base, `go.mod`/`go.sum` kopieren,
   `go mod download`.
2. `lint` — `golangci/golangci-lint:v2.x-alpine`,
   `golangci-lint run ./...` (`LH-QG-001`).
3. `test` — von `deps`, `go test ./...` (`LH-QG-002`).
4. `coverage` — von `deps`, `go test -cover` mit
   `scripts/coverage-gate.sh`-Schwelle (`LH-QG-003`).
5. `build` — von `deps`, statischer Binary-Build.
6. `runtime` — `distroless/static`-Base, Binary kopieren.

### AR-020 — Makefile-Target-Anker

| Target | Wirkung | LH-Anker |
| ------ | ------- | -------- |
| `make build` | Operator-Binary lokal bauen | `AR-019` |
| `make lint` | `docker build --target lint` | `LH-QG-001` |
| `make test` | `docker build --target test` | `LH-QG-002` |
| `make coverage-gate` | Coverage-Schwelle prüfen | `LH-QG-003` |
| `make gates` | Pflicht-Inner-Loop-Bündel | `LH-QG-009` |
| `make security-gates` | `govulncheck` + Trivy | `LH-QG-009` |
| `make image-build VER=X.Y.Z` | Container-Image lokal bauen | `AR-022` |
| `make image-publish` | Image-Publish nach GHCR (Approval-Gated) | `ADR 0011 §2.5` |
| `make release-guard VER=X.Y.Z` | Pre-Release-Konsistenzprüfung | `ADR 0011 §2.5` |

Konkrete Target-Implementierungen entstehen im M1-Slice-Plan,
adaptiert von `/Development/m-trace/Makefile`.

### AR-021 — CI-Workflow

GitHub Actions (impliziert durch `ADR 0011 §2.6` GHSA-Hosting). Zwei
parallele Jobs pro PR:

- `gates`: `make gates` — Lint, Test, Coverage, Boundary,
  Generated-Drift, Doc-Refs (`LH-QG-009`).
- `security-gates`: `make security-gates` —
  `govulncheck` + Image-Scan (`LH-QG-006`/`LH-QG-007`).

DCO-Check (`ADR 0011 §2.4`) läuft als zusätzlicher Workflow-Job
(z. B. `probot/dco`-App). Konkrete `.github/workflows/*.yml`-Inhalte
entstehen mit M1.

### AR-022 — Image-Tagging und -Distribution

- Default-Registry: `ghcr.io/pt9912/k-deskflight` (Folge von
  `ADR 0011 §2.6` und `ADR 0004`).
- Release-Tags: `vX.Y.Z` (SemVer 2.0.0).
- Branch-Builds: `vX.Y.Z-<commit-sha-short>` pro `main`-Commit.
- Pre-Release: `vX.Y.Z-rc.N`.

---

## 9. Test-Konzept (Skizze)

### AR-023 — Unit-Tests

- `internal/domain/*` ohne Cluster-Abhängigkeit. Pure Funktionen,
  hohe Coverage-Zielsetzung (≥ 95 % erreichbar).
- `internal/application/*` mit Port-Mocks (Tabellengetriebene
  Tests, generierte Mocks via `mockery` oder handgeschriebene
  Test-Doubles — Pflichtenheft).
- Hexagonal-Layer macht das ohne Cluster trivial.

### AR-024 — Integration-Tests via `envtest`

- `controller-runtime/pkg/envtest` startet eine echte kube-apiserver-
  Instanz lokal.
- Tests für jede MVP-Check-Implementierung: passed-Case + failed-
  Case (`LH-AK-005..009`).
- Laufzeit: erträglich (Sekunden bis wenige Minuten); Teil von
  `make test` / `make gates`.

### AR-025 — Smoketests via `kind`

- Eigene Target-Gruppe (z. B. `make smoke-kind`), opt-in.
- Echter Cluster, vollständiger Lifecycle (Install, CR-Apply,
  Status-Read, Cleanup).
- Nightly im CI; nicht PR-blockierend.

---

## 10. Verhältnis zu späteren Phasen

| Phase | Was aus dieser Architektur trägt | Was hinzukommt |
| ----- | -------------------------------- | -------------- |
| MVP v0.1 (`LH-REL-001`) | alle hier festgelegten `AR-*` (außer den explizit als v0.2+ markierten ConfigMap- und Event-RBAC-Anteilen) | konkrete Spec-Felder, Probe-Implementierungen, Test-Mocks |
| v0.2 (`LH-PRI-002`) | Helm-Chart als alternativer Distributions-Pfad (`ADR 0005`); CRD bleibt `v1alpha1` | DNS-/TLS-/Netzwerk-Check-Module unter `adapter/check/`; ConfigMap-Report-Adapter; Events-Emission; Domänen-Metriken. RBAC-Erweiterung gegenüber MVP-`AR-016`: `core/events` mit `create`+`patch` (`LH-F-027`) und `core/configmaps` mit `get`/`list`/`create`/`update`/`patch` (`LH-F-028`). |
| v0.3+ (`LH-PRI-003`) | mit-Auth-Checks bauen auf der bestehenden Check-Interface auf (`AR-012`); RBAC-Konzept (`AR-015`) wird um Secret-Read-Rechte erweitert | PostgreSQL-Adapter, S3-Adapter, evtl. CRD-Schema-Erweiterung auf `v1alpha2` oder `v1beta1` |

---

## 11. Offene Architektur-Punkte

| Kennung | Offener Punkt | Status |
| ------- | ------------- | ------ |
| `AR-OP-001` | Konkrete CRD-Spec-Feld-Typen und kubebuilder-Marker (validation, defaulting, printcolumns) | offen — Pflichtenheft (`LH-VM-002`) |
| `AR-OP-002` | Wahl zwischen `mockery`-generierten Mocks und handgeschriebenen Test-Doubles für `internal/port/*` | offen — Pflichtenheft |
| `AR-OP-003` | Worker-Pool-Modell für Check-Parallelisierung (`AR-009 §4` Execute-Phase): sequenziell vs. begrenzt-parallel | offen — Pflichtenheft, vor M3 |
| `AR-OP-004` | Konkrete kubebuilder-Marker für RBAC-Generierung (`AR-015`/`AR-016` Quelle) — Hand-pflege vs. `+kubebuilder:rbac:...`-Annotationen am Controller | offen — M2-Slice-Plan |
| `AR-OP-005` | Operator-Namespace-Konvention (`k-deskflight-system` vs. `k-deskflight-operator` vs. anwender-wählbar) | offen — M1-/M2-Slice-Plan |
| `AR-OP-006` | OTel-Integration: Tracing-Spans im Reconcile-Pfad ja/nein | offen — v0.2-Slice, koordiniert mit `ADR 0007` |
| `AR-OP-007` | Conversion-Webhook für künftige Versionssprünge (`AR-008`) — implementieren oder via Re-Apply lösen | offen — Folge-ADR zu `ADR 0006 §4` |

Die `AR-OP-*`-Punkte werden bei Aktivierung der jeweiligen Slice in
die zugehörigen Slice-Pläne überführt oder, wenn übergreifend, als
eigene Folge-ADRs aufgegriffen.
