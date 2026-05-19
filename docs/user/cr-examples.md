# CR-Beispiele: `evaluation` und `production`

> **Adressat:** Cluster-Betreiber, die eine
> `OpenDeskPreflightCheck`-Ressource gegen ihren Cluster konfigurieren.
>
> **Stand:** MVP v0.1. Zwei Profile sind aktiv (`evaluation`,
> `production`); `custom` und `k3s`/`scs`/`airgapped` folgen mit
> v0.2 ([`LH-PROF-001`/`-004`](../../spec/lastenheft.md)).

---

## 1. Was die Profile entscheiden

Das Feld `spec.profile` selektiert pro CR ein vordefiniertes Set von
Schwellwert-Defaults. Anwender-Werte in `spec.checks.*`
überschreiben die Defaults feldweise — wer in einem `production`-CR
`clusterResources.minCPU: "2"` setzt, bekommt den Anwender-Wert,
nicht den 4-CPU-Default.

| Default-Wert | `evaluation` | `production` |
| ------------ | ------------ | ------------ |
| `kubernetesVersion.min` | `1.34` (Operator-Floor) | `1.34` (Operator-Floor) |
| `clusterResources.minCPU` | `2` | `4` |
| `clusterResources.minMemory` | `4Gi` | `8Gi` |
| `storageClass.names` | leer (nur aktiv bei explizitem Sub-Spec) | leer (nur aktiv bei explizitem Sub-Spec) |
| `ingressClass.names` | leer (nur aktiv bei explizitem Sub-Spec) | leer (nur aktiv bei explizitem Sub-Spec) |
| `certManager` | aktiv (parameterlos) | aktiv (parameterlos) |
| `spec.interval` | `5m` (CRD-Default, identisch) | `5m` (CRD-Default, identisch) |

**Beobachtung:** Die K8s-Version-Defaults sind identisch — die
Differenzierung läuft nicht über die K8s-Version, sondern über die
Ressourcen-Schwellen
([LH-PROF-003](../../spec/lastenheft.md)). Die OpenDesk-Doku-
Untergrenze `≥ v1.24` ist niedriger als der Operator-Floor `1.34` —
eine OpenDesk-Installation auf `1.24` wäre upstream-supported, aber
dieser Operator startet darauf nicht (und schreibt
`KubernetesVersionTooOld` in den Status, falls man es trotzdem
versucht). Hintergrund in
[ADR 0009 §2.3](../plan/adr/0009-k8s-versions-support-und-profile-mindestversionen.md).

---

## 2. Beispiel `evaluation`

Geeignet für Test-/Demo-Cluster (1-3 Nodes, einzelne
Storage-Klasse, keine Hochlast-Erwartung).

```yaml
apiVersion: k-deskflight.geo-terrain.net/v1alpha1
kind: OpenDeskPreflightCheck
metadata:
  name: eval
  namespace: default
spec:
  profile: evaluation
  # Längerer Reconcile-Takt — Test-Cluster braucht keine Minutentaktung.
  # Bounds [30s, 24h] werden vom Operator-Normalisierer durchgesetzt;
  # ungültige Werte landen als ConfigurationInvalid-Condition im Status,
  # nicht als kubectl-apply-Fehler (architecture.md AR-010.1).
  interval: "30m"
  checks:
    kubernetesVersion:
      min: "1.34"
    storageClass:
      names:
        - standard
      requireDefault: false
    ingressClass:
      names:
        - nginx
    certManager: {}
    # Profile-Default greift (2 CPU / 4Gi) — Felder weggelassen.
```

**Was der Operator daraus macht:**

- Phase = `Passed`, wenn alle fünf Checks True liefern.
- Conditions: `KubernetesVersionReady`, `StorageClassReady`,
  `IngressClassReady`, `CertManagerInstalled`,
  `ClusterResourcesReady` (alle alphabetisch sortiert).
- Summary: `passed: 5, failed: 0, warning: 0, unknown: 0,
  checksTotal: 5`.
- Nächster Reconcile in 30 Minuten.

---

## 3. Beispiel `production`

Geeignet für produktive Cluster mit hochverfügbarem Storage, Ingress
und cert-manager.

```yaml
apiVersion: k-deskflight.geo-terrain.net/v1alpha1
kind: OpenDeskPreflightCheck
metadata:
  name: prod
  namespace: default
spec:
  profile: production
  # 5m ist der Default — explizit gesetzt, damit Anwender im YAML sehen,
  # welches Intervall greift.
  interval: "5m"
  checks:
    kubernetesVersion:
      min: "1.34"
    storageClass:
      names:
        - fast-rwx
        - bulk-rwo
      requireDefault: true   # ein Default-StorageClass muss markiert sein
    ingressClass:
      names:
        - nginx
    certManager: {}
    # clusterResources weggelassen → Profile-Default greift
    # (4 CPU / 8Gi, siehe Tabelle in §1). Wer eine konkrete
    # Cluster-Kapazität fordern will, setzt explizit:
    #   clusterResources:
    #     minCPU: "16"        # Beispielwert — an Cluster-Kapazität anpassen
    #     minMemory: "64Gi"   # Beispielwert — an Cluster-Kapazität anpassen
    # Hinweis: zu hohe Werte führen sofort zu InsufficientResources/-CPU/
    # -Memory; Test-Cluster (kind, 1-3 Nodes) erfüllen 16/64Gi typisch
    # nicht.
```

**Was der Operator daraus macht:**

- Phase = `Passed`, wenn alle fünf Checks True liefern.
  - StorageClass passed nur, wenn **beide** Namen (`fast-rwx`,
    `bulk-rwo`) existieren **und** mindestens eine
    StorageClass cluster-weit als Default markiert ist.
  - ClusterResources passed nur, wenn die summierten Ready-Node-
    Allocatables ≥ Profile-Default-Werten sind (Production: 4 CPU /
    8Gi). Wer das überschreibt (Block-Kommentar oben), zieht die
    Mindestwerte entsprechend.
- Phase = `Failed` bei mindestens einem False/critical-Result —
  Default-Severity für die fünf MVP-Check-Failures ist `critical`,
  außer cert-manager (`warning`, weil das Vorhandensein in v0.1
  nicht hart blockiert; Begründung in
  [slice-M4 §9](../plan/planning/done/slice-M4-cluster-state-checks.md)).
- Conditions: identische fünf Types wie im `evaluation`-Beispiel,
  mit ihrem jeweiligen Reason-Code.
- Nächster Reconcile in 5 Minuten.

---

## 4. Was sich zwischen den Beispielen unterscheidet

| Aspekt | `eval` (§2) | `prod` (§3) |
| ------ | ----------- | ----------- |
| `spec.profile` | `evaluation` | `production` |
| `spec.interval` | `30m` (großzügig) | `5m` (eng) |
| `storageClass.requireDefault` | `false` | `true` |
| `storageClass.names` | 1 Eintrag | 2 Einträge |
| `clusterResources.minCPU` | Default `2` (nicht im YAML) | Default `4` (nicht im YAML; Override-Beispiel im Block-Kommentar) |
| `clusterResources.minMemory` | Default `4Gi` (nicht im YAML) | Default `8Gi` (nicht im YAML; Override-Beispiel im Block-Kommentar) |

**Lese-Hilfe für YAML-Reviewer:** Felder, die im YAML **fehlen**,
sind nicht „leer", sondern signalisieren „Default greift".
`clusterResources: {}` würde im `eval`-Beispiel zur gleichen
Auswertung führen wie das hier verwendete Weglassen — beide Pfade
aktivieren den Profile-Default-Pfad (slice-M4 §2.3 in
[`docs/plan/planning/done/slice-M4-cluster-state-checks.md`](../plan/planning/done/slice-M4-cluster-state-checks.md)).

---

## 5. `spec.interval`-Verhalten

`spec.interval` steuert das Wiederholintervall (`RequeueAfter` am
Reconcile-Ende). Default `5m`, Bounds `[30s, 24h]`. Auch im
„CR ist schon einmal `Passed`"-Pfad wird das Intervall benutzt — der
Operator pollt nicht endlos.

**Klassifikations-Regel** (siehe
[architecture.md AR-010.1](../../spec/architecture.md) und
[Slice-M6 §2.3.1](../plan/planning/in-progress/slice-M6-metrics-tests-doku.md)):

| Eingabe | Ergebnis | Status-Effekt |
| ------- | -------- | ------------- |
| nicht gesetzt / `""` | Default `5m` | keine zusätzliche Condition |
| `"30s"` … `"24h"` (gültig, in Bounds) | Wert unverändert | keine zusätzliche Condition |
| `"15s"` (gültig, < `MinInterval`) | clamp auf `30s` | `ConfigurationInvalid=True`, Reason `IntervalClampedMin`, Severity `warning` |
| `"25h"` (gültig, > `MaxInterval`) | clamp auf `24h` | `ConfigurationInvalid=True`, Reason `IntervalClampedMax`, Severity `warning` |
| `"abc"` (ungültiges Format) | Default `5m` | `ConfigurationInvalid=True`, Reason `IntervalUnparseable`, Severity `warning` |

**Wichtig:** Phase wird durch eine Interval-Warning **maximal** auf
`Warning` gehoben — eine Failed-Phase aus einem Check-Failure bleibt
Failed (siehe [Slice-M6 §2.3 `mergeIntervalWarning`](../plan/planning/in-progress/slice-M6-metrics-tests-doku.md)).
Das `Summary`-Zählwerk zählt die Warning-Condition **nicht** mit —
sie ist CR-Spec-Scope, kein Check-Result.

---

## 6. Weiterführend

- [`installation.md`](installation.md) — Operator-Setup und
  Namespace-/Image-Override.
- `conditions.md` *(folgt mit M6 §4 Step 5)* — Reason-Codes pro
  Condition + Anwender-Action.
- `troubleshooting.md` *(folgt mit M6 §4 Step 6)* — typische
  Fehlerbilder.
- [Lastenheft `LH-F-025`](../../spec/lastenheft.md) — Wiederholintervall.
- [Lastenheft `LH-PROF-002`/`-003`](../../spec/lastenheft.md) —
  Profile-Definitionen.
