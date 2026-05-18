# Slice M4 — Cluster-State-Prüfungen (Storage, Ingress, cert-manager, Ressourcen)

**Status:** In Progress
**Eröffnet:** 2026-05-18
**Vorgänger:** [M3 — Erste Prüfung: Kubernetes-Version (Done)](../done/slice-M3-kubernetes-version-check.md)
**Nachfolger:** [M5 — RBAC-Selbstprüfung & Robustheit](roadmap.md#m5--rbac-selbstpr%C3%BCfung--robustheit)
**Bezug:**
[Roadmap §3 M4](roadmap.md#m4--cluster-state-pr%C3%BCfungen),
[`spec/architecture.md` §5 (AR-009), §6 (AR-012, AR-013, AR-014), §7 (AR-015)](../../../../spec/architecture.md),
[ADR 0009 §2.2](../../adr/0009-k8s-versions-support-und-profile-mindestversionen.md),
[ADR 0010](../../adr/0010-externe-dienstpruefungen-und-secret-mechanik.md),
[ADR 0013](../../adr/0013-cluster-smoke-platform.md)

---

## 1. Lieferziel

Vier weitere konkrete Check-Implementierungen, die das MVP-Pflichtset
nach `LH-PRI-001` schließen:

1. **StorageClass** — konfigurierte `StorageClass`-Namen sind im Cluster
   vorhanden; optional Default-StorageClass-Erkennung
   (`LH-F-010`/`LH-F-011`).
2. **IngressClass** — konfigurierte `IngressClass`-Namen sind vorhanden
   (`LH-F-012`).
3. **cert-manager** — die `cert-manager.io`-API-Gruppe ist im Cluster
   bekannt (`LH-F-013`, nur Vorhandensein; `LH-F-014` ClusterIssuer-
   Detail bleibt v0.2).
4. **ClusterResources** — Summe der `Allocatable` CPU/Memory aller
   `Ready`-Nodes erreicht eine konfigurierte Mindestmenge
   (`LH-F-015`/`LH-AK-009`).

Reconciler bleibt im sequenziellen AR-009-Pfad (Phasen 1 + 3 + 4
sequenziell + 5 + 6). Aggregator (AR-014) und Registry (AR-013) sind
bereits vollständig und werden nur um neue Einträge erweitert,
nicht restrukturiert.

**Was M4 noch nicht macht:**

- Worker-Pool, Panic-Boundary, Cross-Constraint-Timeout-Härtung,
  Leader-Election — bleibt M5.
- `SelfSubjectAccessReview`-RBAC-Selbstprüfung — M5.
- ClusterIssuer-Detailprüfung (`LH-F-014`) — v0.2.
- Helm-Chart-Verpackung — v0.2 (`ADR 0005`).

---

## 2. Slice-Entscheidungen

### 2.1 Port-Segregation pro Discovery-Domain

M3 hat `port.KubernetesAPI` als einzelnes Interface mit `ServerVersion`
angelegt. Mit vier neuen Discovery-Calls würde ein einzelnes
Interface auf fünf Methoden anwachsen — Tests müssten dann auch dann
alle Methoden stubben, wenn ein Check nur eine braucht.

**Entscheidung:** vier neue, eng geschnittene Port-Interfaces, jeweils
ein Methodenpaar. `port.KubernetesAPI` bleibt für `ServerVersion`
zuständig und wird **nicht** erweitert. Test-Doubles bleiben dadurch
minimal (Interface-Segregation, Go-idiomatic).

| Port-Interface | Methoden |
| -------------- | -------- |
| `port.StorageClassDiscovery` | `ListStorageClasses(ctx) ([]StorageClassInfo, error)` |
| `port.IngressClassDiscovery` | `ListIngressClasses(ctx) ([]IngressClassInfo, error)` |
| `port.APIGroupDiscovery` | `HasAPIGroup(ctx, name string) (bool, error)` |
| `port.NodeDiscovery` | `ListReadyNodes(ctx) ([]NodeInfo, error)` |

Die `Info`-Structs sind reine Datentypen ohne k8s-Imports — der
Adapter mappt `*storagev1.StorageClass`, `*networkingv1.IngressClass`,
`*metav1.APIGroupList` und `*corev1.Node` in diese Domain-/Port-Typen
um. Die Port-Definition lebt unter `internal/hexagon/port/` und ist
damit unter der `port-no-application`/`port-no-adapter`-depguard-Regel
geschützt.

### 2.2 Default-Aktivierung pro Check

`KubernetesVersion` wird in M3 immer aktiviert (CR ohne explizites
`kubernetesVersion` läuft mit `DefaultKubernetesVersionMin`). Für die
vier M4-Checks gilt:

| Check | Default-Aktivierung | Begründung |
| ----- | ------------------- | ---------- |
| StorageClass | nein | ohne `Names` keine Prüfgrundlage; `RequireDefault`-Only-Run wäre missverständlich |
| IngressClass | nein | ohne `Names` keine Prüfgrundlage |
| cert-manager | **ja** | reine Existence-Prüfung, keine Spec-Parameter — `MVP-Floor`-Verhalten möglich |
| ClusterResources | **ja, mit Profile-Default** | profile-abhängige Code-Defaults (siehe §2.3) machen einen sinnvollen Run möglich auch ohne Spec |

Sobald der Anwender ein Sub-Spec-Objekt setzt (`spec.checks.X = {…}`),
gewinnen die explizit gesetzten Werte; Profile-Defaults füllen nur
leere Felder.

### 2.3 Profile-Defaults für ClusterResources

`LH-PROF-002` (evaluation) und `LH-PROF-003` (production) verlangen
profil-spezifische Mindestanforderungen. Für die anderen drei Checks
(Storage, Ingress, cert-manager) gibt es keinen sinnvollen
Profil-Unterschied — Existenz ist binär.

**ClusterResources-Code-Defaults** (im Reconciler, nicht im
CRD-Schema, damit Profile-Default nicht statisch ins OpenAPI eingeht):

| Profile | `MinCPU` | `MinMemory` |
| ------- | -------- | ----------- |
| `production` | `"4"` (4 vCPU) | `"8Gi"` |
| `evaluation` | `"2"` (2 vCPU) | `"4Gi"` |

Werte sind konservativ gegen die OpenDesk-Doku gewählt; konkrete
Schwellen sind nicht normativ und bleiben in der Anwender-Doku (M6)
diskutierbar. Domain-Konstanten `DefaultClusterResourcesMinCPU…` /
`…Production` etc. halten die Werte zentral, parallel zu
`DefaultKubernetesVersionMin` aus M3.

### 2.4 Registry-Profil-Filter — vorerst weiter allow-all

`Registry.ListByProfile` filtert in M3 nur über `UnknownCheck`-Spec-
Einträge; profil-spezifisches Aktivierungs-Mapping
(`CheckNotAllowedInProfile`) ist im Code als TODO markiert.

**Entscheidung:** M4 behält den allow-all-Pfad bei. Profile beeinflussen
nur die Default-Werte (§2.3), nicht die Activation-Menge. Begründung:
Im MVP gibt es keinen Check, der für ein Profil unzulässig ist —
strikt verbotene Kombinationen kommen erst mit den v0.2-Checks
(`LH-F-018`/-019, externe Dienste, ADR 0010). Der `Reason`-Code
`CheckNotAllowedInProfile` bleibt in `port/checkregistry.go` definiert
und wartet auf M5/v0.2.

### 2.5 Resource-Quantity-Parsing nur im Adapter

`resource.Quantity` lebt unter `k8s.io/apimachinery/pkg/api/resource`
und ist damit für den Domain-Layer per `domain-isolation`-depguard
verboten. Konsequenz:

- `domain.ClusterResourcesSpec.Validate(ctx)` prüft nur Syntax-Plausibilität
  (nicht-leer, keine kompletten Regex-Tests — k8s-API selbst lehnt
  fehlerhafte Werte beim `kubectl apply` ab via OpenAPI-Validierung).
- `adapter/check/clusterresources.go` ruft `resource.ParseQuantity` auf
  Spec-Werten **und** auf Node-`Allocatable`-Werten. Bei Parse-Fehler
  → `Status: Unknown` + `Reason: InvalidSpec` / `Reason: InvalidNodeQuantity`.

### 2.6 Cluster-Smoke-Erweiterung (ADR 0013)

Das aktuelle `scripts/cluster-smoke.sh` bringt kind hoch und appliziert
einen Single-Check-CR. Für M4 attestiert dieser Pfad nur den
KubernetesVersion-Check real; die anderen Checks würden auf bare-kind
unterschiedlich enden:

| Check | Bare-kind-Default-Ergebnis |
| ----- | -------------------------- |
| StorageClass mit `Names: ["standard"]` | passed (`standard` ist kind-Default) |
| StorageClass mit `RequireDefault: true` | passed (`standard` trägt `is-default-class=true`) |
| IngressClass mit beliebigem Namen | failed (kein IngressClass installiert) |
| cert-manager-Existence | failed (cert-manager nicht installiert) |
| ClusterResources mit Production-Defaults | failed (kind-Worker meist < 4 CPU) |

**Entscheidung:** Smoke-Pipeline installiert in der Prep-Phase
`ingress-nginx` und `cert-manager` per `kubectl apply` (statische
Manifeste, gepinnte Versionen; keine Helm-Pflicht für die Smoke).
**Manifest-Quelle ist verbindlich der Repo-Spiegel unter
`hack/cluster-smoke/` — Smoke zieht die Manifeste nicht zur Laufzeit
aus dem Netz**, damit die Pipeline reproducible und offline lauffähig
bleibt (siehe §9 für die Begründung). Smoke-CR konfiguriert alle vier
Checks mit Werten, die gegen bare-kind + die vorinstallierten
Manifeste **passed** sind (`MinCPU: "1"`, `MinMemory: "1Gi"`,
`IngressClass.Names: ["nginx"]`). Damit attestiert ein einziger
Smoke-Run alle vier `LH-AK-*`-Items in der passed-Variante real;
failed-Pfade bleiben Unit-Test-Pflicht.

Alternative überprüft: separate „expected-fail"-Smoke-CRs für die
nicht-installierten-Pfade — verworfen, weil das die Pipeline-Logik
verdoppelt; Unit-/Reconciler-Tests decken die failed-Pfade
deterministischer ab.

### 2.7 Versionspins

| Komponente | Pin in M4 | Quelle |
| ---------- | --------- | ------ |
| `k8s.io/api/storage/v1`, `…/networking/v1`, `…/core/v1` | identisch zur transitiven Auflösung aus controller-runtime v0.24.1 | unverändert |
| `k8s.io/client-go/kubernetes` (Typed Clientset) | identisch zur transitiven Auflösung aus controller-runtime v0.24.1 | als Direct-Require promotet |
| `k8s.io/apimachinery/pkg/api/resource` | identisch zur transitiven Auflösung | bereits Direct-Require über M2-Status-Typen |
| ingress-nginx Manifest (Smoke) | v1.13.x (kind-kompatibel, Floor v1.32+) | gepinnt in `scripts/cluster-smoke.sh` |
| cert-manager Manifest (Smoke) | v1.20.x (kind-kompatibel) | gepinnt in `scripts/cluster-smoke.sh` |

---

## 3. Datei-Inventar

### 3.1 Neue Code-Dateien

| Pfad | Zweck |
| ---- | ----- |
| `internal/hexagon/domain/storageclass.go` | `StorageClassSpec{Names []string, RequireDefault bool}` + `Kind()` + `Validate()`. Konstante `StorageClassSpecKind = "storageClass"`. |
| `internal/hexagon/domain/ingressclass.go` | `IngressClassSpec{Names []string}` + Kind/Validate. Konstante `IngressClassSpecKind = "ingressClass"`. |
| `internal/hexagon/domain/certmanager.go` | `CertManagerSpec{}` (parameterlos) + Kind/Validate. Konstanten `CertManagerSpecKind = "certManager"` und `CertManagerAPIGroup = "cert-manager.io"`. |
| `internal/hexagon/domain/clusterresources.go` | `ClusterResourcesSpec{MinCPU, MinMemory string}` + Kind/Validate (Syntax-only, kein Quantity-Parse). Konstanten für Production-/Evaluation-Profile-Defaults. |
| `internal/hexagon/port/storage.go` | `StorageClassDiscovery` + `StorageClassInfo{Name, IsDefault bool, Provisioner string}`. **Port-Vertrag für `IsDefault`:** Adapter setzt den Wert auf `true`, wenn **eine der beiden** Default-Annotationen aus §9 gesetzt ist (Legacy- + GA-Schlüssel). |
| `internal/hexagon/port/ingress.go` | `IngressClassDiscovery` + `IngressClassInfo{Name, Controller string}`. |
| `internal/hexagon/port/apigroups.go` | `APIGroupDiscovery` (`HasAPIGroup(ctx, name) (bool, error)`). |
| `internal/hexagon/port/nodes.go` | `NodeDiscovery` + `NodeInfo{Name string, AllocatableCPUMilli, AllocatableMemoryBytes int64}`. (Ports liefern bereits in skalaren Einheiten — `resource.Quantity` bleibt adapterseitig.) |
| `internal/adapter/k8s/storage.go` | `StorageClassAdapter` implementiert `port.StorageClassDiscovery` via `kubernetes.Interface.StorageV1().StorageClasses().List`. Liest Default-Annotation `storageclass.kubernetes.io/is-default-class` **und** den Legacy-Schlüssel `storageclass.beta.kubernetes.io/is-default-class` (entweder/oder reicht für `IsDefault=true`, siehe §9-Risiko). |
| `internal/adapter/k8s/ingress.go` | `IngressClassAdapter` implementiert `port.IngressClassDiscovery` via `NetworkingV1().IngressClasses().List`. |
| `internal/adapter/k8s/apigroups.go` | `APIGroupAdapter` implementiert `port.APIGroupDiscovery` via `discovery.DiscoveryInterface.ServerGroups`. (Discovery-Client wird aus dem bestehenden M3-Adapter wiederverwendet — gemeinsamer Konstruktor.) |
| `internal/adapter/k8s/nodes.go` | `NodeAdapter` implementiert `port.NodeDiscovery` via `CoreV1().Nodes().List`. Filter: nur `NodeReady=True`; Allocatable in millicpu/bytes umgerechnet (`resource.Quantity.MilliValue()` / `.Value()`). |
| `internal/adapter/check/storageclass.go` | `StorageClass`-Check + Konstanten `CheckNameStorageClass`, `ConditionTypeStorageClassReady`, Reason-Codes (`StorageClassReady`, `StorageClassMissing`, `DefaultStorageClassMissing`, `LookupFailed`). |
| `internal/adapter/check/ingressclass.go` | analog StorageClass. |
| `internal/adapter/check/certmanager.go` | `CertManager`-Check; Reason `CertManagerInstalled` / `CertManagerMissing`. |
| `internal/adapter/check/clusterresources.go` | `ClusterResources`-Check; parst Spec-Quantities + Node-Quantities, summiert über Ready-Nodes, vergleicht. Reason: `ResourcesSufficient`, `InsufficientCPU`, `InsufficientMemory`, `InvalidSpec`, `InvalidNodeQuantity`, `LookupFailed`. |

### 3.2 Erweiterte Code-Dateien

| Pfad | Änderung |
| ---- | -------- |
| `api/v1alpha1/opendeskpreflightcheck_types.go` | `ChecksSpec` erweitert um vier Sibling-Pointer (`StorageClass`, `IngressClass`, `CertManager`, `ClusterResources`). Vier neue Check-Sub-Spec-Typen mit kubebuilder-Markern (`+kubebuilder:validation:MinItems=1` für `Names`-Listen, `+kubebuilder:validation:Pattern` für resource-Quantity-Strings). Kein `+kubebuilder:default` für die neuen Sub-Specs — Profile-Defaults laufen ausschließlich code-seitig im Reconciler. |
| `internal/hexagon/application/reconciler.go` | `buildSpecMap` erweitert: vier neue Branches, jeweils mit den §2.2-Activation-Regeln und §2.3-Profile-Defaults. `profileWithDefault` wird zu einer Profile-Resolution-Funktion, die zusätzlich die ClusterResources-Default-Werte pro Profile liefert. |
| `internal/adapter/k8s/discovery.go` | Wenn nicht aufgespalten: bleibt für `ServerVersion` zuständig. Wenn aufgespalten (siehe Reihenfolge-Schritt 1): wird zu `ServerVersionAdapter` und teilt einen gemeinsamen `*kubernetes.Clientset` mit den vier neuen Adaptern. **Empfehlung:** gemeinsamer `clientset` + `discoveryClient` zentral in einer neuen Konstruktor-Bündel-Funktion `NewClusterClients(cfg)`. |
| `cmd/operator/main.go` | Wiring: `kubernetes.NewForConfig(cfg)` als gemeinsame Quelle; jeweils ein Adapter pro Discovery-Port konstruieren; `registry.Register(...)` für die vier neuen Checks. `Now`-Override bleibt single-shot `time.Now`. |
| `config/samples/k-deskflight_v1alpha1_opendeskpreflightcheck.yaml` | Erweitert um vollständiges Sub-Spec-Beispiel pro Check (oder zweites Sample-File `…_full.yaml`); Werte gegen bare-kind-Smoke passed. |
| `scripts/cluster-smoke.sh` | Prep-Phase erweitert: `ingress-nginx`-Static-Manifest applizieren, `cert-manager`-Static-Manifest applizieren, `kubectl wait` auf die jeweiligen Deployments. Pinned Manifest-URLs als Konstanten am Skript-Anfang. |

### 3.3 Neue Test-Dateien

| Pfad | Coverage |
| ---- | -------- |
| `internal/hexagon/domain/storageclass_test.go` | Tabelle: leere Names, gültige Names, Names mit Whitespace; Validate-Pfade. |
| `internal/hexagon/domain/ingressclass_test.go` | analog. |
| `internal/hexagon/domain/certmanager_test.go` | Smoke-Test für Kind/Validate (parameterloses Spec). |
| `internal/hexagon/domain/clusterresources_test.go` | Validate mit leeren Strings, validen `"2"` / `"4Gi"`, ungültigen Strings (Syntax-only); Kind. |
| `internal/adapter/check/storageclass_test.go` | Fake `StorageClassDiscovery`: passed (alle Names + Default vorhanden), failed (ein Name fehlt), failed (Default angefordert, keiner gesetzt), unknown (Lookup-Error). |
| `internal/adapter/check/ingressclass_test.go` | Fake-Port; passed / failed / unknown. |
| `internal/adapter/check/certmanager_test.go` | Fake `APIGroupDiscovery`: present → passed (Severity `info`), missing → failed (Severity `warning` — cert-manager-Fehlen ist nicht critical, weil OpenDesk auch ohne deployt werden kann; finale Severity ist Slice-Plan-Entscheidung, Begründung siehe §9). |
| `internal/adapter/check/clusterresources_test.go` | Fake `NodeDiscovery`: passed (Summen ≥ Min), failed-cpu, failed-memory, failed-beide, unknown (Lookup-Error), unknown (Quantity-Parse-Fehler im Spec), unknown (Quantity-Parse-Fehler in Node-Allocatable). |
| `internal/hexagon/application/reconciler_test.go` (Erweiterung) | Neue Fälle: Multi-Check-Run (alle vier passed → Phase `Passed`, vier Conditions), Multi-Check-Run mit einem Failed-critical und einem Passed → Phase `Failed`, Profile-Default-Activation für ClusterResources (production vs. evaluation Defaults). Bestehende Fake-Registry-Helper wird um die vier neuen Stub-Checks erweitert. |

### 3.4 Erweiterte Beispiel-Manifeste

`config/samples/k-deskflight_v1alpha1_opendeskpreflightcheck.yaml`
wird erweitert um:

```yaml
checks:
  kubernetesVersion:
    min: "1.34"
  storageClass:
    names: ["standard"]
    requireDefault: true
  ingressClass:
    names: ["nginx"]
  certManager: {}
  clusterResources:
    minCPU: "1"
    minMemory: "1Gi"
```

Werte zielen auf bare-kind + ingress-nginx + cert-manager (siehe §2.6);
Production-Anwender ersetzen `minCPU`/`minMemory` mit realistischen
Werten oder lassen die Sub-Spec leer und nutzen Profile-Defaults.

### 3.5 Generated-Drift-Refresh

`make manifests` muss neu laufen, weil:

- CRD-Schema bekommt vier neue Sub-Spec-Objekte (`config/crd/…yaml`).
- DeepCopy-Code bekommt vier neue Sub-Typen (`zz_generated.deepcopy.go`).
- RBAC-Marker am Reconciler bleiben unverändert — die in M2
  pre-granted-Verben für `nodes`, `storageclasses`, `ingressclasses`,
  `customresourcedefinitions` sind exakt der M4-Bedarf.
  `config/rbac/role.yaml` sollte daher idempotent bleiben; Drift-Gate
  verifiziert das.

---

## 4. Reihenfolge der Umsetzung

Jeder Schritt = ein eigener Commit; lokal über `make gates` grün
ziehen bevor gepusht wird. Die §3.1-Adapter-Bündelung passiert
in Schritt 2, damit Schritt 1 reine Domain-/Port-Definitionen
bleibt.

1. **Domain + Port-Definitionen.** Acht Files neu unter
   `internal/hexagon/{domain,port}/`. Reine Typen, keine Logik.
   Domain-Tests mit Validate-Tabellen. Aggregator/Reconciler
   bleiben unverändert.
2. **Adapter-Schicht.** Fünf neue Files unter `internal/adapter/k8s/`
   (gemeinsamer `NewClusterClients`-Konstruktor + vier Adapter).
   Vier Check-Implementierungen unter `internal/adapter/check/`. Tests
   pro Check mit lokalen Fake-Ports. `adapter/k8s/*`-Adapter
   bleiben M4-untest (envtest-Pflicht für reale API-Calls ist M6,
   parallel zur K8sVersion-discovery-Lücke).
3. **CRD-Schema-Erweiterung.** `api/v1alpha1/…_types.go` um vier
   Sub-Spec-Typen erweitern, `make manifests` laufen lassen,
   `controller-gen`-Output commiten (zz_generated.deepcopy.go +
   config/crd/…yaml). Drift-Gate verifiziert.
4. **Reconciler-Erweiterung.** `buildSpecMap` für alle vier Checks
   inkl. Profile-Defaults für ClusterResources. `profileWithDefault`
   bekommt einen Begleit-Typ `profileDefaults` (oder ähnlich), der
   ClusterResources-Werte liefert. Multi-Check-Reconciler-Tests
   (Multi-passed, gemischt-failed, Profile-Defaults).
5. **Wiring + Beispiel + Smoke-Skript.** `cmd/operator/main.go` zieht
   `kubernetes.NewForConfig` + vier Adapter + vier `registry.Register`.
   `config/samples/…yaml` auf den §3.4-Stand bringen.
   `scripts/cluster-smoke.sh` um ingress-nginx + cert-manager-Prep
   ergänzen; Smoke-CR auf den Multi-Check-Stand heben.
6. **Slice-Closure.** Nach `done/` ziehen, Roadmap-Status M4 = Done,
   Closure-Notiz mit Verifikations-Ergebnis und etwaigen Folge-Attesten.

---

## 5. Lastenheft-Kennungen

`LH-F-010` (StorageClass prüfen), `LH-F-011` (Default StorageClass
erkennen), `LH-F-012` (IngressClass prüfen), `LH-F-013` (cert-manager
prüfen), `LH-F-015` (Ressourcen prüfen), `LH-F-031` (Schweregrad —
Pflicht-Severity pro Result), `LH-F-032` (Ergebnis-Inhalt — Result-
Struktur weiter genutzt), `LH-NF-003` (Nachvollziehbarkeit —
strukturiertes slog-Logging pro Check), `LH-NF-005` (Fehlertoleranz —
M4-Ausbaustufe: Einzelausfälle erzeugen `Unknown`-Results, andere
Checks laufen; volle panic-Härtung bleibt M5), `LH-PROF-002` /
`LH-PROF-003` (Profile-Defaults für ClusterResources, §2.3),
`LH-DAT-003` (Zeitstempel pro Result), `LH-QA-001` (verständliche
Fehlermeldungen — Message-Texte explizit pro Reason-Code).

---

## 6. Architekturartefakte

Erfüllt aus `spec/architecture.md`:

- `AR-009` Phase 1+3+4 (sequenziell, vier Checks)+5+6 — scharf,
  unverändert gegenüber M3.
- `AR-012` (Check-Interface) — vier neue Implementierungen.
- `AR-013` (Check-Registry) — vier neue `Register`-Aufrufe.
- `AR-014` (Schweregrad-Aggregation + Dedupe/Sort) — unverändert,
  wird durch vier neue Conditions exerziert.
- `AR-015` (ClusterRole MVP-Minimum) — RBAC ist bereits in M2
  pre-granted; M4 verifiziert nur Drift-Stabilität.

Vorbereitet, aktiv ab späterer Slice:

- `AR-009` Phase 2 voll (Cross-Constraint-Timeout) — M5.
- `AR-009` Phase 4 Worker-Pool + Panic-Boundary — M5.
- `AR-010` (Wiederholintervall) — M5.
- `AR-011` (Error-Handling-Härte, leader-election) — M5.
- `AR-018` (SelfSubjectAccessReview-Right operativ) — M5.

---

## 7. Verifikation (Abnahmekriterien)

1. **`make build`** baut alle vier neuen Adapter + Checks + den
   erweiterten Reconciler.
2. **`make lint`** grün — neue Files unter
   `internal/hexagon/{domain,port}/` und `internal/adapter/{check,k8s}/`
   verletzen die `AR-005`-depguard-Regeln nicht. Insbesondere:
   `internal/hexagon/domain/clusterresources.go` importiert kein
   `k8s.io/apimachinery/pkg/api/resource` (Parsing bleibt im Adapter).
3. **`make test`** grün; alle neuen Unit-Tests bestehen; Multi-Check-
   Reconciler-Tests erfolgreich.
4. **`make coverage-gate`** grün — Erwartung: Coverage über
   `internal/hexagon/{domain,application,adapter/check}` bleibt deutlich
   ≥ 80 %. Die vier neuen `adapter/k8s/*`-Adapter bleiben M4-untestet
   (envtest-Pflicht M6) und drücken den Schnitt erwartbar — akzeptiert,
   solange der Schnitt ≥ Schwelle 0 bleibt. M6 hebt strikt auf 90 %.
5. **`make doc-refs`** grün — neue Cross-Refs (`LH-F-010..015`,
   `LH-AK-006..009`, ADR 0010, ADR 0013) sind valide.
6. **`make generated-drift-check`** grün — CRD-YAML und
   `zz_generated.deepcopy.go` sind committet; `config/rbac/role.yaml`
   bleibt unverändert (Pre-Grant aus M2 deckt M4-Bedarf 1:1).
7. **`make gates`** grün (Bundle aus 1+2+3+4+5+6).
8. **`make security-gates`** grün — `govulncheck` ohne neue Findings
   für das neu promotete `k8s.io/client-go/kubernetes`-Paket.
9. **`LH-AK-006` StorageClass prüfbar** — `adapter/check/storageclass_test.go`
   deckt passed-/failed-/unknown-Pfade synthetisch ab; Cluster-Smoke
   attestiert passed real gegen kind `standard`-StorageClass.
10. **`LH-AK-007` IngressClass prüfbar** — analog; Cluster-Smoke
    attestiert passed real gegen ingress-nginx im kind-Cluster.
11. **`LH-AK-008` cert-manager prüfbar** — analog; Cluster-Smoke
    attestiert passed real gegen vorinstallierten cert-manager im
    kind-Cluster.
    **Severity-Erwartung (bewusste Entscheidung):** ein fehlender
    cert-manager liefert `Status: False` mit `Severity: warning`
    und setzt den Gesamtstatus auf `Warning` — **nicht** auf
    `Failed`. `LH-F-013` ist als Capability erfüllt (Operator kann
    erkennen), nicht als Outcome-Blocker (fehlender cert-manager
    blockiert keine OpenDesk-Installation). Begründung,
    Erwartungsdissonanz-Risiko und Rollback-Pfad: §9.
    Unit-Test `internal/adapter/check/certmanager_test.go` fixiert
    diese Severity gegen versehentliche Änderungen.
12. **`LH-AK-009` Ressourcen prüfbar** — `adapter/check/clusterresources_test.go`
    deckt synthetische Node-Mengen ab; Cluster-Smoke attestiert passed
    real gegen kind-Worker-Allocatable mit Smoke-Min-Werten
    `1 CPU / 1Gi`.

---

## 8. Out-of-Scope (geht in M5–M7 oder später)

- **RBAC-Selbstprüfung** (`SelfSubjectAccessReview`) — M5.
- **Robustheit** (Panic-Recovery, Timeout-Härtung pro Check,
  Leader-Election, Cross-Constraint-Validierung) — M5.
- **Metrics-Endpoint volle Auswertung + envtest-Suite** — M6.
- **`LH-F-014` ClusterIssuer-Detailprüfung** — v0.2 (`ADR 0010`).
- **`LH-F-016` Node-Anzahl-Prüfung, `LH-F-017` Node-Zustand
  jenseits Ready** — v0.2.
- **Profile-spezifische Activation-Listen** in `Registry.ListByProfile`
  (`CheckNotAllowedInProfile`) — bleibt vorbereitet, wird erst
  aktiviert, wenn ein Profil einen Check verbietet (v0.2 mit externen
  Diensten).

---

## 9. Risiken und Mitigation

- **`storageclass.kubernetes.io/is-default-class`-Annotation kann
  fehlen** auch wenn eine Default-StorageClass existiert (alte
  Cluster nutzen `storageclass.beta.kubernetes.io/is-default-class`).
  Mitigation: Adapter prüft **beide** Annotationen.
- **`networkingv1.IngressClass` ist erst ab K8s 1.19 GA** —
  Operator-Floor ist `1.34` (`ADR 0009`), daher unkritisch; ein
  Fallback auf `networkingv1beta1` ist nicht erforderlich.
- **cert-manager-Severity-Wahl:** `Warning` (nicht `Critical`).
  Begründung: OpenDesk kann ohne cert-manager deployt werden, wenn TLS
  extern terminiert wird; ein fehlender cert-manager ist daher kein
  Installations-Blocker. **Erwartungsdissonanz-Risiko:** `LH-F-013`
  steht in `LH-PRI-001` als MVP-Pflicht-Check — Anwender könnten
  daraus ableiten, dass ein fehlender cert-manager den Gesamtstatus
  auf `Failed` setzen müsste. `LH-PRI-001` formuliert aber „muss
  prüfbar sein" (Capability), nicht „muss bestanden werden" (Outcome).
  **Verbindliche Mitigation:** Die M6-Anwender-Doku
  (`docs/user/conditions-katalog.md`) muss diese Unterscheidung
  explizit aussprechen — ein eigener Eintrag für
  `CertManagerInstalled` mit Severity-Begründung und Verweis auf
  externe TLS-Terminierungs-Szenarien. Falls Beta-Feedback die Wahl
  kippt, ist die Änderung lokal in `adapter/check/certmanager.go`
  (eine Konstante) und in den vier dazugehörigen Tests; CRD-Schema
  bleibt unangetastet.
- **`resource.Quantity`-Vergleich mit `Cmp`:** semantisch korrekt
  (Quantity normalisiert Einheiten, `"1000m" == "1"` und `"1Gi"`
  ≠ `"1G"`). Test deckt `Gi` vs. `G` ab, damit Edge-Cases nicht
  bei Anwendern auflaufen.
- **Cluster-Smoke-Manifest-URLs sind externe Abhängigkeiten:**
  ingress-nginx- und cert-manager-Releases können vorübergehend
  nicht erreichbar sein. Mitigation (verbindlich, deckungsgleich mit
  §2.6): Manifeste werden im Repo unter `hack/cluster-smoke/`-Mirror
  eingecheckt und ausschließlich von dort applizierten — kein
  Network-Fetch zur Laufzeit. Damit ist die Smoke idempotent und
  offline lauffähig; ein Upstream-Release-Sprung ist ein bewusster
  Repo-Commit, kein implizites Pipeline-Verhalten.
- **Profile-Default für ClusterResources beeinflusst alle
  Production-CRs** — wenn die `4 CPU / 8Gi`-Floor in der Praxis zu
  streng ist, blockiert das Anwender, die mit `spec.checks.clusterResources: {}`
  starten. Mitigation: Anwender-Doku (M6) verweist explizit auf den
  Default und den Override-Pfad. Floor ist im Closure-Review
  reevaluierbar, falls Beta-Feedback das zeigt.
- **`kubectl wait` auf cert-manager-Deployments in kind** kann
  länger dauern als der Smoke-Timeout (cert-manager startet drei
  Deployments + CRDs). Mitigation: Smoke-Wait auf jedes
  cert-manager-Deployment einzeln mit `--timeout=180s`.

---

## 10. Closure

— bleibt leer bis zum Schließen der Slice.
