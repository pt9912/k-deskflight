# Slice M2 — CRD + Controller-Skeleton

**Status:** Done
**Eröffnet:** 2026-05-17
**Geschlossen:** 2026-05-17
**Vorgänger:** [M1 — Repo & Build-Skeleton (Done)](slice-M1-repo-skeleton.md)
**Nachfolger:** [M3 — Erste Prüfung: Kubernetes-Version](../in-progress/roadmap.md#m3--erste-pr%C3%BCfung--kubernetes-version)
**Bezug:**
[Roadmap §3 M2](../in-progress/roadmap.md#m2--crd--controller-skeleton),
[`spec/architecture.md` §4 (AR-006, AR-007, AR-008), §5 (AR-009), §7 (AR-015..AR-018)](../../../../spec/architecture.md),
[ADR 0006](../../adr/0006-api-gruppe-und-crd-scope.md),
[ADR 0009](../../adr/0009-k8s-versions-support-und-profile-mindestversionen.md),
[ADR 0012 §2.4–§2.7](../../adr/0012-quality-gates.md)

---

## 1. Lieferziel

CRD `OpenDeskPreflightCheck` (`v1alpha1`, namespaced, API-Gruppe
`k-deskflight.geo-terrain.net` gemäß [`ADR 0006`](../../adr/0006-api-gruppe-und-crd-scope.md))
mit Spec/Status-Schema, Profile-Defaults im OpenAPI-Schema, einem
Reconciler-Skelett, der Phase-Transitions `Pending → Running →
Passed` schreibt (ohne Check-Logik, leere Conditions), den minimal
nötigen RBAC-Manifesten und aktivem Generated-Drift-Gate. `depguard`-
Layer-Regeln aus [`AR-005`](../../../../spec/architecture.md) sind
ab dieser Slice scharf geschaltet.

**Was M2 noch nicht macht:** keine Check-Implementierung, kein
Run-Context-/Timeout-Pfad, kein Cross-Field-Validate
(`AR-009`-Phasen 1, 2, 4–6) — kommt ab M3.

---

## 2. Slice-Entscheidungen

### 2.1 Bootstrap-Mechanik

**Hand-schreiben mit kubebuilder-Markern.** Keine `kubebuilder init`-
Scaffolds. Begründung: das AR-003-Hybrid-Layout legt den Reconciler
nach `internal/hexagon/application/`, nicht in den kubebuilder-Default
`internal/controller/`. Scaffolds würden zusätzlichen Diff-Ballast
einziehen (PROJECT-File, hack/-Skripte, Boilerplate-Dockerfile/Makefile)
und davon das meiste müsste wieder verworfen werden. Hand-geschriebene
Typen mit `+kubebuilder:`-Markern + `controller-gen` standalone als
Generator-Tool ist der saubere Pfad.

### 2.2 Reconciler-Scope

**Minimal Pending → Running → Passed.** Genau wie Roadmap §3 M2
verlangt: drei Phasen-Transitions, leere Summary, leere Conditions.
Der volle AR-009-Sechs-Phasen-Pfad (Run-Context, Timeout,
Cross-Field-Validate, Active-Check-Resolve, Sammeln, Aggregieren)
kommt ab M3, sobald der erste echte Check (Kubernetes-Version)
existiert.

### 2.3 controller-gen-Invocation

**Eigener `tools`-Stage im Dockerfile + Bind-Mount-Run für die
Generation.** Diese Mechanik (übernommen aus dem Slice-Aktivierungs-
Briefing) sieht so aus:

- `Dockerfile`: neue Stage `FROM deps AS tools` mit `ARG
  CONTROLLER_GEN_VERSION=v0.21.0` und `RUN go install
  sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION}`.
- `make tools`: baut `$(IMAGE):go-tools` (Wrapper für den Stage,
  einmal cachebar).
- `make manifests`: läuft `docker run` gegen `$(IMAGE):go-tools`
  mit `--user "$(id -u):$(id -g)"` und Bind-Mount `$(CURDIR):/src`,
  damit generierte Dateien nicht als root committet werden.
- `make generated-drift-check`: ruft `make manifests` und prüft
  anschließend `git diff --exit-code` auf den generierten Pfaden
  (`api/v1alpha1/zz_generated.deepcopy.go`, `config/crd/`,
  `config/rbac/`). Verstöße brechen den Build.
- `make gates` wird in dieser Slice um `generated-drift-check`
  erweitert (`gates: build lint test coverage-gate doc-refs
  generated-drift-check`).

Der `controller-gen`-Aufruf nutzt **mehrere `paths`**, weil
kubebuilder-Marker an zwei Stellen sitzen:

```text
controller-gen \
    object:headerFile=hack/boilerplate.go.txt \
    crd \
    rbac:roleName=k-deskflight-operator-cluster \
    paths=./api/... \
    paths=./internal/hexagon/application/... \
    output:crd:artifacts:config=config/crd \
    output:rbac:artifacts:config=config/rbac
```

`./api/...` liefert die `+kubebuilder:object`- und
`+kubebuilder:validation`-Marker für CRD + DeepCopy;
`./internal/hexagon/application/...` liefert die
`+kubebuilder:rbac:...`-Marker am `Reconcile`-Receiver gemäß
[`AR-007`](../../../../spec/architecture.md). Das Marker-Set folgt
[`AR-015`](../../../../spec/architecture.md) (ClusterRole-Minimum für
MVP — `opendeskpreflightchecks` + `opendeskpreflightchecks/status` +
`authorization.k8s.io/selfsubjectaccessreviews`/`...rulesreviews` +
die Watch-Rechte auf Discovery-Ressourcen).

### 2.4 Versionspins

| Komponente | Pin in M2 | Quelle |
| ---------- | --------- | ------ |
| `sigs.k8s.io/controller-runtime` | `v0.24.1` | latest stable (2026-05-12), kompatibel mit K8s 1.34 (`ADR 0009`) |
| `sigs.k8s.io/controller-tools` (controller-gen) | `v0.21.0` | latest stable |
| `k8s.io/api`, `k8s.io/apimachinery`, `k8s.io/client-go` | transitiv über controller-runtime v0.24.1 | go-mod-Auflösung, kein Direkt-Pin |

Pin-Hebung ist Routine ohne ADR ([`ADR 0012 §2.8` Abs. 3](../../adr/0012-quality-gates.md));
größere controller-runtime-API-Brüche (z. B. v1.x) bekommen einen
eigenen ADR-Pfad.

---

## 3. Datei-Inventar

### 3.1 Neue Code-Dateien

| Pfad | Zweck |
| ---- | ----- |
| `hack/boilerplate.go.txt` | Header-Boilerplate für generierte Go-Dateien (MIT, Project-Authors) — kubebuilder-Konvention. |
| `api/v1alpha1/groupversion_info.go` | `+groupName=k-deskflight.geo-terrain.net`-Marker, Scheme-Registrierung, `SchemeBuilder`. |
| `api/v1alpha1/opendeskpreflightcheck_types.go` | Spec/Status-Typen gemäß [`AR-006`](../../../../spec/architecture.md): `Spec.Profile` (Enum `production`/`evaluation`, Default `production`), `Spec.Checks.KubernetesVersion.Min` (Default `"1.34"`), `Status.Phase`, `Status.ObservedGeneration`, `Status.Summary`, `Status.Conditions`. Mit kubebuilder-Markern. |
| `internal/hexagon/application/reconciler.go` | Reconciler-Skelett: `Reconcile(ctx, req) (Result, error)` führt `Get` → `Patch Status.Phase=Pending` → `Patch Status.Phase=Running` → `Patch Status.Phase=Passed`. `ObservedGeneration = metadata.generation`. Keine Check-Logik. RBAC-Marker am Receiver gemäß [`AR-015`](../../../../spec/architecture.md). |
| `cmd/operator/main.go` | **Vollumschrieb** vom Smoke-Binary auf controller-runtime-Wiring: Scheme-Registrierung, Manager-Setup, Controller-Builder, Signal-Handler. Health-/Metrics-Probes noch nicht — kommen mit M6 (`AR-024`). |

### 3.2 Generierte Code-Dateien (committet, aber maschinell erzeugt)

| Pfad | Generator | Drift-Gate |
| ---- | --------- | ---------- |
| `api/v1alpha1/zz_generated.deepcopy.go` | `controller-gen object` | `make generated-drift-check` |
| `config/crd/k-deskflight.geo-terrain.net_opendeskpreflightchecks.yaml` | `controller-gen crd` | `make generated-drift-check` |
| `config/rbac/role.yaml` | `controller-gen rbac` | `make generated-drift-check` |

### 3.3 Neue/aktualisierte Manifeste

| Pfad | Zweck |
| ---- | ----- |
| `deploy/manifests/namespace.yaml` | `k-deskflight-system` Namespace ([`AR-016`](../../../../spec/architecture.md)). |
| `deploy/manifests/serviceaccount.yaml` | `k-deskflight-operator` SA ([`AR-017`](../../../../spec/architecture.md)). |
| `deploy/manifests/clusterrole.yaml` | wird aus `config/rbac/role.yaml` per Kustomize-Overlay referenziert oder kopiert. |
| `deploy/manifests/clusterrolebinding.yaml` | `ClusterRoleBinding k-deskflight-operator` → ClusterRole `k-deskflight-operator-cluster` ([`AR-017`](../../../../spec/architecture.md)). |
| `deploy/manifests/deployment.yaml` | Operator-Deployment (1 Replica, distroless-Image, SA-Referenz). |
| `deploy/manifests/kustomization.yaml` | Bündelt die Manifeste; CRD aus `../../config/crd/` einbinden. |
| `config/samples/k-deskflight_v1alpha1_opendeskpreflightcheck.yaml` | Beispiel-CR mit Namen, ohne Inhalt. |

### 3.4 Tooling-/Konfig-Updates

| Pfad | Änderung |
| ---- | -------- |
| `go.mod` | + `sigs.k8s.io/controller-runtime v0.24.1` (mit Transitiven). `go mod tidy` schreibt `go.sum`. |
| `Dockerfile` | + `tools`-Stage. |
| `Makefile` | + `tools`, `manifests`, `generated-drift-check`; `gates` um `generated-drift-check` erweitert. Coverage-Range nutzt jetzt `./internal/...` real (nicht mehr Bootstrap-Modus, weil internal/ Code hat). |
| `.golangci.yml` | `depguard.rules`: Stub durch die fünf Layer-Regelblöcke aus [`AR-005`](../../../../spec/architecture.md) ersetzt. |

---

## 4. Reihenfolge der Umsetzung

Jeder Schritt = ein eigener Commit; Phase A (dieser Plan) zuerst.

1. **Dependencies & API-Types-Bootstrap:**
   `go.mod` um controller-runtime erweitern, `hack/boilerplate.go.txt`
   anlegen, `api/v1alpha1/groupversion_info.go` + `opendeskpreflightcheck_types.go`
   hand-schreiben. **Kein `controller-gen`-Lauf in diesem Commit** —
   der kommt im nächsten Schritt.
2. **Tools-Stage + Manifests-Targets:**
   Dockerfile `FROM deps AS tools`, Makefile-Targets `tools`,
   `manifests`, `generated-drift-check`. Erster Lauf von
   `make manifests` produziert `zz_generated.deepcopy.go`,
   `config/crd/…yaml`, `config/rbac/role.yaml` — diese werden
   mit-committet (Generated-Drift-Gate vergleicht zukünftig genau
   gegen diesen Stand).
3. **Reconciler-Skelett + main.go-Rewrite:**
   `internal/hexagon/application/reconciler.go` (minimaler
   Pending→Running→Passed-Pfad mit RBAC-Markern am Receiver).
   `cmd/operator/main.go` vom Smoke-Binary auf controller-runtime-
   Wiring umgestellt (Manager, Scheme, Controller-Builder, Signal-
   Handling). **Nach diesem Commit** sollte `controller-gen rbac`
   einen anderen `role.yaml`-Inhalt liefern (Marker-Pickup);
   `make manifests` neu laufen lassen und das aktualisierte
   `config/rbac/role.yaml` in denselben Commit ziehen.
4. **deploy/manifests + config/samples:**
   Namespace, ServiceAccount, ClusterRoleBinding, Deployment,
   Kustomization, Beispiel-CR. Kein Generator-Output — alles
   hand-geschrieben.
5. **depguard scharf schalten + gates-Erweiterung:**
   `.golangci.yml` mit den fünf AR-005-Regelblöcken füllen, lokal
   verifizieren, dass keine Verletzung im Skelett-Code besteht.
   `Makefile gates`-Target um `generated-drift-check` erweitern;
   `make gates` lokal grün ziehen. Coverage-Range-Selector justieren
   (Bootstrap-Modus-Marker fällt weg, weil `./internal/...` jetzt Code hat).
6. **Slice-Closure:**
   Slice nach `done/` ziehen, Roadmap-Status M2 = Done, Closure-Notiz
   schreiben.

---

## 5. Lastenheft-Kennungen

`LH-F-001` (CRD bereitstellen), `LH-F-002` (Anlegen einer CR),
`LH-F-003` (Verarbeitung durch Controller), `LH-F-004` (Status-
Aktualisierung), `LH-F-005` (Conditions), `LH-F-006` (Gesamtphase),
`LH-F-007` (Summary), `LH-F-009` (API-Erreichbarkeit — implizit
über controller-runtime Manager-Start), `LH-F-035` (lesender Betrieb),
`LH-NF-002` (K8s-Konventionen), `LH-NF-004` (Stabilität),
`LH-NF-006` (Minimalrechte-Konzept), `LH-PROD-002` (API-Gruppe),
`LH-PROF-002`/`LH-PROF-003` (Profile-Schema-Defaults), `LH-AK-015`
(RBAC dokumentiert), `LH-DAT-002` (Status-Speicherung),
`LH-QG-003` (Coverage), `LH-QG-004` (Boundary scharf),
`LH-QG-005` (Generated-Drift scharf).

---

## 6. Architekturartefakte

Erfüllt aus `spec/architecture.md`:

- `AR-005` (depguard-Regeln) — scharf geschaltet.
- `AR-006` (CRD Type-Layout) — `Spec.Profile`, `Spec.Checks`,
  `Status.Phase/ObservedGeneration/Summary/Conditions`-Struktur
  angelegt. Konkrete Feld-Typen und Constraint-Marker entstehen mit
  M2.
- `AR-007` (Generated-Drift-Mechanik) — vollständig aktiv.
- `AR-008` (`v1alpha1` initial).
- `AR-009` Phase 1+3 (Fetch + Status-Update) — Skelett vorhanden,
  Run-Context/Timeout/Validate/Aggregate-Phasen kommen ab M3.
- `AR-015` (ClusterRole MVP-Minimum) — RBAC-Marker am Reconcile-
  Receiver setzen die Pflicht-Verben, `controller-gen rbac`
  produziert `role.yaml`.
- `AR-017` (ServiceAccount) — `k-deskflight-operator` im Namespace
  `k-deskflight-system`.

Vorbereitet, aktiv ab späterer Slice:

- `AR-009` Phase 2/4/5/6 — ab M3 (erster Check).
- `AR-016` (Namespace-Reconcile-Scope-Modus) — Default
  Cluster-Wide-Modus reicht für M2.
- `AR-010` ff. (Wiederholintervall, Error-Handling) — operativ wirksam
  ab M3.

---

## 7. Verifikation (Abnahmekriterien)

1. **`make tools`** baut den Tools-Stage; `$(IMAGE):go-tools` existiert.
2. **`make manifests`** läuft, schreibt `zz_generated.deepcopy.go`,
   `config/crd/…yaml`, `config/rbac/role.yaml` mit den Permissions
   des Aufrufers (kein root-Owned-File).
3. **`make generated-drift-check`** läuft nach `make manifests`
   grün; ein bewusster Edit (z. B. Marker entfernen) erzeugt
   `git diff --exit-code`-Failure.
4. **`make lint`** läuft grün mit allen fünf AR-005-Regelblöcken
   in `.golangci.yml`.
5. **`make build`** läuft grün; Image enthält den controller-runtime-
   Manager.
6. **`make test`** läuft grün; Smoke-Test für den Reconciler verifiziert
   Phasen-Transition Pending→Running→Passed (gegen `fake.NewClientBuilder`
   aus controller-runtime).
7. **`make coverage-gate`** läuft grün mit nicht-trivialer
   Coverage über `./internal/hexagon/application/`. Threshold bleibt
   in M2 niedrig (`COVERAGE_BOOTSTRAP=0` aktiv, M6 hebt auf 90 %).
8. **`make gates`** ruft (1)–(7) plus `generated-drift-check`
   gebündelt und ist grün.
9. **CRD installierbar:** `kubectl apply -f config/crd/…yaml` legt
   die CRD an (`LH-AK-001`).
10. **Operator startbar:** Deployment-Manifest mit dem in M2 gebauten
    Image rollt aus, Pod erreicht `Ready` (`LH-AK-002`).
11. **CR verarbeitbar:** `kubectl apply -f config/samples/…yaml`,
    `kubectl get opendeskpreflightcheck -o yaml` zeigt `Status.Phase:
    Passed`, leere Conditions, `ObservedGeneration` gleich
    `metadata.generation` (`LH-AK-003`, `LH-AK-004`, `LH-AK-011`).

Items 9–11 sind **observational** für den lokalen Cluster-Lauf —
sie erfordern eine kind-/minikube-Instanz und sind nicht Teil von
`make gates`. Sie werden in §10.5 attestiert (analog M1).

---

## 8. Out-of-Scope (geht in M3 oder später)

- **Echte Check-Implementierungen** — M3 (KubernetesVersion), M4
  (StorageClass/IngressClass/cert-manager/Resources), M5 (RBAC-
  Selbstprüfung).
- **AR-009 Run-Context, Timeout-Gate, Cross-Field-Validate,
  Aggregator** — kommen mit M3.
- **`AR-010` Wiederholintervall** — operativ ab M3 (sonst kein
  Re-Reconcile-Bedarf).
- **`AR-011` Error-Handling/Fehlertoleranz** — relevant ab M5.
- **`AR-024` Health-/Metrics-Probes** — ab M6.
- **`AR-026` Leader-Election** — ab M5 (Robustheit).
- **`AR-016` Namespace-Reconcile-Scope-Modus** — operativ optional;
  Default Cluster-Wide reicht.
- **Helm Chart** — nicht im MVP (`ADR 0005`).

---

## 9. Risiken und Mitigation

- **controller-runtime v0.24.1 API-Brüche gegenüber älteren m-trace-
  Referenzen:** m-trace ist kein Operator-Projekt; es gibt keine
  Vergleichsquelle. Mitigation: Reconciler-Code möglichst klein und
  idiomatisch halten, an der offiziellen kubebuilder-Doku entlang.
- **`controller-gen` Marker-Drift:** wenn jemand einen Marker-
  Kommentar editiert ohne `make manifests` zu laufen, weicht der
  generierte Output vom git-Stand ab — `generated-drift-check` fängt
  das. Risiko ist mit der Mechanik beherrscht.
- **`depguard`-Regeln zu streng beim ersten controller-runtime-Import:**
  AR-005 erlaubt `cmd/` ohne `depguard`-Restriktion (Wiring-Schicht).
  Internal Layer-Boundaries werden über die fünf Regelblöcke geprüft;
  der Reconciler liegt korrekt in `internal/hexagon/application/`
  und importiert nur `port` und `domain`. Sollte ein Import drüber
  hinausschießen, hilft die Regel-Fehlermeldung beim Debug.
- **Bind-Mount-UID-Mismatch in `make manifests`:** mit
  `--user "$(id -u):$(id -g)"` sind generierte Files nicht-root.
  Lokale macOS-Hosts können hier abweichen — falls jemand auf macOS
  arbeitet und Probleme sieht, dokumentieren wir den Override in
  CONTRIBUTING.md.

---

## 10. Closure (2026-05-17)

### 10.1 Geliefertes Datei-Set

Alle Einträge aus §3 sind committet. Die §4-Reihenfolge wurde gegenüber
dem Plan-Stand verdichtet, weil Step 1 (API-Typen) und Step 2 (Reconciler/
tools-Stage/manifests) wechselseitig hart abhängig sind — die DeepCopy-
Methode fehlt ohne Step 2, der RBAC-Generator hat ohne Step 2 keine
Marker zu scannen. Drei Code-Commits + eine Test-Ergänzung + Closure:

| Commit | Inhalt |
| ------ | ------ |
| `d61492e docs(plan): activate slice M2 …` | Phase A — Slice-Plan, Roadmap-Status, in-progress-Readme. |
| `00ccb94 feat(crd): CRD types + reconciler skeleton + controller-gen pipeline (M2 §4 Steps 1-2)` | Combined Step 1+2: go.mod (+ controller-runtime v0.24.1, k8s.io/apimachinery v0.36.0), hack/boilerplate.go.txt, api/v1alpha1/groupversion_info.go + opendeskpreflightcheck_types.go + zz_generated.deepcopy.go, internal/hexagon/application/reconciler.go, cmd/operator/main.go-Rewrite, Dockerfile tools-Stage, Makefile tools/manifests/generated-drift-check (+ gates extension), .golangci.yml depguard scharf nach AR-005. golang.org/x/net v0.49→v0.53 wegen GO-2026-4918. |
| `6125ebb feat(deploy): operator manifests + sample CR (M2 §4 Step 3)` | deploy/manifests/{namespace,serviceaccount,clusterrolebinding,deployment,kustomization}.yaml, config/samples/k-deskflight_v1alpha1_opendeskpreflightcheck.yaml, ci.yml-Job-Display-Name an erweitertes Bundle angepasst. |
| `2bb1e13 test(application): reconciler smoke + not-found + idempotent (M2 §7 #6)` | Drei fake-client-Tests in `internal/hexagon/application/reconciler_test.go`; Coverage 83.3 % über das Paket. |

### 10.2 Verifikations-Ergebnis (§7)

| # | Item | Ergebnis |
| - | ---- | -------- |
| 1 | `make tools` | ✓ Image `k-deskflight:go-tools` mit controller-gen v0.21.0 |
| 2 | `make manifests` | ✓ Erzeugt `zz_generated.deepcopy.go`, `config/crd/k-deskflight.geo-terrain.net_opendeskpreflightchecks.yaml`, `config/rbac/role.yaml` mit Caller-User (kein root-owned File) |
| 3 | `make generated-drift-check` | ✓ Grün nach `manifests` (idempotent); bewusste Marker-Edits brechen reproduzierbar mit klarer Diff-Ausgabe |
| 4 | `make lint` | ✓ `0 issues` mit den fünf AR-005-Regelblöcken aktiv |
| 5 | `make build` | ✓ Distroless-Image enthält den controller-runtime-Manager |
| 6 | `make test` | ✓ Drei Tests grün (`TestReconcileSmokeTransitionToPassed`, `TestReconcileNotFound`, `TestReconcileIdempotent`) |
| 7 | `make coverage-gate` | ✓ 83.3 % über `internal/hexagon/application/` — nicht-trivial, M2-Schwelle 0 % bleibt |
| 8 | `make gates` | ✓ Bundle aus `build + lint + test + coverage-gate + doc-refs + generated-drift-check` |
| 9 | `kubectl apply -f config/crd/…yaml` (LH-AK-001) | observational cluster-only — siehe §10.5 |
| 10 | Operator-Deployment ausrollen (LH-AK-002) | observational cluster-only — siehe §10.5 |
| 11 | `kubectl apply -f config/samples/…yaml` → Phase=Passed (LH-AK-003/004/011) | ✓ via fake-client-Tests; Cluster-Attest zusätzlich observational |

Items 9 und 10 sind strikt cluster-pflichtig (Lastenheft §17:
„installierbar in einem Kubernetes-Cluster" / „startbar in einem
Kubernetes-Cluster"); Item 11 wird durch die M2-Reconciler-fake-
client-Tests abgedeckt (CR wird gelesen, Status mit `Phase=Passed`
und leeren Conditions wird geschrieben — `LH-AK-003`/`LH-AK-004`/
`LH-AK-011` erfüllt). Ein zusätzlicher Cluster-Lauf bleibt als
observational Attest in §10.5 stehen, ohne dass der Slice re-opened
werden muss.

### 10.3 Out-of-Scope-Übergaben an M3

- AR-009 Phase 2/4/5/6 (Cross-Field-Validate, Active-Check-Resolve,
  Sammeln, Aggregieren) — M3 mit dem ersten echten Check.
- AR-010 ff. (Wiederholintervall, Error-Handling) — operativ ab M3.
- Generated-Drift-Gate ist scharf; jedes M3+-CRD-Schema-Update braucht
  `make manifests`-Commit.
- `Spec.Checks.IngressClass`, `Spec.Checks.StorageClass`, … kommen mit
  M3/M4 als optionale Sibling-Felder neben `KubernetesVersion`.

### 10.4 Lessons learned

- **controller-gen v0.21.0 + nicht-root-Container:** `controller-gen`
  ruft intern `go list`, das wiederum versucht
  `/root/.cache/go-build` zu schreiben. Mit `--user $(id -u):$(id -g)`
  ist `/root` nicht beschreibbar — die Fehlermeldung ist intransparent
  (controller-gen druckt nur die Help-Page). Lösung: `GOCACHE` und
  `GOMODCACHE` per Env-Var auf `/tmp/...` umleiten. Pattern jetzt im
  Makefile-Header dokumentiert.
- **controller-gen Output-Syntax:** Die korrekte Form ist
  `output:<generator>:dir=<pfad>`, nicht `output:<generator>:artifacts:config=<pfad>`.
  Die `artifacts:config`-Form ist ein eigener Generator-Modus, kein
  per-Generator-Output-Selector. Bei Fehl-Syntax druckt controller-gen
  schweigend die Help-Page (kein eindeutiger Parse-Error).
- **golangci-lint v2 `depguard.rules: {}`:** Leeres Rules-Map heißt
  in v2 **nicht** „keine Regeln", sondern „Default-Deny-Liste `Main`
  greift". Slice §2.3 war zu optimistisch formuliert — die Konsequenz
  war ein lint-roter Zwischenstand mit 17 verbotenen Imports. Lesson:
  depguard entweder vollständig disablen oder mit ≥ 1 lax-Regel
  aktivieren; halbe Aktivierung gibt es nicht.
- **`scheme.Builder` aus controller-runtime ist deprecated** (SA1019):
  api/-Pakete sollen schlank an transitiven Imports bleiben. Das
  `runtime.NewSchemeBuilder(addKnownTypes)`-Pattern aus apimachinery
  ist der vorgeschriebene Weg — schöner Nebeneffekt: das `init()` in
  `*_types.go` entfällt, `gochecknoinits`-Carveout für api/v1alpha1/
  wird damit unnötig.
- **`metav1.Time` ist sekundengenau:** RFC3339 ohne Nanoseconds. Tests,
  die `Time`-Werte vergleichen, müssen mit `time.Date(...0, time.UTC)`
  arbeiten (sekundengenaue Literale), sonst trunkiert der fake-client
  und der Vergleich wird unstable.
- **`res.Requeue` ist SA1019-deprecated in controller-runtime v0.24**:
  `RequeueAfter` ist der Nachfolger; `RequeueAfter=0` deckt beide
  semantischen Fälle (no requeue / requeue-after-0) ab.

### 10.5 Folge-Attest

| Item | Datum | Notiz |
| ---- | ----- | ----- |
| §7 #9 — `LH-AK-001` CRD installierbar (kubectl apply der CRD) | 2026-05-17 | Attestiert durch ersten CI-Lauf des `cluster-smoke`-Workflows (ADR 0013): kind 0.31.0 + kindest/node:v1.34.0; `kubectl apply -f config/crd/…yaml` + `kubectl wait --for=condition=Established crd/…` grün. Run-URL: <https://github.com/pt9912/k-deskflight/actions/runs/25999750149>. |
| §7 #10 — `LH-AK-002` Operator startbar via Deployment-Manifest | 2026-05-17 | Attestiert durch denselben `cluster-smoke`-Lauf: `kubectl apply -k …` mit Image-Override auf das lokal gebaute `k-deskflight:go`; `kubectl wait deployment/k-deskflight-operator --for=condition=Available` grün; zusätzlich HTTP-Smoke gegen `/healthz`, `/readyz` und `/metrics` (Step 9 des cluster-smoke.sh, scripts/operator-http-smoke.sh) liefert OK / Prometheus-Exposition-Format. |
| §7 #11 — `LH-AK-003`/`LH-AK-004`/`LH-AK-011` CR verarbeitbar + Status + Conditions | 2026-05-17 | Via M2/M3-Reconciler-fake-client-Tests verifiziert: Reconciler liest CR, schreibt `Status.Phase`, `ObservedGeneration` und (in M3+) `Conditions` deterministisch. Ein realer Cluster-Lauf (`kubectl apply -f config/samples/…yaml` + `kubectl get opendeskpreflightcheck` Beobachtung) bleibt als zusätzliches observational Attest sinnvoll, ist aber für die Lastenheft-Anforderung nicht zwingend („wird erkannt" / „schreibt Status" / „Ergebnisse als Conditions" sind alle code-pfad-Aussagen). |
| §7 #8 / CI — gates-Bundle inkl. drift-check grün auf GitHub-Actions | 2026-05-17 | Run beim Push von SHA `00ccb94` (M2 Step 1+2) bereits grün: `gates` 379 s, `security-gates` 41 s (Run `25991308791`). Folgende Pushes (`6125ebb`, `2bb1e13`) ziehen kleinere Diffs. |
