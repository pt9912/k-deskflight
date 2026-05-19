# Installation

> **Adressat:** Cluster-Betreiber, die den k-deskflight-Operator in
> einem Kubernetes-Cluster ausrollen wollen.
>
> **Stand:** MVP v0.1. Helm Chart ist
> [bewusst nicht im MVP](../plan/adr/0005-helm-chart-nicht-im-mvp.md);
> Distribution erfolgt über raw manifests.

---

## 1. Voraussetzungen

- Ein laufender Kubernetes-Cluster ≥ 1.34 (Operator-Floor laut
  [ADR 0009 §2.2](../plan/adr/0009-k8s-versions-support-und-profile-mindestversionen.md)).
- `kubectl` mit Cluster-Admin-äquivalenten Rechten für den
  Installations-Vorgang (CRD-Apply, ClusterRole, ClusterRoleBinding,
  Namespace-Anlage).
- Lokal verfügbar: das k-deskflight-Repository ausgecheckt — die
  Manifeste werden aus `deploy/manifests/` per `kubectl apply -k`
  appliziert.

Wer den Operator ohne Repo-Checkout installieren möchte, kann die
Kustomize-Base per Git-URL referenzieren — Beispiel-Pattern in §4.

---

## 2. Default-Installation

Die mitgelieferte Kustomize-Base in
[`deploy/manifests/`](../../deploy/manifests/) deployt:

- den Namespace `k-deskflight-system`,
- die CRD `opendeskpreflightchecks.k-deskflight.geo-terrain.net`,
- ServiceAccount, ClusterRole, ClusterRoleBinding für den Operator,
- das Operator-Deployment (single replica, runAsNonRoot,
  readOnlyRootFilesystem),
- den Metrics-Service `k-deskflight-operator-metrics:8080` und die
  Pattern-Asset-ClusterRole `k-deskflight-metrics-scrape` (siehe
  §5 für die Wirkung).

Default-Apply:

```bash
kubectl apply -k deploy/manifests/
```

Verifikation:

```bash
kubectl -n k-deskflight-system wait \
    --for=condition=Available deployment/k-deskflight-operator \
    --timeout=120s
kubectl -n k-deskflight-system get svc k-deskflight-operator-metrics
```

---

## 3. Operator-Image-Pin

Der `kustomization.yaml`-Default zeigt auf
`ghcr.io/pt9912/k-deskflight:dev` (Platzhalter für lokale Tests).
Für produktive Installationen die Image-Tag-Override-Mechanik
von Kustomize nutzen — in einem eigenen Overlay-Verzeichnis:

```yaml
# myoverlay/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../deploy/manifests
images:
  - name: ghcr.io/pt9912/k-deskflight
    newName: ghcr.io/pt9912/k-deskflight
    newTag: v0.1.0  # konkrete Release-Version (kommt mit M7)
```

```bash
kubectl apply -k myoverlay/
```

Der MVP veröffentlicht das Operator-Image als Teil des
`v0.1.0`-Releases (M7); bis dahin baut man es lokal mit
`make build` und lädt es per `kind load docker-image` oder eigenem
Registry-Push.

---

## 4. Namespace-Override (`AR-OP-005`)

Der Default-Namespace `k-deskflight-system` ist im
`deploy/manifests/namespace.yaml`-Stanza und in den Pod-/SA-Metadaten
verankert. Für eine andere Namespace-Wahl ist ein
Kustomize-Overlay der saubere Weg
(geschlossener Architektur-Punkt
[`AR-OP-005`](../../spec/architecture.md)):

```yaml
# myoverlay/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: my-operators  # alle namespaced Resources werden umgeschrieben
resources:
  - ../deploy/manifests
patches:
  - target:
      kind: Namespace
      name: k-deskflight-system
    patch: |-
      - op: replace
        path: /metadata/name
        value: my-operators
  - target:
      kind: ClusterRoleBinding
      name: k-deskflight-operator
    patch: |-
      - op: replace
        path: /subjects/0/namespace
        value: my-operators
```

```bash
kubectl apply -k myoverlay/
```

**Was dabei passiert:**

- `namespace: my-operators` in der `kustomization.yaml` ändert
  alle namespaced Manifeste auf den neuen Namespace.
- Die zwei Patches fixieren zwei Stellen, die Kustomize nicht
  automatisch mit-anpasst: die `Namespace`-Ressource selbst (sie
  hat keinen `metadata.namespace`-Eintrag) und der
  `ClusterRoleBinding.subjects[0].namespace`-Verweis auf die
  ServiceAccount.
- Cluster-scoped Resources (CRD, ClusterRole, ClusterRoleBinding)
  behalten ihre namen-globalen Identitäten.

Die Cluster-Smoke-Pipeline
([`scripts/cluster-smoke.sh`](../../scripts/cluster-smoke.sh) Step 4)
nutzt das gleiche Overlay-Pattern operativ für den Image-Override
ohne Namespace-Wechsel — der Code ist ein nützliches Referenz-Snippet.

---

## 5. Prometheus-Scrape-Binding (Pattern-Asset)

Die mitgelieferte ClusterRole `k-deskflight-metrics-scrape` hat
`nonResourceURLs: ["/metrics"]` mit `verbs: ["get"]`. **In v0.1 ist
sie funktional ohne Wirkung**, weil der Operator den
controller-runtime-Default-`/metrics`-Endpoint ohne Auth-Filter
liefert — jeder Pod im Cluster mit Netzwerk-Zugriff auf den Service
kann scrapen, auch ohne Token-Binding. Der Disclaimer steht im
Manifest-Kommentar
([`deploy/manifests/metrics-clusterrole.yaml`](../../deploy/manifests/metrics-clusterrole.yaml)).

**Trotzdem mitgeliefert**, weil:

- Die Rolle ist Pattern-Asset für künftige v0.2-Auth-Filter-
  Aktivierung (`kube-rbac-proxy`-Sidecar oder
  controller-runtime-FilterProvider; eigene ADR-Folge zu
  [ADR 0007](../plan/adr/0007-prometheus-metrik-scope-im-mvp.md)).
- Anwender, die bereits in v0.1 mit `kube-rbac-proxy` arbeiten und
  ihn vor den Operator-Pod schalten, sparen sich die ClusterRole-
  Definition.

**Beispiel-Binding für einen Prometheus-Operator-Stack**
(`namespace: monitoring`, ServiceAccount `prometheus-k8s`):

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: k-deskflight-metrics-scrape-prometheus
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: k-deskflight-metrics-scrape
subjects:
  - kind: ServiceAccount
    name: prometheus-k8s
    namespace: monitoring
```

Für andere Stacks (VictoriaMetrics, Grafana Agent, Vector) das
`subjects`-Feld auf den jeweiligen SA-Namen + Namespace anpassen.

**NetworkPolicy ist Anwender-Pflicht.** Wer Cross-Namespace-Scrape
einschränken will, sollte eine `NetworkPolicy` im
`k-deskflight-system`-Namespace ergänzen, die Ingress auf Port 8080
nur aus dem Monitoring-Namespace erlaubt.

---

## 6. Wiederherstellung und Update

- **CR-Edit:** Änderungen an einer `OpenDeskPreflightCheck`-CR
  lösen einen erneuten Reconcile aus
  ([`LH-F-026`](../../spec/lastenheft.md)).
- **Re-Run ohne Spec-Änderung:** eine Annotation-Bump (z. B.
  `kubectl annotate opdc smoke kdeskflight.geo-terrain.net/rerun=$(date +%s) --overwrite`)
  oder ein erneutes Apply ohne Änderung triggert einen neuen Reconcile.
- **Periodischer Re-Run:** `spec.interval` steuert den
  RequeueAfter-Wert (Default 5m, Bounds 30s–24h). Siehe
  [cr-examples.md](cr-examples.md) für Beispiele.
- **Deinstallation:**
  ```bash
  kubectl delete -k deploy/manifests/
  ```
  Die CRD-Löschung kaskadiert auf alle `OpenDeskPreflightCheck`-CRs
  cluster-weit.

---

## 7. Weiterführend

- [`cr-examples.md`](cr-examples.md) — zwei CR-Beispiele
  (`evaluation` + `production`) mit Profile-Default-Erklärung.
- `conditions.md` *(folgt mit M6 §4 Step 5)* — Reason/Severity pro
  Condition.
- `troubleshooting.md` *(folgt mit M6 §4 Step 6)* — typische
  Fehlerbilder.
- [ADR 0007](../plan/adr/0007-prometheus-metrik-scope-im-mvp.md) —
  Metrik-Scope und Auth-Filter-Übergabe an v0.2.
- [ADR 0009](../plan/adr/0009-k8s-versions-support-und-profile-mindestversionen.md)
  — Operator-Floor und Profile-Mindestversionen.
