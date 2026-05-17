# Slice M3 — Erste Prüfung: Kubernetes-Version

**Status:** Done
**Eröffnet:** 2026-05-17
**Geschlossen:** 2026-05-17
**Vorgänger:** [M2 — CRD + Controller-Skeleton (Done)](slice-M2-crd-controller-skeleton.md)
**Nachfolger:** [M4 — Cluster-State-Prüfungen](../in-progress/roadmap.md#m4--cluster-state-pr%C3%BCfungen)
**Bezug:**
[Roadmap §3 M3](../in-progress/roadmap.md#m3--erste-pr%C3%BCfung--kubernetes-version),
[`spec/architecture.md` §5 (AR-009), §6 (AR-012, AR-013, AR-014), §7 (AR-018)](../../../../spec/architecture.md),
[ADR 0009](../../adr/0009-k8s-versions-support-und-profile-mindestversionen.md)

---

## 1. Lieferziel

Erste fachliche Prüfung scharf: `KubernetesVersionCheck` vergleicht
die per `discovery.ServerVersion()` ermittelte Server-Version mit dem
in `spec.checks.kubernetesVersion.min` konfigurierten Mindeststand
(Default `"1.34"` per [`ADR 0009 §2.2`](../../adr/0009-k8s-versions-support-und-profile-mindestversionen.md)).
Reconciler durchläuft jetzt einen minimal-sequenziellen AR-009-Pfad,
schreibt eine Aggregator-Condition `KubernetesVersionReady` und füllt
`status.summary` mit echten Zählern.

**Was M3 noch nicht macht:** Worker-Pool, panic-Recovery, Late-Result-
Drops, Timeout-Cross-Constraints, Leader-Election — kommt mit M5
(Robustheit). Mehrere Checks parallel — M4.

---

## 2. Slice-Entscheidungen

### 2.1 Reconciler-Tiefe — minimal-sequenziell

AR-009-Phasen 1, 3, 4 (sequenziell, 1 Check), 5, 6 sind scharf;
Phase 2 (Cross-Field-Validate) bleibt auf den Basis-Check
(`KubernetesVersionSpec.Validate`) beschränkt — die volle
Cross-Constraint-Härtung (`CHECK_TIMEOUT_SECONDS` vs.
`RECONCILE_TIMEOUT_SECONDS`, `OPERATOR_STRICT_CONFIG`) kommt in M5.
Run-Context-Timeout in M3 simpel via
`context.WithTimeout(ctx, defaultReconcileTimeout)` mit hartem
Default `120 s` ohne Cross-Field-Korrektur. Begründung: ein einziger
Check rechtfertigt noch keine Worker-Pool-Mechanik, und ein
maskierter Cross-Constraint-Pfad wäre untestbar ohne Konkurrenz.

### 2.2 CheckRegistry — ja, mit 1 Eintrag

[`AR-013`](../../../../spec/architecture.md) wird sofort implementiert:
Port-Interface in `internal/hexagon/port/checkregistry.go`,
Map-Implementierung in `internal/adapter/check/registry.go`,
Wiring in `cmd/operator/main.go`. Damit erweitert M4 nur die
`Register`-Aufrufe, nicht die Reconciler-Logik.

### 2.3 Semver-Vergleich — Masterminds

`github.com/Masterminds/semver/v3` (bereits transitiv via
controller-tools im `go.sum`) wird auf Direct-Require promotet.
Behandelt `v`-Prefix, optionale Patch-Version (`1.34` → `1.34.0`),
Pre-Releases (`1.34.0-rc.1`). `MustParse`-Varianten werden
vermieden — der Check returniert bei Parse-Fehler eindeutige
`Unknown`/`InvalidSpec`-Results statt zu panicken (siehe
[`AR-012`](../../../../spec/architecture.md)).

### 2.4 Aggregator-Heimat

`internal/hexagon/application/aggregator.go` implementiert
[`AR-014`](../../../../spec/architecture.md): Severity-Mapping zu
Phase, deterministische Conditions-Sortierung, Dedupe per Type.
Aggregator nimmt `[]domain.Result` entgegen und liefert
`api/v1alpha1`-Typen (`Phase`, `[]Condition`, `Summary`) — der
Reconciler nutzt das Ergebnis direkt für `Status().Update`.
Application-Layer darf das per AR-005 (keine Regel verbietet api-
Import aus application/).

### 2.5 Versionspins

| Komponente | Pin in M3 | Quelle |
| ---------- | --------- | ------ |
| `github.com/Masterminds/semver/v3` | latest stable (v3.4.0 transitiv via controller-tools) | Direct-Require, kein neuer Tag-Lookup |
| `k8s.io/client-go` (Discovery-Adapter) | identisch zur transitiven Auflösung aus controller-runtime v0.24.1 | unverändert |

---

## 3. Datei-Inventar

### 3.1 Neue Code-Dateien

| Pfad | Zweck |
| ---- | ----- |
| `internal/hexagon/domain/check.go` | AR-012 Check + Result + Severity + ConditionStatus + CheckSpec-Interface. Pure Domäne, keine k8s-Imports (depguard `domain-isolation` enforced). |
| `internal/hexagon/domain/kubernetesversion.go` | `KubernetesVersionSpec` implementiert CheckSpec; `Min string`-Feld + `Validate(ctx)` (semver-Parse-Check). |
| `internal/hexagon/port/kubernetes.go` | `KubernetesAPI`-Interface mit `ServerVersion(ctx) (string, error)`. Application konsumiert; Adapter implementiert. |
| `internal/hexagon/port/checkregistry.go` | AR-013 `CheckRegistry`-Interface + `CheckSelectionIssue`. |
| `internal/adapter/k8s/discovery.go` | Implementiert `port.KubernetesAPI` via `k8s.io/client-go/discovery.NewDiscoveryClientForConfig`. Konstruktor nimmt `*rest.Config`. |
| `internal/adapter/check/kubernetesversion.go` | Check-Implementierung: nimmt `port.KubernetesAPI`, prüft `spec.Kind() == "kubernetesVersion"`, ruft `ServerVersion`, vergleicht mit `spec.Min` via Masterminds, liefert `domain.Result` mit `KubernetesVersionReady`-Condition. |
| `internal/adapter/check/registry.go` | Map-basierte `CheckRegistry`-Impl. `ListByProfile` ist in M3 trivial (1 Check, alle Profile aktiv) — M4 differenziert. |
| `internal/hexagon/application/aggregator.go` | AR-014 Severity-Aggregation + Conditions-Dedupe/Sort. Nimmt `[]domain.Result`, liefert `api/v1alpha1.Phase`, `Summary`, `[]Condition`. |

### 3.2 Erweiterte Code-Dateien

| Pfad | Änderung |
| ---- | -------- |
| `internal/hexagon/application/reconciler.go` | Phasen 1+3+4+5+6 scharf: Get → translate `Spec.Checks.KubernetesVersion` zu `domain.KubernetesVersionSpec` → `registry.ListByProfile` → run check sequenziell mit `runCtx` → aggregator → Status-Update. Idempotency-Klausel bleibt. RBAC-Marker unverändert (AR-015 ist bereits in M2 angelegt). |
| `cmd/operator/main.go` | Wiring: discovery-Adapter aus `mgr.GetConfig()`, KubernetesVersionCheck-Konstruktion, Registry, Inject in Reconciler. |
| `go.mod` | `github.com/Masterminds/semver/v3` als direct-Require (vorher indirect). |

### 3.3 Neue Test-Dateien

| Pfad | Coverage |
| ---- | -------- |
| `internal/hexagon/domain/check_test.go` | Smoke-Coverage über Result/Severity/Status-Konstanten (kein Verhalten — nur Konsistenz). |
| `internal/hexagon/domain/kubernetesversion_test.go` | `Validate()` mit gültigen/ungültigen Semvern; `Kind()` Stable-String. |
| `internal/adapter/check/kubernetesversion_test.go` | Fake-`KubernetesAPI`-Implementierung; Tabelle: passed (1.34 vs. 1.34.2), failed (99.99 vs. 1.34.2), InvalidSpec, ServerError, Min-Parse-Fehler. |
| `internal/adapter/check/registry_test.go` | Register + Resolve + ListByProfile mit aktivem/inaktivem Check. |
| `internal/hexagon/application/aggregator_test.go` | Tabelle: alle-Passed → Passed; ein critical-Failed → Failed; ein warning-Failed ohne critical → Warning; nur Unknown → Unknown; gemischte Severity-Dedupe; Sort-Stabilität. |
| `internal/hexagon/application/reconciler_test.go` (Erweiterung) | Existierende Smoke/NotFound/Idempotent bleiben. Neuer Test: passed-Case (fake API liefert 1.34.2, min=1.34, erwartet Phase=Passed, Condition=KubernetesVersionReady=True). Neuer Test: failed-Case (synthetisch min=99.99, erwartet Phase=Failed, Severity=critical, Reason=KubernetesVersionTooOld). |

### 3.4 Neue Beispiel-Manifeste

| Pfad | Inhalt |
| ---- | ------ |
| `config/samples/k-deskflight_v1alpha1_opendeskpreflightcheck.yaml` | Erweitert um `spec.checks.kubernetesVersion.min: "1.34"` als explizites Beispiel (Default-Pfad bleibt zusätzlich verfügbar mit `spec: {}`). |

### 3.5 Generated Drift Refresh

`make manifests` muss neu laufen, weil:

- Neue depguard-Carveouts oder Linter-Adjustments können nötig werden,
  falls Adapter-Imports neue Pfade öffnen.
- CRD-Schema bleibt unverändert (Reconciler-Internals ändern, nicht
  die externen Typen) — `zz_generated.deepcopy.go`,
  `config/crd/…yaml` und `config/rbac/role.yaml` sollten idempotent
  bleiben. Drift-Gate verifiziert das.

---

## 4. Reihenfolge der Umsetzung

Jeder Schritt = ein eigener Commit; lokal über `make gates` grün
ziehen bevor gepusht wird.

1. **Domain + Port-Interfaces:** `check.go`, `kubernetesversion.go`,
   `port/kubernetes.go`, `port/checkregistry.go`. Reine Interface-/
   Typen-Definitionen, keine Logik. Tests für Domain.
2. **Adapter:** `adapter/check/kubernetesversion.go`, `adapter/check/registry.go`,
   `adapter/k8s/discovery.go`. Tests für check/* mit Fake-Port; Discovery-
   Adapter bleibt in M3 untestet (envtest-Pflicht für realen API-Call
   ist M6).
3. **Aggregator:** `application/aggregator.go` + Tests. Pure Funktion
   `Aggregate([]domain.Result) → (Phase, Summary, []Condition)`.
4. **Reconciler-Rewrite + Wiring:** `application/reconciler.go` zieht
   Registry + KubernetesAPI per Constructor-Injection. `cmd/operator/
   main.go` wiret discovery-Adapter und KubernetesVersionCheck.
5. **CR-Beispiel + reconciler_test-Erweiterung:** zwei neue Reconciler-
   Tests (passed/failed) gegen Fake-Registry + Fake-API.
6. **Slice-Closure:** nach `done/` ziehen, Roadmap-Status M3 = Done,
   Closure-Notiz schreiben.

---

## 5. Lastenheft-Kennungen

`LH-F-008` (Kubernetes-Version prüfen), `LH-F-009` (API-Erreichbarkeit
— implizit via discovery), `LH-F-031` (Schweregrad), `LH-F-032`
(Ergebnis-Inhalt — Result-Struktur), `LH-NF-003` (Nachvollziehbarkeit
— strukturiertes slog-Logging), `LH-DAT-003` (Zeitstempel — Result
LastTransition), `LH-QA-001` (verständliche Fehlermeldungen —
Message-Inhalt klar), `LH-PROF-002`/`LH-PROF-003` (Profile-Defaults
aktiv).

---

## 6. Architekturartefakte

Erfüllt aus `spec/architecture.md`:

- `AR-009` Phase 1+3+4 (minimal-sequenziell)+5+6 — scharf.
- `AR-012` (Check-Interface) — Domain-Definition.
- `AR-013` (Check-Registry) — Port + Map-Adapter.
- `AR-014` (Schweregrad-Aggregation + Dedupe/Sort) — application/aggregator.

Vorbereitet, aktiv ab späterer Slice:

- `AR-009` Phase 2 voll (Cross-Constraint Timeout) — M5.
- `AR-009` Phase 4 Worker-Pool + panic-Boundary — M5.
- `AR-010` (Wiederholintervall) — M5.
- `AR-011` (Error-Handling-Härte, leader-election) — M5.

---

## 7. Verifikation (Abnahmekriterien)

1. **`make build`** baut neuen Reconciler-Stand und Discovery-Adapter.
2. **`make lint`** grün — depguard-Regeln aus AR-005 prüfen die neuen
   Adapter/Port-Files; Application importiert nur `port` und
   `domain`, kein `adapter`.
3. **`make test`** grün; Reconciler-Tests inkl. der zwei neuen Fälle
   (passed/failed) bestehen.
4. **`make coverage-gate`** grün; Coverage über
   `internal/hexagon/{domain,application,adapter/check}` sollte deutlich
   über 0 % liegen (Ziel ~70 %+, M6 hebt auf 90 %). `adapter/k8s/discovery.go`
   bleibt M3-untested und drückt den Schnitt; das ist akzeptiert.
5. **`make doc-refs`** grün.
6. **`make generated-drift-check`** grün — CRD-Schema unverändert.
7. **`make gates`** grün (alle obigen gebündelt).
8. **`make security-gates`** grün; `govulncheck` ohne neue Findings.
9. **`LH-AK-005`** — synthetischer Smoketest gegen ein kind-/minikube-
   Cluster: `kubectl apply -f config/samples/…yaml`, dann
   `kubectl get opendeskpreflightcheck` zeigt `Phase: Passed` und
   eine Condition `KubernetesVersionReady=True` mit nicht-leerem
   Message. Observational, attestiert via §10.5.

---

## 8. Out-of-Scope (geht in M4–M7 oder später)

- **Weitere Checks** (StorageClass, IngressClass, cert-manager,
  Resources) — M4.
- **RBAC-Selbstprüfung** (`SelfSubjectAccessReview`) — M5.
- **Robustheit** (Panic-Recovery, Timeout-Härtung, Leader-Election) — M5.
- **Metrics-Endpoint, kind/envtest-Suite** — M6.
- **`LH-F-014` ClusterIssuer-Detailprüfung** — v0.2 (nicht MVP).
- **CRD-Schema-Erweiterung** für StorageClass/IngressClass-Sub-Specs —
  M4 (M3 lässt die Sibling-Felder in `ChecksSpec` weg).

---

## 9. Risiken und Mitigation

- **`discovery.ServerVersion()` Format-Variabilität:** Server-Versions
  können `v1.34.2`, `v1.34.2+build`, `1.34.2` etc. zurückgeben. Adapter
  normalisiert auf semver-parsbar (Strip `v`-Prefix, Build-Suffix
  abschneiden). Tests decken die drei Varianten ab.
- **Min-Config ohne Patch-Version:** `"1.34"` ist explizit erlaubt
  ([`ADR 0009 §2.2`](../../adr/0009-k8s-versions-support-und-profile-mindestversionen.md)).
  Masterminds-Constraint parsed das als `>=1.34.0`. Test deckt ab.
- **Discovery-Client braucht REST-Config:** main.go-Wiring zieht das
  aus dem controller-runtime-Manager (`mgr.GetConfig()`). Funktioniert
  out-of-cluster (kubeconfig) und in-cluster (ServiceAccount-Token)
  ohne Sonderfall.
- **`adapter/k8s/discovery.go` unbedeckt:** Coverage-Range schließt
  diesen Adapter aktuell mit ein. Wir akzeptieren M3-Untests und holen
  das in M6 via envtest nach. `make coverage-gate` läuft mit
  Threshold 0 weiter; keine Build-Blockade.
- **`Status().Update` Optimistic-Locking-Konflikt:** Reconciler liest
  die CR, schreibt Status. Bei parallelen Update-Versuchen (z. B.
  Cache-Update + Edit) kann `Update` 409 zurückgeben. M3 returniert
  den Error → controller-runtime requeued automatisch. M5 ergänzt
  explizite RetryOnConflict-Handhabung falls nötig.

---

## 10. Closure (2026-05-17)

### 10.1 Geliefertes Datei-Set

Sechs Code-Commits + Phase-A + Closure. Die §4-Reihenfolge wurde
weitgehend eingehalten — Step 4 (Reconciler-Rewrite) und Step 5
(CR-Beispiel + Tests) sind zu einem Commit verbunden, weil die
existierenden M2-Reconciler-Tests an die neue Reconciler-Struct-
Signatur angepasst werden mussten und alleine nicht lint-stabil
gewesen wären.

| Commit | Inhalt |
| ------ | ------ |
| `c93683a docs(plan): activate slice M3 …` | Phase A — Slice-Plan, Roadmap-Status, in-progress-Readme. |
| `b39a4d3 feat(domain,port): Check interface + KubernetesAPI + CheckRegistry (M3 §4 Step 1)` | domain/check.go (AR-012), domain/kubernetesversion.go (Spec+Validate), port/kubernetes.go (`ServerVersion`), port/checkregistry.go (AR-013-Interface). 12 Tabellen-Tests für `KubernetesVersionSpec.Validate`. |
| `b6e0aa2 feat(adapter): KubernetesVersion check + Registry + Discovery (M3 §4 Step 2)` | adapter/check/registry.go (Map-Impl, threadsafe), adapter/check/kubernetesversion.go (Masterminds/semver/v3, sieben Tests inkl. invalid-spec/build-suffix), adapter/k8s/discovery.go (client-go-Wrapper, M3-untestet). ireturn-Allow um Domain/Port-Pattern erweitert. |
| `c101a1f feat(application): Aggregator — severity → phase + conditions dedupe/sort (M3 §4 Step 3)` | application/aggregator.go (AR-014: Severity→Phase + Dedupe-by-Name mit höchster Severity + alphabetische Sort). Acht Tests. |
| `113951d feat(application,cmd): reconciler runs real checks + wire KubernetesVersion (M3 §4 Steps 4-5)` | Reconciler-Rewrite mit Run-Context (120s-Timeout), Phase-2-Validate, Phase-3-Registry-Resolve, Phase-4-sequenzielle-Check-Execution, Phase-5-Aggregate, Phase-6-Status-Update. SpecInvalid-Pfad für Validation- und Registry-Issues. cmd/operator/main.go wiret discoveryAdapter + KubernetesVersionCheck + Registry. Reconciler-Tests erweitert (passed/failed/spec-invalid + bestehende Tests an neue Struct angepasst). config/samples/ erweitert um `kubernetesVersion.min: "1.34"`. |

### 10.2 Verifikations-Ergebnis (§7)

| # | Item | Ergebnis |
| - | ---- | -------- |
| 1 | `make build` | ✓ Image enthält neuen Reconciler + Discovery-Adapter |
| 2 | `make lint` | ✓ `0 issues` mit allen depguard-Regeln scharf (`application-no-adapter` greift im Reconciler-_test.go-File — daher die lokale fakeRegistry/stubCheck statt Adapter-Import) |
| 3 | `make test` | ✓ 27 Tests grün (domain: 13, adapter/check: 11, application: 13 inkl. neue passed/failed/spec-invalid) |
| 4 | `make coverage-gate` | ✓ 86.1 % über alle internal/-Pakete (vorher 83.3 %); discovery.go bleibt untestet pro Plan |
| 5 | `make doc-refs` | ✓ All documentation links OK |
| 6 | `make generated-drift-check` | ✓ controller-gen-Outputs unverändert |
| 7 | `make gates` | ✓ Bundle (build+lint+test+coverage-gate+doc-refs+drift-check) |
| 8 | `make security-gates` | ✓ govulncheck ohne Findings (Masterminds/semver/v3 v3.4.0 sauber) |
| 9 | `LH-AK-005` (CR mit Phase=Passed + KubernetesVersionReady=True auf realem Cluster) | observational — siehe §10.5 |

### 10.3 Out-of-Scope-Übergaben an M4

- Weitere Check-Implementierungen (StorageClass, IngressClass,
  cert-manager, Resources) — M4 ergänzt sie nur als Sibling-Felder
  in `ChecksSpec` und ruft `registry.Register(...)` in main.go;
  die Reconciler-Mechanik bleibt unangetastet.
- `ListByProfile` profil-differenzierend — M4 (sobald mehrere Checks
  pro Profil unterschiedlich aktivieren).

### 10.4 Lessons learned

- **depguard application-no-adapter greift auch in `_test.go`:**
  `internal/hexagon/application/reconciler_test.go` darf nicht
  `internal/adapter/check` importieren. Lösung: lokale `fakeRegistry`
  + `stubCheck` direkt im Test-File. Sauberer Schnitt: Application-
  Layer-Tests verifizieren Plumbing gegen Domain-/Port-Doubles, die
  echten Check-Implementierungen sind im adapter-Package separat
  unit-getestet. Architektonisch das richtige Test-Pyramid-Layout.
- **`ireturn`-Allow muss Domain-/Port-Pattern enthalten:** sobald
  ein Adapter eine Domain-Interface-Methode wie `Resolve(name) (domain.Check, bool)`
  exportiert, schlägt ireturn ohne explizite Allow-Regex zu. M1 hatte
  das als Folge angekündigt; M3 hat es scharf geschaltet.
- **`unparam` flagt Methoden, die immer denselben Wert returnen.**
  Initial hatte `writeStatus` `(ctrl.Result, error)` als Rückgabe; der
  Result-Teil war immer leer. unparam erkannte das. Lösung: writeStatus
  liefert nur `error`, Caller wrappen `ctrl.Result{}` an der Aufrufsite.
  Saubereres API, kein Compromise.
- **client-go-`discovery.DiscoveryInterface` nimmt keinen Context:**
  unser `KubernetesAPI.ServerVersion(ctx)` hat den Context an der
  Aufrufsite, aber die client-go-Library ignoriert ihn. M3-Adapter
  fängt das transparent ab (rest.Config-Timeout greift), M5-Härtung
  kann ein Goroutine+Select-Pattern einziehen, falls explizite
  Cancellation gebraucht wird.
- **`metav1.Time` als Status-Field:** der `time.Time` im `domain.Result`
  ist nanos-genau, `metav1.NewTime` strippt das auf Sekunden bei der
  Serialisierung. Reconciler-Tests müssen mit `time.Date(...0, UTC)`
  arbeiten — die fixedClock()-Helper-Funktion ist deshalb explizit
  sekundengenau.

### 10.5 Folge-Attest

| Item | Datum | Notiz |
| ---- | ----- | ----- |
| §7 #9 — LH-AK-005 auf realem Cluster | pending | kind- oder minikube-Smoketest: `kubectl apply -k deploy/manifests/`, danach `kubectl apply -f config/samples/k-deskflight_v1alpha1_opendeskpreflightcheck.yaml`, dann `kubectl get opendeskpreflightcheck smoke -o yaml`. Erwartung bei aktueller Server-Version ≥ 1.34: `status.phase: Passed`, `status.summary.passed: 1`, `status.conditions[0].type: KubernetesVersionReady`, `status.conditions[0].status: "True"`, `status.conditions[0].severity: info`. Pattern analog M2 §10.5. |
| §7 #7 + CI — gates-Bundle inkl. controller-gen-Pipeline grün auf GitHub-Actions | 2026-05-17 | Closure-Push (`315b5dd`) bestätigt: `gates (build + lint + test + coverage-gate + doc-refs + generated-drift-check)` 407 s grün, `security-gates (govulncheck)` 41 s grün. Run-URL: <https://github.com/pt9912/k-deskflight/actions/runs/25997783188>. Damit ist §7 #7 final attestiert; verbleibende Items #9 (Cluster-Smoke) bleiben observational. |
| Review-Befunde 1–3 nach Closure | 2026-05-17 | Drei nach-Closure-Findings aus Code-/Manifest-Inspektion adressiert (siehe §10.6). |

### 10.6 Nach-Closure-Review-Fixups

Drei Befunde wurden nach dem Closure-Push gemeldet und in einem
zusätzlichen Code-Commit eingearbeitet:

1. **Default-Pfad führt keinen Check aus** (Reconciler war strikt
   pointer-abhängig). Fix: `buildSpecMap` aktiviert KubernetesVersion
   immer mit `domain.DefaultKubernetesVersionMin = "1.34"`, override
   nur, wenn `spec.checks.kubernetesVersion.min` explizit gesetzt ist.
   Neuer Test `TestReconcileDefaultActivatesKubernetesVersion`.
2. **Non-Passed-Idempotency** — Failed/Warning/Unknown wurden bei
   jedem Resync neu geschrieben. Fix: `isAlreadyReconciled` skippt
   jetzt alle terminalen Phasen bei matching ObservedGeneration
   (Pending/Running bleiben bewusst non-terminal). Neuer Test
   `TestReconcileIdempotentFailed`.
3. **Pending → Running → Final Transitions** — M3 hatte den M2-§3-
   Roadmap-Wortlaut nicht eingelöst. Fix: neue Helper-Methode
   `markPhase` schreibt Pending vor Phase-2-Validation und Running
   vor Check-Execution; finaler Aggregator-Status bleibt im
   `writeStatus`-Pfad. Beobachter sehen die Transitions via
   `kubectl get -w`. ObservedGeneration wird erst beim finalen
   `writeStatus` gehoben, damit ein abgebrochener Pending/Running-
   Reconcile sauber neu läuft.

Domain-Konstante `DefaultKubernetesVersionMin = "1.34"` führt das
ADR-0009-§2.2-Default zentral; CRD-Schema-Default
`+kubebuilder:default="1.34"` und Reconciler-Default-Activation
sind damit synchron gehalten.

Gates + Security weiterhin grün nach den Fixes.
