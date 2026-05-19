# Conditions-Katalog

> **Adressat:** Cluster-Betreiber, die `kubectl describe opdc <name>`
> lesen und die Reason-Codes interpretieren wollen — sowie Schreiber
> von Alert-Regeln / Dashboards, die nach konkreten Reasons filtern.
>
> **Stand:** MVP v0.1. Alle hier aufgeführten Reason-Codes sind
> stabil über die `v1alpha1`-Lebensdauer; neue Reasons können in
> Patch-Releases hinzukommen, bestehende werden nicht umbenannt
> ohne ADR-Folge.

---

## 1. Lese-Konvention

Pro Condition-Typ steht eine Tabelle mit den möglichen Reason-Codes,
ihrer Severity (`critical` / `warning` / `info`), einer kurzen
Bedeutung und einer Anwender-Action.

Doku-Konvention (intern, für den Conditions-Drift-Check):

- Jeder Reason-Code wird mit dem Pflicht-Schema **`**Reason:** <Name>`**
  in der zugehörigen Tabellen-Zeile aufgeführt.
- Quelle der Wahrheit: die `Reason*`-/`reason*`-Konstanten in
  [`internal/adapter/check/`](../../internal/adapter/check/) und
  [`internal/hexagon/application/`](../../internal/hexagon/application/).
- **Bewusste grep-Lücke:** Drei Reasons leben im Code als
  String-Literals statt als Konstanten — `"SpecInvalid"` und
  `"UnknownCheck"` (aus
  [`internal/hexagon/application/reconciler.go`](../../internal/hexagon/application/reconciler.go)
  + [`internal/adapter/check/registry.go`](../../internal/adapter/check/registry.go)).
  Der strukturierte grep-Pattern matched sie deshalb nicht — sie
  sind hier in §9 vollständig dokumentiert und müssen bei
  v0.2-Refactor auf Konstanten gehoben werden, falls weitere
  String-Literal-Reasons dazukommen.
- **ConditionType-Konstanten** (`ConditionTypeXxx`) tauchen im
  grep-Output mit auf, sind aber **keine** Reasons, sondern die
  Type-Felder, die als Sektion-Überschriften §2–§9 erscheinen. Pro
  Type ein eigener Doku-Abschnitt.
- Neue Reasons im Code → Eintrag in diesem Dokument; siehe
  [Slice-M6 §4 Step 5](../plan/planning/done/slice-M6-metrics-tests-doku.md)
  für den `make`-losen strukturierten grep-Check, der die
  Konsistenz pro PR-Review verifiziert.

**Reason kommt aus zwei Quellen:**

1. **Vom Check selbst** — die in §2–§6 pro ConditionType
   aufgeführten Reasons (z. B. `KubernetesVersionTooOld`,
   `StorageClassMissing`). Das ist der Normalfall.
2. **Vom Per-Check-Runner** (slice-M5 §2.3-§2.5) — wenn ein
   Check via SAR-Pre-Execution, Per-Check-Timeout oder
   Per-Check-Recover unterbrochen wurde, **ersetzt** der Runner
   das Check-Result durch eines aus der §10-Liste
   (`RBACInsufficient`, `RBACCheckFailed`, `Timeout`,
   `ReconcileTimeout`, `ReconcileCanceled`, `InternalError`).
   Diese Reasons können in **jeder** Check-Condition erscheinen
   und sind nicht in den §2–§6-Tabellen wiederholt.

Wer also z. B. `KubernetesVersionReady=Unknown` mit
`reason: RBACInsufficient` sieht, sucht in §10, nicht in §2.

---

**Severity → Phase-Mapping** (aus
[`spec/architecture.md` AR-014](../../spec/architecture.md)):

| Severity | Phase-Effekt | Bedeutung |
| -------- | ------------ | --------- |
| `critical` | Failed | Blockierender Befund — Installation nicht empfohlen. |
| `warning` | Warning (wenn keine critical da ist) | Nicht-blockierend; Anwender sollte aber prüfen. |
| `info` | Passed (oder neutral) | Diagnose, keine Action nötig. |

---

## 2. ConditionType `KubernetesVersionReady`

Vom Check `KubernetesVersion` gesetzt (Adapter:
[`internal/adapter/check/kubernetesversion.go`](../../internal/adapter/check/kubernetesversion.go)).

| Reason | Severity | Status | Bedeutung | Anwender-Action |
| ------ | -------- | ------ | --------- | --------------- |
| **Reason:** KubernetesVersionReady | info | True | Server-Version erfüllt die konfigurierte Mindestversion. | Keine. |
| **Reason:** KubernetesVersionTooOld | critical | False | Server-Version unter `spec.checks.kubernetesVersion.min`. | Cluster updaten oder `min` absenken — beachte Operator-Floor in [ADR 0009 §2.2](../plan/adr/0009-k8s-versions-support-und-profile-mindestversionen.md). |
| **Reason:** ServerVersionLookupFailed | warning | Unknown | Discovery-API nicht erreichbar oder fehlerhafte Antwort. | Cluster-Discovery prüfen (`kubectl version`, API-Server-Logs). |
| **Reason:** ServerVersionParseFailed | warning | Unknown | Server-Version-String nicht in `MAJOR.MINOR(.PATCH)`-Form. | Sehr selten — meist symptomatisch für custom-K8s-Build; Bug-Report öffnen. |
| **Reason:** MinVersionParseFailed | critical | Unknown | `spec.checks.kubernetesVersion.min` ist kein gültiges Versions-Format. | `spec.checks.kubernetesVersion.min` auf `MAJOR.MINOR`-Form korrigieren (Beispiel: `"1.34"`). |
| **Reason:** InvalidSpec | critical | Unknown | Cross-Field-Validierung der Spec gescheitert. | Spec-Felder prüfen — siehe Status-Message. |

---

## 3. ConditionType `StorageClassReady`

Vom Check `StorageClass` gesetzt (Adapter:
[`internal/adapter/check/storageclass.go`](../../internal/adapter/check/storageclass.go)).

| Reason | Severity | Status | Bedeutung | Anwender-Action |
| ------ | -------- | ------ | --------- | --------------- |
| **Reason:** StorageClassReady | info | True | Alle konfigurierten `spec.checks.storageClass.names` existieren; falls `requireDefault: true`, ist eine Default-StorageClass markiert. | Keine. |
| **Reason:** StorageClassMissing | critical | False | Mindestens ein konfigurierter Name fehlt im Cluster. | `kubectl get storageclass` — entweder die fehlende StorageClass anlegen oder den Namen aus der Spec entfernen. |
| **Reason:** DefaultStorageClassMissing | critical | False | `requireDefault: true`, aber keine StorageClass trägt die `storageclass.kubernetes.io/is-default-class`-Annotation (oder den Beta-Key). | Eine StorageClass per `kubectl patch storageclass <name> -p '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'` als Default markieren. |
| **Reason:** StorageClassLookupFailed | warning | Unknown | API-Aufruf an `storage.k8s.io/v1/storageclasses` ist gescheitert (Permission, Connectivity). | RBAC prüfen — der Operator-ServiceAccount braucht `list` auf `storage.k8s.io/storageclasses`. |
| **Reason:** InvalidSpec | critical | Unknown | `storageClass.names` ist leer oder enthält ungültige Werte. | Spec-Liste prüfen — mindestens ein Eintrag erforderlich, sobald die Sub-Spec gesetzt ist. |

---

## 4. ConditionType `IngressClassReady`

Vom Check `IngressClass` gesetzt (Adapter:
[`internal/adapter/check/ingressclass.go`](../../internal/adapter/check/ingressclass.go)).

| Reason | Severity | Status | Bedeutung | Anwender-Action |
| ------ | -------- | ------ | --------- | --------------- |
| **Reason:** IngressClassReady | info | True | Alle konfigurierten `spec.checks.ingressClass.names` existieren. | Keine. |
| **Reason:** IngressClassMissing | critical | False | Mindestens ein konfigurierter Name fehlt im Cluster. | `kubectl get ingressclass` — fehlende IngressClass anlegen oder Namen aus der Spec entfernen. |
| **Reason:** IngressClassLookupFailed | warning | Unknown | API-Aufruf an `networking.k8s.io/v1/ingressclasses` ist gescheitert. | RBAC prüfen — `list` auf `networking.k8s.io/ingressclasses`. |
| **Reason:** InvalidSpec | critical | Unknown | `ingressClass.names` ist leer oder enthält ungültige Werte. | Spec-Liste prüfen — mindestens ein Eintrag erforderlich, sobald die Sub-Spec gesetzt ist. |

---

## 5. ConditionType `CertManagerInstalled`

Vom Check `CertManager` gesetzt (Adapter:
[`internal/adapter/check/certmanager.go`](../../internal/adapter/check/certmanager.go)).

| Reason | Severity | Status | Bedeutung | Anwender-Action |
| ------ | -------- | ------ | --------- | --------------- |
| **Reason:** CertManagerInstalled | info | True | Die API-Gruppe `cert-manager.io` ist im Cluster registriert. | Keine. |
| **Reason:** CertManagerMissing | warning | False | API-Gruppe `cert-manager.io` ist nicht registriert. **Nicht** critical: für viele OpenDesk-Deployments ist cert-manager optional (eigene TLS-Termination per Ingress-Controller, externer Cert-Issuer). Severity-Entscheidung in [slice-M4 §9](../plan/planning/done/slice-M4-cluster-state-checks.md). | cert-manager installieren ([Upstream-Doku](https://cert-manager.io/docs/installation/)) oder die fehlende API-Gruppe als bewusst akzeptieren. |
| **Reason:** CertManagerLookupFailed | warning | Unknown | Discovery-API ist nicht erreichbar. | Cluster-Discovery prüfen (`kubectl api-resources`). |
| **Reason:** InvalidSpec | critical | Unknown | `certManager`-Sub-Spec hat ungültige Werte (heute parameterlos; v0.2 bringt ClusterIssuer-Detail-Validierung). | Spec prüfen. |

---

## 6. ConditionType `ClusterResourcesReady`

Vom Check `ClusterResources` gesetzt (Adapter:
[`internal/adapter/check/clusterresources.go`](../../internal/adapter/check/clusterresources.go)).

Die Check-Logik summiert die `allocatable`-Werte (CPU, Memory) aller
`Ready`-Nodes und vergleicht gegen die konfigurierten Mindestwerte.

| Reason | Severity | Status | Bedeutung | Anwender-Action |
| ------ | -------- | ------ | --------- | --------------- |
| **Reason:** ResourcesSufficient | info | True | Allocatable-Summe ≥ konfigurierte Min. | Keine. |
| **Reason:** InsufficientCPU | critical | False | Nur CPU-Summe < `minCPU`; Memory ausreichend. | Cluster-CPU-Kapazität erhöhen (mehr Nodes oder größere) oder `minCPU` absenken. |
| **Reason:** InsufficientMemory | critical | False | Nur Memory-Summe < `minMemory`; CPU ausreichend. | Cluster-Memory erhöhen oder `minMemory` absenken. |
| **Reason:** InsufficientResources | critical | False | Beide Werte zu niedrig. | Cluster-Kapazität insgesamt erhöhen oder beide Mindestwerte absenken. |
| **Reason:** ClusterResourcesLookupFailed | warning | Unknown | API-Aufruf an `nodes` ist gescheitert. | RBAC prüfen — `list` auf `core/nodes`. |
| **Reason:** InvalidSpec | critical | Unknown | `clusterResources.minCPU` / `minMemory` sind keine gültigen Kubernetes `resource.Quantity`-Strings. | Werte auf `resource.Quantity`-Format korrigieren (`"4"`, `"500m"`, `"8Gi"`, `"2048Mi"`). |

---

## 7. ConditionType `ConfigurationInvalid` (CR-Spec-Scope)

Vom Reconciler gesetzt, wenn `spec.interval` normalisiert werden
musste (Adapter:
[`internal/hexagon/application/interval.go`](../../internal/hexagon/application/interval.go)).

Die `ConfigurationInvalid`-Condition zählt **nicht** in
`Summary.passed/warning/failed/unknown` — sie ist CR-Spec-Scope, kein
Check-Result. Sie hebt die Phase aber **mindestens** auf `Warning`
(falls die Aggregator-Phase `Passed` war); `Failed`/`Unknown` bleiben
unverändert (siehe
[Slice-M6 §2.3 `mergeIntervalWarning`](../plan/planning/done/slice-M6-metrics-tests-doku.md)).

| Reason | Severity | Status | Bedeutung | Anwender-Action |
| ------ | -------- | ------ | --------- | --------------- |
| **Reason:** IntervalUnparseable | warning | True | `spec.interval` ist kein gültiges `time.ParseDuration`-Format. Reconciler fällt auf Default `5m` zurück. | `spec.interval` auf Format `30s` / `5m` / `1h30m` korrigieren. Siehe [cr-examples.md §5](cr-examples.md). |
| **Reason:** IntervalClampedMin | warning | True | `spec.interval` parsed gültig, ist aber `< 30s` (`MinInterval`). Reconciler clampt auf `30s`. | Anwender-Wert auf ≥ `30s` heben, oder das Clamp-Verhalten akzeptieren. |
| **Reason:** IntervalClampedMax | warning | True | `spec.interval` parsed gültig, ist aber `> 24h` (`MaxInterval`). Reconciler clampt auf `24h`. | Anwender-Wert auf ≤ `24h` senken, oder das Clamp-Verhalten akzeptieren. |

---

## 8. ConditionType `ReconcileError` (Operator-interne Diagnose)

Vom Reconciler gesetzt, wenn der Outer-`defer/recover` einen Panic
abgefangen hat (Adapter:
[`internal/hexagon/application/reconciler.go`](../../internal/hexagon/application/reconciler.go)).
Phase wird auf `Unknown` gesetzt; der Status enthält **nur** diese
Condition (kein Aggregat aus Check-Results).

| Reason | Severity | Status | Bedeutung | Anwender-Action |
| ------ | -------- | ------ | --------- | --------------- |
| **Reason:** ReconcilePanic | critical | False | Ein unerwarteter Panic im Reconciler ist via Outer-Recover abgefangen worden. Stack-Trace landet im Operator-Log (sanitisiert per `LogResult`-Hook), **nicht** im CR-Status (LH-SEC-002 / LH-NF-007). | Operator-Pod-Logs (`kubectl logs -n k-deskflight-system deployment/k-deskflight-operator`) prüfen und Bug-Report öffnen. |

---

## 9. ConditionType `SpecInvalid` (Aggregat-Pseudo-Type)

Vom Reconciler gesetzt, wenn `buildSpecMap` Validierungsfehler aus
einer oder mehreren Check-Sub-Specs sammelt, **bevor** die Checks
ausgeführt werden — oder wenn die `CheckRegistry`-Resolution
unauflösbare Einträge meldet (Adapter:
[`internal/hexagon/application/reconciler.go`](../../internal/hexagon/application/reconciler.go)).

Phase wird auf `Failed` gesetzt; der Status enthält **nur** diese
eine `SpecInvalid`-Condition (kein Aggregat aus erfolgreichen Checks).

| Reason | Severity | Status | Bedeutung | Anwender-Action |
| ------ | -------- | ------ | --------- | --------------- |
| **Reason:** SpecInvalid | critical | False | Spec-Validierung einer oder mehrerer Sub-Specs fehlgeschlagen. Status-Message listet die Sub-Specs mit ihrem jeweiligen Fehlertext. | Spec-Felder gemäß Message korrigieren — siehe die `InvalidSpec`-Zeilen in den Tabellen §2–§6. |
| **Reason:** UnknownCheck | critical | False | Registry konnte einen aus `buildSpecMap` erzeugten Check-Kind nicht auflösen. Tritt im MVP nur bei manueller Code-Modifikation auf. | Bug-Report öffnen — der Reconciler sollte nur bekannte Spec-Kinds in `buildSpecMap` erzeugen. |

---

## 10. Per-Check-Runner-Reasons (Robustheit, Slice-M5)

Diese Reasons können in **jeder** Check-Condition (`KubernetesVersionReady`,
`StorageClassReady`, …) erscheinen, wenn der Per-Check-Runner einen
Fehler abgefangen hat (Adapter:
[`internal/hexagon/application/runner.go`](../../internal/hexagon/application/runner.go)).
Sie werden **nicht** vom Check selbst gesetzt — der Runner ersetzt
das Check-Result, wenn ein Hänger, Cancel oder Panic auftritt.

| Reason | Severity | Status | Bedeutung | Anwender-Action |
| ------ | -------- | ------ | --------- | --------------- |
| **Reason:** RBACInsufficient | critical | Unknown | `SelfSubjectAccessReview` hat `denied` zurückgegeben — dem Operator-SA fehlt eines der vom Check deklarierten Rechte. | RBAC prüfen — `kubectl auth can-i <verb> <resource>` mit dem Operator-SA. ClusterRole im `deploy/manifests/`-Set deckt alle MVP-Checks ab (LH-AK-015). |
| **Reason:** RBACCheckFailed | critical | Unknown | SAR-Subsystem-Fehler (z. B. `authorization.k8s.io`-API nicht erreichbar). **Unterschied zu `RBACInsufficient`**: transient/infrastrukturell, nicht Permission-Drift (slice-M5 §2.3). | Operator-Log prüfen (Error-Eintrag mit Wrap), Auth-Webhook-Status verifizieren. |
| **Reason:** Timeout | critical | Unknown | Per-Check-Timeout überschritten (Default 30s aus `runner.go.defaultCheckTimeout`). Der jeweilige Check hat zu lange gebraucht. | Cluster-Connectivity prüfen (Discovery / API-Server-Latenz). |
| **Reason:** ReconcileTimeout | critical | Unknown | Parent-Reconcile-Timeout (`defaultReconcileTimeout`, 120s) hat den Check abgebrochen, **bevor** das Per-Check-Timeout greifen konnte. | Cluster-Performance prüfen — alle MVP-Checks zusammen sollten in < 30s durchlaufen. |
| **Reason:** ReconcileCanceled | info | Unknown | Reconcile-Kontext wurde gecancelt (z. B. Operator-Shutdown, Leader-Election-Loss). Kein Anwender-Fehler. | Keine, sofern der nächste Reconcile sauber durchläuft. |
| **Reason:** InternalError | critical | Unknown | Panic im Check-Code via Per-Check-Recover abgefangen. Stack-Trace im Operator-Log (sanitisiert). | Operator-Pod-Logs prüfen, Bug-Report öffnen. |

---

## 11. Diagnostische Sicht: was `kubectl describe opdc` zeigt

Schematische Darstellung der `Status`-Sektion eines voll-passierten
CRs mit `evaluation`-Profil (`kubectl describe` formatiert
Field-Namen mit Leerzeichen, z. B. `Last Transition Time:`; die
yaml-nahe Form unten ist als Lese-Hilfe verkürzt):

```yaml
Status:
  Phase: Passed
  Observed Generation: 1
  Summary:
    Checks Total: 5
    Passed: 5
    Failed: 0
    Warning: 0
    Unknown: 0
    Last Checked: 2026-05-19T12:00:00Z
  Conditions:                    # alphabetisch sortiert (aggregator.go)
    - Type: CertManagerInstalled
      Status: "True"
      Reason: CertManagerInstalled
      Message: API group "cert-manager.io" is registered
      Severity: info
      Last Transition Time: 2026-05-19T12:00:00Z
    - Type: ClusterResourcesReady
      Status: "True"
      Reason: ResourcesSufficient
      ...
    # drei weitere Conditions (IngressClassReady, KubernetesVersionReady,
    # StorageClassReady), gleiche Sortierung wie alphabetisch
```

Bei einem `IntervalUnparseable`-Warning kommt zusätzlich eine
`ConfigurationInvalid`-Zeile dazu (alphabetisch zwischen
`CertManagerInstalled` und `KubernetesVersionReady`); Summary-Counts
bleiben unverändert; Phase wechselt auf `Warning`.

Echten `kubectl get opdc <name> -o yaml`-Output erhält man am
schnellsten via Cluster-Smoke-Attest:

```bash
cat .cluster-smoke-attest.yaml   # nach erfolgreichem `make cluster-smoke`
```

---

## 12. Weiterführend

- [`installation.md`](installation.md) — Operator-Setup, RBAC,
  Metrics-Service.
- [`cr-examples.md`](cr-examples.md) — zwei vollständige CR-Beispiele
  mit Profile-Default-Erklärung und `spec.interval`-Verhalten.
- `troubleshooting.md` *(folgt mit M6 §4 Step 6)* — Diagnose-Pfade
  pro typischem Fehlerbild.
- [Slice-M5](../plan/planning/done/slice-M5-rbac-self-check-robustness.md)
  — RBAC-Selbstprüfung und Robustheits-Reasons (§10 hier).
- [`spec/architecture.md` AR-014](../../spec/architecture.md) —
  Severity-Aggregation und Phase-Mapping.
