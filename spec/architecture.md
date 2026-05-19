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
  Retry-Budgets (`retry backoff policy`, `maxRetryAttempts`), sofern nicht in
  `AR-009`/`AR-026` festgelegt — Pflichtenheft und Slice-Pläne.
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
  Trennung: `hexagon/domain/`, `hexagon/application/`, `hexagon/port/`,
  `internal/adapter/`. Damit
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
│   ├── hexagon/
│   │   ├── domain/                  # Reine Domänentypen, keine k8s-Abhängigkeit
│   │   │   ├── check.go             # Check-Interface, Result, Severity
│   │   │   └── profile.go           # Profile-Konstanten, Default-Werte
│   │   ├── application/             # Use-Cases / Reconciler-Orchestrierung
│   │   │   ├── reconciler.go        # Reconcile-Loop (high-level)
│   │   │   └── aggregator.go        # Status-Aggregation per LH-F-031
│   │   ├── port/                    # Interfaces, von application konsumiert
│   │   │   ├── kubernetes.go        # KubernetesAPI-Interface
│   │   │   ├── checkregistry.go     # CheckRegistry-Interface
│   │   │   └── clock.go             # Clock-Interface (Test-Inversion)
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
- `cmd/operator` ist die Wiring-Schicht: importiert
  `internal/hexagon/application` und `internal/adapter`, instanziiert
  konkrete Adapter und injiziert sie in den Reconciler. Der fachliche
  Coverage-Scope (`LH-QG-003`) fokussiert auf `internal`-Layer; die
  Konfigurations- und Bootstrap-Pfade in `cmd/operator` (env/flag-Parsing,
  Start-Guards, Strict-Modus) werden zusätzlich über gezielte
  Startup-/Smoke-Tests abgesichert.
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
        - '**/internal/hexagon/domain/**'
      deny:
        - pkg: k8s.io
          desc: domain layer must not depend on Kubernetes libraries (use port)
        - pkg: sigs.k8s.io
          desc: domain layer must not depend on controller-runtime
        - pkg: github.com/pt9912/k-deskflight/internal/hexagon/application
          desc: domain must not depend on application (AR-004)
        - pkg: github.com/pt9912/k-deskflight/internal/hexagon/port
          desc: domain must not depend on port (AR-004)
        - pkg: github.com/pt9912/k-deskflight/internal/adapter
          desc: domain must not depend on adapter (AR-004)
    application-no-adapter:
      list-mode: lax
      files:
        - '**/internal/hexagon/application/**'
      deny:
        - pkg: github.com/pt9912/k-deskflight/internal/adapter
          desc: application must depend on ports, not on adapter implementations (AR-004)
    port-no-application:
      list-mode: lax
      files:
        - '**/internal/hexagon/port/**'
      deny:
        - pkg: github.com/pt9912/k-deskflight/internal/hexagon/application
          desc: ports are abstractions, must not depend on application (AR-004)
        - pkg: github.com/pt9912/k-deskflight/internal/adapter
          desc: ports define abstractions, must not depend on adapter implementations (AR-004)
    adapter-no-application:
      list-mode: lax
      files:
        - '**/internal/adapter/**'
      deny:
        - pkg: github.com/pt9912/k-deskflight/internal/hexagon/application
          desc: adapter implements port, must not call into application directly (AR-004)
    api-no-internal:
      list-mode: lax
      files:
        - '**/api/v1alpha1/**'
      deny:
        - pkg: github.com/pt9912/k-deskflight/internal/hexagon
          desc: api/v1alpha1 declares CRD types only — must not depend on internal/* (AR-004)
```

`cmd/` ist bewusst ohne `depguard`-Restriktion — die Wiring-Schicht
muss alle Layer importieren dürfen, um sie zu verdrahten (`AR-004`).
Das ist die einzige Ausnahme; alle anderen Pakete tragen mindestens
eine Deny-Regel.

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
  resources, rbac). MVP-Set siehe `LH-PRI-001`.
- `Status.Phase`: Enum (`LH-F-006`: `Pending`/`Running`/`Passed`/
  `Warning`/`Failed`/`Unknown`).
- `Status.ObservedGeneration`: Kopie von `metadata.generation`, auf die sich
  der zuletzt vollständig ausgeführte Reconcile bezieht.
- `Status.Summary`: Aggregat (`LH-F-007`) mit den Feldern `passed`,
  `warning`, `failed`, `unknown`, `checksTotal` sowie `lastChecked`.
- `Status.Conditions`: Kubernetes-ähnliche Condition-Liste mit
  `Type`/`Status`/`Reason`/`Message`/`LastTransitionTime`
  und explizitem `Severity`-Feld (`info|warning|critical`) für die
  Check-Schweregrad-Auswertung. `Severity` ist Pflichtfeld (Default `info`,
  wenn der Produzent nichts setzt).
  `Reason`/`Severity`/`Message` folgt
  `LH-F-031`/`LH-F-032`.

### AR-007 — Generated-Drift-Mechanik

`controller-gen` (kubebuilder-Tooling) erzeugt:

- `zz_generated.deepcopy.go` im `api/v1alpha1/`-Verzeichnis.
- CRD-YAML unter `config/crd/`.
- RBAC-Skelette unter `config/rbac/` (basierend auf
  kubebuilder-Markern im Controller-Code).

**Marker-Platzierung:** `+kubebuilder:rbac:...`-Annotationen werden
**direkt am `Reconcile`-Receiver** in
`internal/hexagon/application/reconciler.go` platziert. Das Hybrid-Layout
(`AR-003`) verlegt den Reconciler aus dem kubebuilder-üblichen
`internal/controller/`-Pfad nach `internal/hexagon/application/`;
`controller-gen rbac` bekommt den Marker-Pfad über sein
`paths`-Argument im `Makefile`-Target, z. B.
`controller-gen rbac:roleName=k-deskflight-operator-cluster
paths=./internal/hexagon/application/...`. Die Platzierung ist damit
verbindlich; offen bleibt nur das konkrete Marker-Set pro Ressource
(siehe `AR-OP-004`).

Diese Artefakte sind committet, aber von Hand nicht editierbar.
`LH-QG-005` (Generated-Drift-Gate, `ADR 0012 §2.7`) regeneriert sie
bei jedem CI-Lauf und vergleicht via `git diff --exit-code`.

### AR-008 — Versioning-Strategie

Initial `v1alpha1` (`ADR 0006 §2.3`). Schema-Brüche zwischen
MVP-Releases sind nach Kubernetes-Konvention für `v1alpha1`
zulässig. Für produktive Instanzen wird dieses Risiko über einen
bewussten Migrationspfad abgefedert:

- CRD-Brüche im Release werden mit Release-Notes, betroffenen Feldern und
  Re-Apply-Anleitung dokumentiert.
- Der Operator transformiert bestehende CRs **nicht** automatisch.
- Migration erfolgt kontrolliert: Reconcile kurz temporär stoppen,
  betroffene CRs exportieren, auf neues Spec/CRD setzen, Reconcile
  wieder starten.

Migration auf `v1alpha2` oder `v1beta1` ist Folge-ADR-Stoff
(siehe `ADR 0006 §4`).

Conversion-Webhooks sind **nicht** Bestandteil des MVP. Ein
Versionssprung im MVP-Zeitfenster wird über CR-Re-Apply gelöst, nicht
über Server-seitige Conversion.

---

## 5. Controller-Reconciler

### AR-009 — Reconcile-Pfad

Der Reconciler lebt in `internal/hexagon/application/reconciler.go` und
folgt einem deterministischen sechs-Phasen-Pfad pro Reconcile-Lauf:

1. **Initialize Run Context + Fetch** — Für jeden Reconcile-Lauf wird ein
   Kinder-Kontext `runCtx` erzeugt, der hart durch
   `RECONCILE_TIMEOUT_SECONDS` gedeckelt ist (nach den Normalisierungsregeln
   aus dieser Architektur). Der Reconcile verarbeitet den CR ausschließlich in
   diesem Kontext.
   - `runCtx` wird als `context.WithTimeout(ctx, reconcileTimeout)` aufgebaut,
     daraus wird `runDeadline, hasDeadline := runCtx.Deadline()` verwendet.
   - Die Architektur nimmt `hasDeadline=true` als Normfall an.
   - Falls `hasDeadline=false` auftritt (Sollbruch), wird deterministisch
     `runDeadline = time.Now().Add(reconcileTimeout)` gesetzt, damit `resultLimitCh`
     und Folgepfade nicht mit einem ungültigen Zeitfenster arbeiten.
   CR über `client.Get` lesen. Bei `NotFound`: kein Requeue (CR gelöscht).
   Fällt der Laufkontext durch Zeitüberschreitung aus, endet der Reconcile
   deterministisch mit `Phase: Unknown`, `Condition: ReconcileError`,
   `Reason: ReconcileTimeout`. Diese Beendigung hat Präzedenz vor der
   normalisierten Check-Aggregation. Nicht verarbeitete Checks werden als
   `Unknown`/`Timeout` in die Result-Sammlung übernommen.
2. **Cross-Field-Validate** — OpenAPI-Constraints (Enum-Werte,
   Range-Constraints, Pflicht-Felder) werden bereits beim
   `kubectl apply` von der CRD-Schema-Validation geprüft (siehe
   `AR-006`/`AR-007`). Diese Phase ergänzt **Cross-Field-Konsistenz**,
   die das Schema nicht ausdrücken kann — z. B. „`Profile=evaluation`
   schließt Check-Typ X aus", oder „konfigurierte
   `kubernetesVersion.min` liegt im vom Operator unterstützten
   Bereich gemäß `ADR 0009`". Bei Validierungsfehler: Phase `Failed`,
   Condition `SpecInvalid`.
3. **Determine Active Checks** — basierend auf Profile und
   Spec.Checks-Map die zu aktivierenden Check-Instanzen aus der
   `CheckRegistry` (`AR-013`) auflösen.
   - Die Auflösung ist strict: sind in `Spec.Checks` unbekannte Check-Namen
     eingetragen oder ist ein Check nicht im aktivierten Profil erlaubt,
     gilt der Auswahlschritt als ungültig.
   - Die betroffenen Namen werden vor der Fehlerausgabe als eindeutige,
     alphabetisch sortierte Liste verarbeitet.
   - Für unbekannte Namen wird `Reason: UnknownCheck` gesetzt.
   - Für profil-inkompatible Namen wird `Reason: CheckNotAllowedInProfile`
     gesetzt.
   - Reconcile endet in Phase `Failed` mit `Condition: SpecInvalid`,
     deterministischem Fehlertext und führt keine Check-Execution aus.
4. **Execute Checks** — basierend auf der Registry-Auflösung werden die
   Check-spezifischen Werte in `CheckSpec`-Instanzen transformiert und via
   `spec.Kind()` gegen `Check.SpecKind()` validiert.
   - Die gesamte Check-Pre-Execution- und Execution-Pipeline ist panic-härtet:
     `ListByProfile`-Ergebnis, `CheckSpec`-Transformation, `CheckSpec.Validate`
     und `Check.Run` laufen hinter einem gemeinsamen Panic-Boundary (`defer/recover`).
     Bei Panics in diesem Pfad entstehen ein deterministischer Result-Eintrag
     `Status: Unknown`, `Reason: InternalError` und `Message` ohne Status-Datenleck.
   Danach wird die Ausführung durch ein verbindliches Worker-Pool-Modell
   bestimmt (Pflichtenheft-Detail):
   - **Timeout-Vertrag (MVP):**
     - `RECONCILE_TIMEOUT_SECONDS`:
       Default `120`, Min `5`, Max `600`; begrenzt einen kompletten
       Reconcile-Lauf.
     - `CHECK_TIMEOUT_SECONDS`:
       Default `30`, Min `1`, Max `120`; begrenzt die Laufzeit pro Check
       (`domain.Check.Run`).
     - **Cross-Constraint:** `CHECK_TIMEOUT_SECONDS` darf nie die aktuelle gültige
       Reconcile-Laufzeit überschreiten. Falls nach Normalisierung
       `CHECK_TIMEOUT_SECONDS > RECONCILE_TIMEOUT_SECONDS`, wird `CHECK_TIMEOUT_SECONDS`
       auf `RECONCILE_TIMEOUT_SECONDS` gekappt und als
       `Status=Warning + Condition=ConfigurationInvalid` dokumentiert.
     - Konfigurationen werden strikt geparst:
       - parsefaule Werte (leer, nicht-zahlbar, negative Werte) werden auf die
         Defaults normalisiert.
       - Werte `<= 0` werden auf den Default normalisiert.
       - Werte `> Max` werden auf `Max`, Werte `< Min` auf `Min` normalisiert.
       - Für jede betroffene Konfigurationsvariante gilt ein feldspezifischer
         `Reason` für den Status-Condition-Eintrag:
         - `Reason=ReconcileTimeoutConfig` bei Anpassung von `RECONCILE_TIMEOUT_SECONDS`.
         - `Reason=CheckTimeoutConfig` bei Anpassung von `CHECK_TIMEOUT_SECONDS`
           inkl. Cross-Constraint-Korrektur.
         - `Reason=K8SApiQPSConfig` bei Anpassung von `K8S_QPS`.
         - `Reason=K8SApiBurstConfig` bei Anpassung von `K8S_BURST`.
         - `Reason=WorkerPoolSizeConfig` bei Anpassung von `WORKER_POOL_SIZE`.
         - `Reason=LeaderReadinessGraceConfig` bei Anpassung von `leaderReadinessGrace`.
        - `Reason=LeaderReadLeaseErrorToleranceConfig` bei Anpassung von
           `leaderReadLeaseErrorTolerance`.
       - Optional: `OPERATOR_STRICT_CONFIG=true` verwandelt solche
         Normalisierungen in ein Start-Blocking für Operator-Startkonfigurationen
         (z. B. Umgebungs- oder Flag-Parameter, nicht für `Spec.Interval`):
         Operator-Start abgebrochen,
         klarer Startfehler bis die Konfiguration korrigiert ist).
        - `OPERATOR_STRICT_CONFIG=true` gilt zusätzlich auf alle Konfigurationspfade,
         inkl. der oben beschriebenen Cross-Constraint-Korrektur, jedoch nur für
         Operator-Laufzeitkonfiguration (nicht für CR-Inline-Konfigurationen wie
         `Spec.Interval`).
     - Normalisierung/Parsefehler erzeugen `Status=Warning +
       Condition=ConfigurationInvalid`, den beschriebenen feldspezifischen
       Reason-Code und die gewählte Normalisierung.
      - `Result`-Ausgabe bei Überschreitung bleibt: `Unknown` + `Reason: Timeout`.
       - Jede Check-Ausführung läuft zusätzlich über einen hart begrenzten
         `runCheckWithTimeout`-Ausführungspfad: `Check.Run(runCtx, spec)` wird in
         einem separaten Goroutine gestartet, der Reconciler wartet auf Ergebnis,
         `runCtx.Done()` oder Check-Timeout. Bei Timeout wird sofort `Unknown` +
         `Reason: Timeout` gemappt und die Check-Execution wird als abgeschlossen
         markiert.
       - Nicht-kooperative Implementierungen werden über Adapter-Hardening abgefangen:
         cancelbare API-Clients, begrenzte I/O-Timeouts und zentrale Fehlerpfade.
       - Die Timeout-Guards verhindern Blockade des Reconcile-Laufes; ein
         eventuell weiterlaufender Worker wird nicht mehr beobachtet und darf
         keine Ressourcen leaken. `domain.Check`-Implementierungen,
         die den Kontext nach Timeout systematisch ignorieren, sind Architekturverstoß
         und werden als `Unknown` + `Reason: Timeout` bewertet; zusätzlich wird
         `deskflight_check_internal_error_total{check_name,kind="noncooperative"}` inkrementiert.
       - **Check-Kontrakt gegen Resource-Leaks (verbindlich):**
         - `domain.Check`-Implementierungen müssen `ctx.Done()` deterministisch
           beachten (inkl. API-/I/O-Calls mit Kontext).
         - Der `adapter/check`-Layer MUSS interne API-Clients mit harten
           I/O-/Dial-Timeouts ausstatten.
         - Nicht-kooperative Checks sind Architekturverstoß; CI prüft das über
           einen Injected-Check mit absichtlich ignorierten Kontextsignalen.
         - Worker, die ihre Task nicht abschließen, können nach Run-Deadline
           nicht blockierend weiterlaufen. Sie haben keinen unbounded Einfluss auf
           Ressourcenkonsum; Drops auf Write-Pfaden erfolgen nur, wenn der
           Run nicht mehr aktiv ist oder die `taskID`/`runID` veraltet ist.
       - Der Check-Worker darf daher niemals blockierend auf `resultCh` schreiben.
         Er schickt Ergebnisse nur per `select`:
         - auf das Task-Abbruchsignal (`taskDoneCh`),
         - auf `runCtx.Done()`-Abbruch,
         - auf die Ergebnis-Zeitschranke (`resultLimitCh`) und
         - auf den Ergebniskanal (`resultCh`-Write),
         - ansonsten über `default` bei veralteter `runID`/`taskID` oder nach Abschluss
           des Runs (`runCtx.Done()`/`resultLimitCh` bereits signalisiert).
         Späte Results nach Timeout werden verworfen, nicht weiter verarbeitet
         und dürfen den Worker nicht blockieren.
       - Der Reconciler wartet nicht auf die echte Rückkehr eines bereits
         time-outed Workers; dieser Worker bleibt isoliert. Er darf kurzfristig
         weiterlaufen, ohne den Reconcile-Run zu blockieren.
         Späte Ergebnisse werden verworfen; Run-Abschluss bleibt deterministisch.
       - Nicht-kooperative Checks müssen zusätzlich als `Unknown` + `Reason:
         Timeout`/`ContextCancelled` abgefangen werden und deterministisch in den
         Reconcile-Status einfließen.
   - **MVP-Entscheidung:** bounded parallel über Worker-Pool (`workerPoolSize`
     konfigurierbar), fallback auf sequenziell bei `workerPoolSize <= 1`.
     Der Wert stammt aus dem Operator-Config (`WORKER_POOL_SIZE` / Flag) mit
     Default `4` und wird in `Fetch` validiert:
     - `< 1` wird auf `1` normalisiert.
     - `> 32` wird auf `32` normalisiert.
     - Bei `OPERATOR_STRICT_CONFIG=true` blockiert jede erzwungene
       Normalisierung den regulären Betriebsstart.
     Jede Normalisierung wird als Konfigurationsfehler mit `Status=Warning`
     und `Condition=ConfigurationInvalid` in den Status geschrieben; die
     Ausführung läuft mit dem normalisierten Wert weiter.
       - Der Reconciler erzeugt pro aktivem Check eine immutable
         Task-Repäsentation und pusht diese in ein **gepuffertes Task-Channel**.
         Die Puffergröße ist mindestens `len(activeChecks)`; damit bleiben
         Produzenten nicht blockiert, solange Worker noch starten.
       - Resultat-Sammelpfad:
         - `resultCh` ist per Run mit mindestens `len(activeChecks)` gepuffert.
         - Während eines aktiven Runs darf kein valides Resultat über den Send-Pfad
           verworfen werden; der Reconciler-Thread konsumiert den Kanal
           deterministisch bis `allResultsReceived` oder Abbruch.
         - Für die deterministische Run-Abschlusslogik wird ein dedizierter
           per-run Channel `resultLimitCh` eingeführt:
            - Er wird mit der Restlaufzeit von `runCtx` initialisiert und dient als
              One-Shot-Zeitgeber für die Ergebnis-Aggregation.
            - Typisch: `resultBudget := time.Until(runDeadline); if resultBudget < 0 { resultBudget = 0 }`;
             `resultLimitTimer := time.NewTimer(resultBudget)`  
             `resultLimitCh := resultLimitTimer.C`  
             `defer func() { if !resultLimitTimer.Stop() { select { case <-resultLimitTimer.C: default: } } }()`
             (auch bei frühem Abschluss durch `allResultsReceived`).
         `runDeadline` ist die in `runCtx` hinterlegte (oder substituierte)
         Deadline aus Schritt 1.
       - `runCtx.Done()` ist der harte Abbruchpfad für Check-Execution,
         API-Aufrufe und weitere Blockaden; `resultLimitCh` ist die
         deterministische Abschlussmarke der Ergebnis-Sammelphase.
       - Wird `resultLimitCh` ausgelöst, ist der Run finalisiert: offene Tasks
         werden als Timeout/ContextCancelled markiert, weitere Worker-Sends
         werden verworfen.
   - Die erwartete Ergebnisanzahl ist `expectedResults := len(activeChecks)`.
     Der Reconciler verwaltet:
     - `dispatchedTaskIDs`: Menge aller ausgesandten Tasks
     - `resultTaskIDs`: Menge bereits accepted Ergebnis-Einträge
     - `timedOutTaskIDs`: Menge synthetisch beendeter Tasks nach Laufzeitgrenzen.
   - Für jeden Task wird exakt ein Ergebnis benötigt (`resultTaskIDs ∪
     timedOutTaskIDs` == `dispatchedTaskIDs`).
   - Resultate laufen ausschließlich über ein **gepuffertes Ergebnis-Channel**
     (`checkResultEnvelope` mit `taskID`, `runID`, `domain.Result`, Check-Name),
     konsumiert durch einen single-threaded Aggregator im Reconcile-Thread.
     - Bei normaler Abarbeitung werden echte Ergebnisse accepted, wenn `runID`
       passt und die Task-ID noch nicht final ist.
     - Bei `runCtx.Done()` oder `resultLimitCh` werden offene Tasks als
       `Unknown` + `Reason: Timeout`/`ContextCancelled` deterministisch
       nachgefüllt und in `timedOutTaskIDs` markiert.
     - `allResultsReceived` ist erfüllt, wenn
       `len(resultTaskIDs) + len(timedOutTaskIDs) == expectedResults`.
     - Der Ergebnis-Kanal wird nicht primär zur Laufzeitsteuerung geschlossen.
       Der Reconciler beendet die Ergebnisaufnahme deterministisch über das
       Zusammenspiel aus `allResultsReceived` und
       `runCtx.Done()`/`resultLimitCh`; offene Worker-Kanäleignisse nach
       Abschluss werden verworfen.
   - Alle Worker und Kanäle werden mit demselben Reconcile-`runCtx`
     orchestriert: Bei `runCtx.Done()` brechen sie frühzeitig ab, geben
     `Unknown`/`Reason=ContextCancelled` zurück und beenden sich deterministisch.
     Der Aggregator beendet die Aufzeichnung, sobald `allResultsReceived` true ist
     oder `runCtx.Done()`/`resultLimitCh` ausgelöst wird.
     Späte Sends nach Abschluss werden verworfen.
   - Kein geteilter mutable Check-Zustand im Adapter-/Reconcile-Pfad, um
     Datenrennen auszuschließen.
   - `workerTaskRunner` verwaltet `sync.WaitGroup` für Worker-Hygiene, startet
     Worker nur im gültigen Pool-Bereich und schließt das Ergebnis-Channel nicht
     am unendlichen Join von potenziell blockierenden Workern, sondern via
     deterministischem Abschlusskriterium (`allResultsReceived` oder
     `runCtx.Done()`/`resultLimitCh`).
   - Pro Reconcile-Lauf wird ein monotones `runID`/`epoch` erzeugt. Der
     Sender darf auf `resultCh` nur mit passendem `runID`-Match schreiben.
   - Der Send-Pfad ist non-blocking (`select`) über:
     - `case <-taskDoneCh:` (Task lokal abgeschlossen)
     - `case <-runCtx.Done():` (Reconcile-Kontext beendet)
     - `case <-resultLimitCh:` (globale Run-Deadline/Timeout)
     - `resultCh <- envelope`
     - `default:` (Drop nur bei nicht-aktiven Runs oder veralteten `runID`/`taskID`)
     - Late/abweichende Sends (falsche `runID`, `taskDone` bereits gesetzt oder
       `resultLimitCh` bereits signalisiert) werden verworfen.
   - Worker-Sender verwenden den Guard `taskID` + `runID`, damit verspätete
     Sends eindeutig als veraltet erkannt und verworfen werden.
   - Timeout-/Fehlerbehandlung wird zentral im Reconcile-Executor abgefangen
     und als `Result` nach außen gegeben (`Unknown` bei Laufzeit- oder
     Kontextabbruch).
   - Pro Check: `Run(ctx, spec) Result` ohne Cluster-Mutation.
5. **Aggregate** — Schweregrade auf die Gesamtphase mappen
   (`LH-F-031`/`ADR 0010 §2.3`).
   - Wenn `ReconcileError/Reason=ReconcileTimeout` bereits gesetzt ist,
     bleibt die Ergebnis-Phase deterministisch `Unknown`, unabhängig von
     teils erfolgreichem Check-Outcome.
6. **Update Status** — `client.Status().Patch` (mit `client.MergeFrom`) von
   `Phase`/`Summary`/`ObservedGeneration`/`Conditions`. Konflikte
   (resourceVersion)
   werden mit Re-Fetch + Retry behandelt.
   - Die Status-Aktualisierung ist idempotent: nur wenn sich `Phase`,
     `Summary`, `ObservedGeneration` oder `Conditions` tatsächlich ändern,
     wird geschrieben.
   - Der Original-Status wird vor dem Update kopiert; bei Gleichstand
     (z. B. via `equality.Semantic.DeepEqual`) wird keine Änderung
     persistiert.
   - Bei Änderung wird `client.Status().Patch` mit
     `client.MergeFrom` bevorzugt eingesetzt, um unnötige Self-Writes zu
     reduzieren.
   - `ObservedGeneration` wird auf die aktuelle `metadata.generation` des
     zuletzt verarbeiteten Objekts gesetzt.

### AR-010 — Wiederholintervall (`LH-F-025`)

`Spec.Interval` (Default Pflichtenheft, vorgeschlagener Startwert
`5m`) steuert `RequeueAfter` am Ende des Reconcile-Laufs.

- `Interval` ist als Duration (z. B. `5m`, `30s`, `1h`) in der Spec
  vorgesehen und wird wie folgt normalisiert:
  - Default `5m`, wenn Feld leer ist.
  - `min=30s`, `max=24h`.
  - Nicht parsebare Werte und Werte außerhalb des Bereichs werden auf den
    nächstgelegenen erlaubten Wert geklemmt.
  - Jede Normalisierung wird als `Status=Warning` + `Condition=ConfigurationInvalid`
    dokumentiert.
  - `OPERATOR_STRICT_CONFIG` ist auf dieser Ebene nicht restart-blockierend.
    Für CR-spezifische `Spec.Interval`-Fehler bleibt der Reconcile deterministisch
    lauffähig; die normale Normalisierung wird verwendet.
  - Ein ungültiger `Spec.Interval` verhindert keinen Lauf; der normalisierte Wert
    wird deterministisch verwendet.
  - `OPERATOR_STRICT_CONFIG` blockiert nur harte Fehler in der
    Operator-Laufzeitkonfiguration (ENV/Flags, Start-Guards, Client-/Worker-
    Limits); CR-Ebene (insbesondere `Spec.Interval`) bleibt nicht start-blockierend.

#### AR-010.1 — Konfigurationsklassifizierung und Fehler-Scoping

- **Runtime-Konfiguration (`Operator-Kontext`):** `RECONCILE_TIMEOUT_SECONDS`,
  `CHECK_TIMEOUT_SECONDS`, `K8S_QPS`, `K8S_BURST`, `WORKER_POOL_SIZE`,
  `LEADER_READINESS_GRACE`, `LEADER_READ_LEASE_ERROR_TOLERANCE`,
  `OPERATOR_EXPECTED_REPLICA_COUNT`, `--expected-replica-count`,
  `--leader-elect`, Single-Pod-Topologie-Guard.
  - Parse-/Clamp-Fehler werden als `ConfigurationInvalid` gemeldet.
  - Bei `OPERATOR_STRICT_CONFIG=true` und harter Fehlersituation wird der
    Operatorstart blockiert.
- **CR-konfigurationswerte (`Spec`-Scope):** `Spec.Interval`, `Spec.Profile`,
  `Spec.Checks` und check-spezifische Felder.
  - Parse-/Clamp-Fehler bleiben liveness-sicher: der Reconcile läuft mit
    normalisiertem Wert weiter.
  - Sie führen nie zu einem Operatorstart-Abbruch; stattdessen werden
    `SpecInvalid`/`ConfigurationInvalid` deterministisch im Reconcile-Status
    ausgedrückt.

Diese Trennung ist verpflichtend für Reason-zu-Status-Mapping und die
Interpretation von `OPERATOR_STRICT_CONFIG`.

Auslösung (`LH-F-026`) erfolgt weiterhin durch Anwender-Edit am CR
(Annotation-Bump oder Spezifikationsänderung) — zusätzlich zu den
normierten `RequeueAfter`-Planungen über den controller-runtime-Watch.

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
  - Fehlerpräzedenz ist verbindlich: `ReconcileTimeout` hat absolute Priorität.
  - Primäre Fehlerursache gewinnt:
    - Ist der Lauf bereits in `Reason=ReconcileTimeout` (durch `runCtx.Done()` oder
      `resultLimitCh`) gefallen, bleibt diese `Reason` erhalten.
    - Sonst wird bei Reconciler-Panic `Reason=ReconcilePanic` gesetzt.
- `SpecInvalid` ist ausschließlich für inhaltliche Spec-Fehler vorgesehen
  (z. B. Cross-Field-Validierung), `ConfigurationInvalid` nur für
  Operator-/Ausführungs-Konfigurationsfehler (z. B. Bereichsüberschreitung
  von `workerPoolSize`).

### AR-026 — Leader-Election und Replica-Modell

Der `controller-runtime`-Manager wird konfigurierbar gestartet.
Standardmäßig ist `LeaderElection` aktiv:
`Manager.Options.LeaderElection = !leaderElectOff` mit Flag/Env
`--leader-elect` (`true`=Default, optional `false`).
`LeaderElectionID = "k-deskflight-operator"` (oder äquivalent) und
`LeaderElectionNamespace = <Operator-Namespace>` (siehe `AR-016`) gelten dabei.

Damit gilt:

- **`--leader-elect=true` (MVP-Standard):**
  - Mehrere Replicas im selben Deployment sind erlaubt und erzeugen
    genau eine aktive Leader-Replica, die übrigen Replikate bleiben Standby.
  - Kein Doppel-Reconcile, keine Status-Konflikte durch den lokalen Reconcile-Pfad
    (`AR-009 §6`; resourceVersion-Konflikte bleiben auf legitime Konfliktszenarien beschränkt).
  - **Failover** erfolgt über das Standard-Lease-Renewal-Schema
    (`coordination.k8s.io/leases`, Default-Lease-Duration 15 s,
    Renew-Deadline 10 s, Retry-Period 2 s — konkrete Werte gehören
    ins Pflichtenheft, controller-runtime-Defaults sind akzeptabel).
  - `Manager.Options.MaxConcurrentReconciles` begrenzt die interne Parallelität
    (Default `4`, Max `8`).
  - Readiness bleibt leader-basiert: nur der aktuelle Leader gilt als bereit
    (siehe AR-027).
- **`--leader-elect=false` (Single-Pod-Modus):**
  - Diese Kombination ist nur als Debug-/isolierter Betriebsmodus vorgesehen.
  - Der Start ist hart geblockt, wenn der Single-Pod-Guard nicht erfüllt ist:
    `OPERATOR_EXPECTED_REPLICA_COUNT` / `--expected-replica-count`
    dokumentiert den zulässigen Modus, Default `1`.
    - Werte <1 werden auf `1` normalisiert.
    - Werte >1 sind im deaktivierten Leader-Modus ein harter Konfigurationsfehler:
      Start der Operator-Prozesse wird mit Fehler abgebrochen.
    - Der Start-Guard wird in `cmd/operator/main.go` zentral durchgesetzt.
    - Zusätzlich erfolgt eine Topologieprüfung der tatsächlich laufenden Pods:
      - Ermittlung der Operator-Pods über Namespace + identische Pod-/Deployment-
        Label, optional über Deployment-Namenderivation.
      - Bei `OPERATOR_EXPECTED_REPLICA_COUNT=1` und aktivem `--leader-elect=false`
        darf die laufende aktive Pod-Anzahl nicht >1 sein.
      - Andernfalls bricht der Start mit `Reason=SinglePodTopologyMismatch`
        und Operator-NotReady auf.
      - Der Guard wird beim Start geprüft und regelmäßig revalidiert (Debounce,
        um Start-Flapping zu vermeiden).
  - Die Kombination aus `--leader-elect=false` + `OPERATOR_EXPECTED_REPLICA_COUNT=1`
    ist ein optionaler Single-Pod-Modus für Debug/isolierte Test-Deployments und
    kein aktives HA-Vorhaben.
  - Bei deaktivierter Leader-Election ist der Readiness-Check operativ
    reduziert (`isLeader()==false`, bis der Manager gestartet und Cache-Sync
    abgeschlossen ist; danach deterministisch `true`).

Zusätzlich gilt:

- Zusammengesetzt ergibt sich die obere Grenze parallel aktiver
  Check-Worker pro Operator-Instanz als:
  `MaxConcurrentReconciles * workerPoolSize_normalized`.
- Der Kubernetes-Client nutzt explizit `RESTClientConfig`-Rate-Limits
  (`QPS=10`, `Burst=20` als konservativer Standard). `K8S_QPS` und
  `K8S_BURST` haben harte Grenzen:
  - `K8S_QPS`: Min `1`, Max `100`, Default `10`.
  - `K8S_BURST`: Min `1`, Max `200`, Default `20`.
  - `K8S_BURST` wird mindestens auf `ceil(K8S_QPS)` normalisiert.
  OOB-Werte (inkl. Parsefehler) werden geclamped und als
  `Status=Warning + Condition=ConfigurationInvalid` dokumentiert.
- API-Throttling ist deterministisch abgesichert: der Standard-Rate
  Limiter (`TokenBucket`) bleibt aktiv und arbeitet mit bounded
  exponential backoff.
  Bei dauerhafter Exzessexposition (`TooManyRequests` nach Exhaustion)
  führt der betroffene Check auf `Result{Status: Unknown, Reason:
  ApiThrottled}`; Reconcile-Result bleibt deterministisch aggregiert.
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

### AR-027 — Liveness- und Readiness-Probes

Der `controller-runtime`-Manager bindet die Standard-HTTP-Endpoints
`/healthz` (Liveness) und `/readyz` (Readiness) auf einem
konfigurierbaren Port (Default `:8081` per controller-runtime, im
Pflichtenheft pinnen). Das Deployment-Manifest unter
`deploy/manifests/` (und später im Helm Chart, `ADR 0005`) verdrahtet
beide Probes:

- **`livenessProbe`** auf `/healthz` mit moderaten Schwellen
  (`initialDelaySeconds: 15`, `periodSeconds: 20`,
  `failureThreshold: 3`) — Restart bei länger anhaltender
  Reconcile-Schleifen-Panik.
- **`readinessProbe`** auf `/readyz` mit straffen Schwellen
  (`initialDelaySeconds: 5`, `periodSeconds: 10`,
  `failureThreshold: 3`) — Operator wird ready, wenn Controller-Manager
  läuft, Cache-Sync abgeschlossen ist und ein expliziter Leader-Check true
  meldet.
  Damit bleiben Standby-Pods bei `replicas>1` unready.
  Die Readiness-Read-Logik ist verbindlich:
   - Der Operator registriert einen benannten Readiness-Check (z. B.
    über `mgr.AddReadyzCheck("leader", ...)`), der auf den aktuellen
    Leader-Status (`LeaderElection`) prüft.
      - Der `leader`-Check ist verbindlich auf ein internes Flag `isLeader()` gemappt.
        Für MVP gilt eine eindeutige Implementierung:
      - `cacheSyncReady` ist ein internes Runtime-Flag:
        initial `false`, wird auf `true` gesetzt, sobald der Controller-Manager-Cache
        mindestens einmal erfolgreich synchronisiert wurde.
      - Ein kleiner `leaderWatcher`-Runnable liest periodisch den Lease-Eintrag
        des Operator-Lease-Namens im Operator-Namespace.
      - `isLeader()` ist `true`, wenn der Manager den Cache-Sync abgeschlossen hat,
        `lease.spec.holderIdentity == POD_NAME`
      und `lease.status.renewTime` innerhalb der Readiness-Frischheit liegt:
      `now - renewTime <= leaderReadinessSlack`, wobei
      `leaderReadinessSlack = leaseDuration + leaderReadinessGrace` gilt.
      `leaderReadinessGrace` ist neu und standardmäßig `5s` (konfigurierbar,
      begrenzt auf mindestens `5s`, max `30s`).
      `leaseDuration` ist die im Manager konfigurierte Dauer; bei
      fehlender expliziter Konfiguration der Runtime-Default aus
      `controller-runtime`.
      - Bei fehlender oder null `renewTime` ist das Ergebnis nur dann `true`,
        wenn der Watcher bereits mindestens eine erfolgreiche Lease-Lektur für den
        aktuellen `POD_NAME` hatte und die letzte erfolgreiche Lease-`Transition`
        innerhalb von `leaderReadinessSlack` liegt.
    - Der Runnable setzt bei Fehlern (Lease nicht lesbar) `isLeader()==false`
      und schreibt einen strukturierten Warn-Log.
      - Transiente Lease-Leseausfälle sind toleriert: bis zu
      `leaderReadLeaseErrorTolerance` (Standard `2`) aufeinanderfolgende
      Lesefehler werden mit `isLeader()==true` gehalten, sofern der letzte
      erfolgreiche Lease-Halt noch innerhalb von
      `leaderReadinessSlack` liegt.
    - Wird die Toleranzgrenze überschritten, wird `isLeader()` deterministisch auf
      `false` gesetzt; anschließend wird erst nach erfolgreicher neuer Lease-Lektur
      wieder auf `true` zurückgerollt.
  - Konfigurierbarkeit dieser Readiness-Parameter:
      - `leaderReadinessGrace` (`--leader-readiness-grace` / `LEADER_READINESS_GRACE`):
        Default `5s`, Min `5s`, Max `30s`.
      - `leaderReadLeaseErrorTolerance` (`--leader-read-lease-error-tolerance` /
        `LEADER_READ_LEASE_ERROR_TOLERANCE`): Default `2`, Min `1`, Max `10`.
      - Parser-/Clamp-Verstoß dokumentiert als `ConfigurationInvalid`; bei
        `OPERATOR_STRICT_CONFIG=true` wird der Operatorstart blockiert.
      - `leader` liefert nur dann `true`, wenn `cacheSyncReady == true`
        und der Lease-Test positiv ist.
      Bei deaktivierter Leader-Election (`--leader-elect=false`) ist der Lease-Check
      nicht aktiv.
      - Readiness ist dann aktivitätsabhängig auf den Controller-Startup:
        `isLeader()` ist `false`, bis der Manager gestartet und der Cache erfolgreich
        synchronisiert ist.
    - Anschließend liefert `isLeader()` deterministisch `true`, sofern die
      Operator-Instanz in gesundem Laufbetrieb ist.
    - Diese Variante ist bewusst als Single-Pod-/Nicht-HA-Verhalten
      dokumentiert.
    - In diesem Modus ist `--leader-elect=false` nur mit
      `OPERATOR_EXPECTED_REPLICA_COUNT/--expected-replica-count=1`
      zulässig; bei anderer Konfiguration ist der Guard hart
      (`cmd/operator/main.go`) und der Operatorstart wird blockiert.
      Deployment-Templates sollten dieses Profil technisch einschränken.
  - Optional kann zusätzlich ein expliziter Alias-Endpunkt wie
    `/readyz/leader` bereitgestellt werden.

Konkrete Werte gehören ins Pflichtenheft; die Default-Werte oben sind
controller-runtime-Standard und tragen den MVP-Pfad.

### AR-028 — Minimale Observability-Metriken und Log-Schema (MVP)

Um reproduzierbare Fehlersuche über `Unknown`/`Timeout`/`Conflict`-Fälle zu
gewährleisten, ist im MVP mindestens Folgendes verpflichtend:

- Die semantische Erfassung der Metriken ist als Architektur-Contract zwingend
  (via internes Observer-Interface).
- Der Export-Kanal (Prometheus/OTel/anderer) ist optional; ein fehlender Export
  darf die Operator-Logik nicht blockieren.

- Metrics (MVP-Minimum):
  - `deskflight_reconcile_total{result_phase="pending|running|passed|warning|failed|unknown",namespace_scope="cluster|namespace"}`
  - `deskflight_reconcile_duration_seconds` (Histogramm)
  - `deskflight_status_update_retries_total` (für resourceVersion-Konflikte)
- `deskflight_check_result_total{check_name, reason, status}`
  - `check_name` ist auf die registrierten Check-Namen beschränkt.
  - `reason` ist auf die in AR-009/AR-011 definierte Reason-Taxonomie begrenzt.
    Unbekannte oder unvollständige Werte werden auf neutrale Reason-Codes
    normalisiert, um Metrik-Kardinalität stabil zu halten.
- `deskflight_check_timeout_total{check_name}`
- `deskflight_check_internal_error_total{check_name, kind}`
- `kind` bei `deskflight_check_internal_error_total` kennt mindestens die Werte:
  `noncooperative` (nicht-kooperatives Kontextverhalten), `panic`,
  `internal` (allg. Laufzeitfehler).
- Log-Kontrakt:
  - Pro Reconcile ein stabiler `reconcileRequestID` (z. B. aus Namespaced Name
    + generation), inkl. Ergebnis-Phase.
  - Explizite Event-Logs für `SpecInvalid`, `ConfigurationInvalid`,
    `ContextCancelled`, `ApiThrottled`, `resourceVersionConflict`.
  - Keine sensiblen Stacktraces im CR-Status; Stacktrace-Dump nur in Logs.

Der Readiness-/Liveness-Handler darf bei nicht initialisiertem Metrics-Stack
funktional arbeiten; Metrik-Export ist optional, Log- und Status-Fallbacks
sind weiterhin Pflicht.

---

## 6. Check-Plugin-Architektur

### AR-012 — Check-Interface

Definiert in `internal/hexagon/domain/check.go`:

```go
package domain

import "context"
import "time"

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
    Name() string
    SpecKind() string                               // Erwarteter Check-Spec-Typ-Token
    Run(ctx context.Context, spec CheckSpec) Result
}

type CheckSpec interface {
    Kind() string
    Validate(ctx context.Context) error
}

// Bei `CheckRegistry`- und Reconcile-Ebene ist vor dem `Run` zu prüfen,
// dass `spec.Kind() == check.SpecKind()`. Bei Abweichung: kein Panic,
// sondern `Unknown` + `Reason=InvalidSpec`. `Run` muss bei
// Kontextabbruch (`ctx.Done()`) deterministisch terminieren.
```

### AR-013 — Check-Registry

Definiert in `internal/hexagon/port/checkregistry.go` (Interface) und
implementiert in `internal/adapter/check/registry.go`.

```go
package port

import "github.com/pt9912/k-deskflight/internal/hexagon/domain"

type CheckRegistry interface {
    Register(c domain.Check)
    Resolve(name string) (domain.Check, bool)
    ListByProfile(profile string, spec map[string]domain.CheckSpec) ([]domain.Check, []CheckSelectionIssue)
}

type CheckSelectionIssue struct {
	Name   string
	Reason string
}
```

`cmd/operator/main.go` registriert beim Start alle MVP-Checks
(KubernetesVersion, StorageClass, IngressClass, certManager,
Resources, RBAC).

Kontrakt:
- `ListByProfile` liefert exakt die aktivierbaren Checks als erste
  Rückgabe.
- Die zweite Rückgabe enthält Tupel `(Name, Reason)` für nicht auflösbare Checks.
  - `Reason` ist stabil und enthält entweder `UnknownCheck` oder
    `CheckNotAllowedInProfile`.
  - Tupel sind dedupliziert, stabil alphabetisch sortiert (nach Name).
- Ist diese Liste nicht leer, bricht der Reconcile als
  `Phase: Failed` + `Condition: SpecInvalid` ab und erzeugt keine
  Check-Execution.
- `Resolve` bleibt für explizite Direktanfragen vorgesehen; der produktive
  aktivierende Pfad läuft über `ListByProfile`.
- Bestehende Call-Sites sind auf `ListByProfile`-Resultate umzustellen.
  Eine direkte, aktive Reconcile-Schleife auf Basis von `Resolve(name)`
  ist nicht mehr zulässig.

### AR-014 — Schweregrad-Aggregation und Conditions-Sortierung

Implementierung in `internal/hexagon/application/aggregator.go`. Mappt eine
Liste von `Result` auf die Gesamtphase nach `LH-F-031`-Tabelle.
Reihenfolge: höchster Schweregrad eines Failed-Results bestimmt die
Phase. `Unknown`-Results aus `ConnectivityUnknown`-artigen Checks
(`ADR 0010 §2.3`) führen zu Gesamtphase `Unknown`, sofern kein
`critical`/`warning`-Fail vorliegt.

**Conditions-Sortierung:** Vor `Status().Update` werden `Conditions`
zuerst **dedupliziert** und dann deterministisch nach `Type`
  (Condition-Name) alphabetisch sortiert. Bei doppelten Types wird ein
  einziger Eintrag rekonstruiert: zuerst höchste Severity
  (`critical > warning > info`), danach neustes `LastTransition`, bei
  Gleichstand stabil nach `Name`.
  - `Severity` ist ein verpflichtendes Eingangs-Contract (`info|warning|critical`) für
    den Aggregatorpfad. Ein leerer/ungültiger Wert muss vor der Übergabe
    normalisiert (in der Producer-Pipeline) werden.
  - Unabhängig von der Ausführungsreihenfolge der Checks
(`AR-009 §4` Sequenz vs. Parallel) bleibt der CR-Status-Diff zwischen
Reconcile-Läufen stabil und reflektiert nur tatsächliche
Zustandsänderungen, nicht Sortierungs-Rauschen.

---

## 7. RBAC-Konzept

### AR-015 — ClusterRole MVP-Minimum

Für die MVP-Prüfungen werden die minimal notwendigen Cluster-Rechte benötigt
(`LH-F-035`, `LH-AK-015`, `LH-NF-006`):

| API-Gruppe | Ressourcen | Verben | Begründung |
| ---------- | ---------- | ------ | ---------- |
| `""` (core) | `nodes` | `get`, `list`, `watch` | Allocatable CPU/Memory, Ready-Status (`LH-F-015`) |
| `k-deskflight.geo-terrain.net` | `opendeskpreflightchecks` | `get`, `list`, `watch` | CR-Verarbeitung/Discovery (`LH-F-004`) |
| `k-deskflight.geo-terrain.net` | `opendeskpreflightchecks/status` | `get`, `update`, `patch` | Status-Update (`LH-F-004`) |
| `storage.k8s.io` | `storageclasses` | `get`, `list`, `watch` | `LH-F-010`/`LH-F-011` |
| `networking.k8s.io` | `ingressclasses` | `get`, `list`, `watch` | `LH-F-012` |
| `apiextensions.k8s.io` | `customresourcedefinitions` | `get`, `list` | `cert-manager.io`-Discovery (`LH-F-013`) |
| `authorization.k8s.io` | `selfsubjectaccessreviews`, `selfsubjectrulesreviews` | `create` | `LH-F-024` |
| `coordination.k8s.io` | `leases` | `get`, `list`, `watch`, `create`, `update`, `patch` | Leader-Election (`AR-026`) |

Strikte Minimal-Rechte-Linie (`LH-SEC-001`, `LH-NF-006`): Keine
weiteren Rechte über die aufgeführten Verben/Typen hinaus.
Insbesondere keine `persistentvolumeclaims`-Rechte im MVP (`LH-PRI-001` enthält PVC-Inspektion nicht).

### AR-016 — Role im Operator-Namespace

**Default-Operator-Namespace:** `k-deskflight-system` (Konvention
analog `cert-manager`, `metallb-system`, `kube-system`).
**Betriebsmodus:**  
1) **Cluster-Wide Mode (Default):** Reconciliation über alle Namespaces.
   Dafür enthält `AR-015` die vollständigen Rechte inkl. `opendeskpreflightchecks`
   und `/status`.
2) **Namespace-Reconcile-Scope Mode (optional):** Scope-Reduktion der automatisch
   beobachteten CRs auf `k-deskflight-system` über
   `--namespace=k-deskflight-system`; dies ist ein **Reconcile-Scope-Modus**
   und kein Tenant-/Security-Isolationsmodell.
   **Kein Sicherheits-Isolationsmodell.**
   Zusätzliche Namespaced-RBAC-Objekte im Operator-Namespace sind optional.
   **Wichtig:** Viele Checks operieren auf clusterweiten Ressourcen
   (`nodes`, `storageclasses`, `ingressclasses`, `customresourcedefinitions`),  
   daher bleibt für diese Ressourcen die Cluster-Lese-Berechtigung in
   der `AR-015`-ClusterRole aktiv. Der Namespace-Reconcile-Scope-Modus reduziert
   primär den Reconcile-Scope (`--namespace`) und die Namespaced-Status-/Spec-
   Rechte, **nicht** automatisch jede clusterweite Leseberechtigung.
   Der Namespace-Reconcile-Scope-Modus ist primär eine Reconcile-Scope-Steuerung
   und kein vollständiges Security-Isolationsmodell.
   - Für echte Tenant-/Namespace-Isolation ist ein eigener Erweiterungsmodus
     vorgesehen: dedizierte Operator-Instanz/Profil mit eingeschränkter
     ClusterRole und ohne allgemeine Cluster-Weit-Lesezugriffe, sofern
     die Check-Profile angepasst werden.

Anwender-Overridebarkeit per Kustomize-Overlay ist möglich; ab v0.2
zusätzlich per Helm-Values (`ADR 0005`). Die nachfolgenden
Role-Definitionen leben in diesem Namespace ausschließlich für den
Namespace-Reconcile-Scope Mode.

| API-Gruppe | Ressourcen | Verben | Begründung |
| ---------- | ---------- | ------ | ---------- |
| `k-deskflight.geo-terrain.net` | `opendeskpreflightchecks` | `get`, `list`, `watch`, `update`, `patch` | CR-Verarbeitung (Namespace-Modus) |
| `k-deskflight.geo-terrain.net` | `opendeskpreflightchecks/status` | `get`, `update`, `patch` | Status-Updates (`LH-F-004`) |

Strikte Minimal-Rechte-Linie: keine `events.create`/`events.patch`
(`LH-F-027` ist v0.2) und keine `configmaps`-Rechte (`LH-F-028` ist
v0.2). Diese Rechte kommen mit den jeweiligen v0.2-Slices in das
RBAC-Set — siehe `§10` Verhältnis zu späteren Phasen.

### AR-017 — ServiceAccount

`k-deskflight-operator` im Operator-Namespace. Bindings:

- **Cluster-Wide Mode (Default):**
  - ClusterRoleBinding `k-deskflight-operator` → ClusterRole
    `k-deskflight-operator-cluster` (AR-015).
- **Namespace-Reconcile-Scope Mode (optional):**
  - Zusätzlich RoleBinding `k-deskflight-operator` im Operator-Namespace →
    Role `k-deskflight-operator-namespace` (AR-016).
  - In diesem Profil bleibt die ClusterRoleBinding für die
    clusterweiten Read-Ressourcen erhalten.

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

1. `deps` — `golang:${GO_VERSION}`-Base (z. B. aus `go.mod`-Version oder
   Projekt-Matrix, empfohlen `golang:1.22.x` als aktuelle Basis),
   `go mod download`.
   Konkreter Build-Pfad: `Dockerfile` enthält `ARG GO_VERSION` (Default
   aus `ARG GO_VERSION=1.22.3`) und validiert diesen ebenfalls im
   `make`-Target gegen `go env GOVERSION`, damit Container- und Modul-Toolchain
   synchron bleiben.
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

- `internal/hexagon/domain/*` ohne Cluster-Abhängigkeit. Pure Funktionen,
  hohe Coverage-Zielsetzung (≥ 95 % erreichbar).
- `internal/hexagon/application/*` mit Port-Mocks (Tabellengetriebene
  Tests, generierte Mocks via `mockery` oder handgeschriebene
  Test-Doubles — Pflichtenheft).
  - `cmd/operator/main.go` als Start-/Bootstrap-Tests:
   `env`/`flag`-Parsing, `RECONCILE_TIMEOUT_SECONDS`/`CHECK_TIMEOUT_SECONDS`/
   `WORKER_POOL_SIZE`/`K8S_QPS`/`K8S_BURST`-Defaults und Verhalten bei
   `OPERATOR_STRICT_CONFIG=true`.
    - zusätzliche Coverage für:
      - `LEADER_READINESS_GRACE`/`leader-readiness-grace`
      - `LEADER_READ_LEASE_ERROR_TOLERANCE`/`leader-read-lease-error-tolerance`
        inkl. Clamp-/Strict-Verhalten und `--leader-elect=false`-Modus.
      - `OPERATOR_EXPECTED_REPLICA_COUNT` als harte Single-Replica-Guard im
        `--leader-elect=false`-Profil und dessen fehlerhafte Konfiguration.
      - `--leader-elect=false` + unerwartetes Replica-Set-Scaling (`replicas > 1`)
        als harte Topologieabweichung im laufenden Betrieb.
      - `SinglePodTopologyMismatch` beim Liveness/Readiness-Verhalten.
  - `AR-009`-Invarianten explizit:
  - harte Abschlussbedingung `allResultsReceived`,
  - runID-/taskID-Guards gegen späte/veraltete Worker-Sends,
  - deterministische finalisierte Abschlusslogik bei `runCtx.Done()`/`resultLimitCh`.
  - `AR-018`-Pfadfälle: Fehler von `SelfSubjectAccessReview` und
    `SelfSubjectRulesReview` werden deterministisch auf Status/Condition
    abgebildet.
- Reconcile-spezifische Robustheitsfälle: `defer/recover` im Reconciler,
  `context`-Timeouts bei Check-Ausführung, `Unknown`-Ergebnisse bei
  Ausführungsfehlern sowie Condition/Phase-Mapping bei diesen Fehlern.
- Hexagonal-Layer macht das ohne Cluster trivial.

### AR-024 — Integration-Tests via `envtest`

- `controller-runtime/pkg/envtest` startet eine echte kube-apiserver-
  Instanz lokal.
- Tests für jede MVP-Check-Implementierung: passed-Case + failed-
  Case (`LH-AK-005..009`).
- Tests für `resourceVersion`-Konflikte beim Status-Update, inklusive
  Retry-Verhalten im Reconciler.
- Integrationstest für `workerPoolSize`-Grenzfälle (Fallback auf Sequenz,
  Begrenzung der Parallelität, sauberer Abbruch bei Timeout/Ctx-Cancel).
- Integrationstest für `--leader-elect=false` bei mehrfacher Pod-Instanz in derselben
  Deployment-Scope (fehlerhafte Topologie) mit erwarteter Guard-Blockade.
- Integrations- oder Chaos-Test für `SelfSubjectAccessReview`/`RulesReview` ohne
  Rechte (erwartetes `Spec`-/`ConfigurationInvalid`-Mapping).
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

**Nicht im MVP-Scope:** Validating- oder Mutating-Webhooks für
`OpenDeskPreflightCheck`. Die OpenAPI-Schema-Validation
(`AR-006`/`AR-007`) plus die Cross-Field-Validation im Reconciler
(`AR-009 §2`) decken die MVP-Pflichten ab. Webhooks kommen erst,
wenn ein konkreter Anwendungsfall sie zwingt — z. B. Defaulting,
das `kubectl apply --dry-run=server` benötigt, oder serverseitige
Spec-Validierung jenseits dessen, was Cross-Field-Validate
ausdrücken kann. Dann eigene Folge-ADR.

---

## 11. Offene Architektur-Punkte

| Kennung | Offener Punkt | Status |
| ------- | ------------- | ------ |
| `AR-OP-001` | Konkrete CRD-Spec-Feld-Typen und kubebuilder-Marker (validation, defaulting, printcolumns) | Geschlossen mit M2 ([`api/v1alpha1/opendeskpreflightcheck_types.go`](../api/v1alpha1/opendeskpreflightcheck_types.go)): vollständige Spec/Status-Typen, `+kubebuilder:validation:Enum`/`Pattern`/`MinLength`, `+kubebuilder:default=production`/`"1.34"`/`info`, Printcolumns `Phase`/`Age` und Shortname `opdc`. M3 ergänzt `KubernetesVersionSpec.Validate` für Cross-Field-Constraints. |
| `AR-OP-002` | Wahl zwischen `mockery`-generierten Mocks und handgeschriebenen Test-Doubles für `internal/hexagon/port/*` | Geschlossen mit M2/M3: **handgeschriebene Test-Doubles**. `fake.NewClientBuilder` aus controller-runtime für den k8s-Client; lokale `stubCheck` und `fakeRegistry` pro Test-File (siehe `internal/hexagon/application/reconciler_test.go`, `internal/adapter/check/*_test.go`). Kein `mockery`-Generator nötig — Test-Doubles bleiben klein und gut lesbar. |
| `AR-OP-004` | Konkrete `+kubebuilder:rbac:...`-Marker-Sets am `Reconcile`-Receiver: genauer Verb-Satz pro Ressource, Marker-Doppelungen bei mehreren API-Gruppen, Konsolidierung mit `AR-015`/`AR-016` (Platzierung in `AR-007` festgelegt) | Geschlossen mit M2 ([`internal/hexagon/application/reconciler.go`](../internal/hexagon/application/reconciler.go)): 8 `+kubebuilder:rbac:`-Marker decken die `AR-015`-MVP-Minimum-Tabelle 1:1 ab; `controller-gen rbac` produziert `config/rbac/role.yaml` deterministisch (Generated-Drift-Gate aktiv). |
| `AR-OP-005` | Anwender-Overridebarkeit des Default-Operator-Namespace `k-deskflight-system` via Kustomize-Overlay (ab MVP) bzw. Helm-Values (ab v0.2, `ADR 0005`) — exakte Override-Mechanik | Geschlossen mit M6 ([`docs/plan/planning/done/slice-M6-metrics-tests-doku.md §2.4`](../docs/plan/planning/done/slice-M6-metrics-tests-doku.md) + [`docs/user/installation.md §4`](../docs/user/installation.md)): Kustomize-Overlay-Pattern mit zwei Pflicht-Patches (Namespace-Resource selbst + ClusterRoleBinding-Subject-Namespace) dokumentiert; alle anderen namespaced Resources werden via `namespace:`-Kustomize-Feld automatisch umgeschrieben. Cluster-Smoke-Pipeline (`scripts/cluster-smoke.sh`) nutzt das gleiche Overlay-Pattern operativ für Image-Override. Helm-Values-Variante kommt mit v0.2 (`ADR 0005`). |
| `AR-OP-006` | OTel-Integration: Tracing-Spans im Reconcile-Pfad ja/nein | offen — v0.2-Slice, koordiniert mit `ADR 0007` |
| `AR-OP-007` | Conversion-Webhook für künftige Versionssprünge (`AR-008`) — implementieren oder via Re-Apply lösen | offen — Folge-ADR zu `ADR 0006 §4` |
| `AR-OP-008` | Harte Namespace-/Tenant-Isolation (separate Operator-Instanz, eingeschränkte ClusterRole, profilierte Check-Matrix) | offen — v0.2/ Folge-ADR |

Die `AR-OP-*`-Punkte werden bei Aktivierung der jeweiligen Slice in
die zugehörigen Slice-Pläne überführt oder, wenn übergreifend, als
eigene Folge-ADRs aufgegriffen.
