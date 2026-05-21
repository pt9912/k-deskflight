# Installation

> **Adressat:** Cluster-Betreiber, die den k-deskflight-Operator in
> einem Kubernetes-Cluster ausrollen wollen.
>
> **Stand:** v0.1 (MVP) ausgeliefert; v0.2 in Arbeit. Default-
> Distribution sind die raw Kubernetes-Manifeste unter
> [`deploy/manifests/`](../../deploy/manifests/) (§2–§7). Ab v0.2
> steht zusätzlich ein Helm-Chart unter
> [`deploy/charts/k-deskflight/`](../../deploy/charts/k-deskflight/)
> bereit (§8); beide Pfade rendern dasselbe Funktionsset und werden
> über die Cluster-Smoke-Matrix (`install-mode: [manifests, helm]`)
> gemeinsam attestiert.

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

## 6. Beispiel-CR applizieren

Mit installiertem Operator legst du eine `OpenDeskPreflightCheck`-CR
an, die der Operator picked up und reconciled. Zwei vorgefertigte
Vorlagen sind unter [`deploy/samples/`](../../deploy/samples/)
ausgeliefert:

- [`cluster-readiness-production.yaml`](../../deploy/samples/cluster-readiness-production.yaml)
  — Profil `production`, alle fünf MVP-Checks aktiv. Schwellen: 16 CPU / 64 Gi
  Ressourcen, `default`+`backup` StorageClass mit Default-Anforderung,
  `nginx` IngressClass, cert-manager-Vorhandensein.
- [`cluster-readiness-evaluation.yaml`](../../deploy/samples/cluster-readiness-evaluation.yaml)
  — Profil `evaluation`, vier Checks (kein `ingressClass`-Sub-Spec).
  Schwellen: 2 CPU / 4 Gi, `standard` StorageClass ohne Default-Anforderung.

**Beide Vorlagen sind anwendungsorientiert** — die Schwellen sind
gegen reale Cluster konfiguriert. Auf einem Test-Cluster (kind, k3d
mit Default-Profil, etc.) liefert besonders die Production-Vorlage
oft `Failed`-Conditions; das ist beabsichtigt und kein Defekt der
Vorlage. Für lokale Test-Apply-Pfade siehe
[cr-examples.md](cr-examples.md), das pädagogische CR-Beispiele mit
schmaleren Anforderungen führt.

Apply der Production-Vorlage:

```bash
kubectl apply -f deploy/samples/cluster-readiness-production.yaml
```

Status verfolgen:

```bash
kubectl get opendeskpreflightcheck cluster-readiness-production -o yaml
```

Auf eine terminale Phase warten (`Passed` / `Failed` / `Warning` /
`Unknown`). Auf einem Cluster, der die Production-Schwellen erfüllt,
fängt `kubectl wait` das `Passed` ab; auf einem Test-Cluster läuft
es ins Timeout und der Fallback zeigt die tatsächliche Phase samt
Summary an:

```bash
kubectl wait --for=jsonpath='{.status.phase}'=Passed \
    opendeskpreflightcheck/cluster-readiness-production --timeout=120s \
  || kubectl get opendeskpreflightcheck cluster-readiness-production \
       -o jsonpath='{"phase="}{.status.phase}{" summary="}{.status.summary}{"\n"}'
```

(`kubectl wait --for=jsonpath=` matched String-gleich, kann also
`Failed`/`Warning`/`Unknown` nicht direkt mitnehmen — daher der
`||`-Fallback.)

Anwender, die ohne Repo-Checkout arbeiten, applizieren die Vorlage
direkt aus der Repo-URL — z. B. via
`kubectl apply -f https://raw.githubusercontent.com/pt9912/k-deskflight/v0.1.0/deploy/samples/cluster-readiness-production.yaml`.
**Vor Tag-Setzung liefert diese URL 404**; ab dem `v0.1.0`-Tag
(Slice-M7-Closure) ist sie stabil pinned und stabil scrapebar.

Für CR-Lebenszyklus-Operationen (Re-Run, Edit, Delete) siehe
[§7 Wiederherstellung und Update](#7-wiederherstellung-und-update).

---

## 7. Wiederherstellung und Update

- **CR-Edit:** Änderungen am `spec`-Block einer
  `OpenDeskPreflightCheck`-CR bumpen `metadata.generation` und lösen
  einen erneuten Reconcile aus
  ([`LH-F-026`](../../spec/lastenheft.md)).
- **Re-Run ohne Spec-Änderung:** Ist der CR in einer terminalen Phase
  (`Passed`, `Warning`, `Failed`, `Unknown`), überspringt der
  Reconciler nachfolgende Events ohne Spec-Diff (`isAlreadyReconciled`-
  Skip in [`internal/hexagon/application/reconciler.go`](../../internal/hexagon/application/reconciler.go)).
  Ein **Annotation-Bump alleine reicht deshalb nicht**, weil
  Annotationen `metadata.generation` nicht bumpen.

  Drei zuverlässige Pfade für einen Re-Run vor Ablauf des
  Intervalls:
  ```bash
  # (a) Spec-Patch mit harmlosem Wechsel (z. B. Interval-Bump):
  kubectl patch opdc smoke --type=merge -p '{"spec":{"interval":"6m"}}'
  # (b) Delete + Re-Apply der CR:
  kubectl delete opdc smoke && kubectl apply -f config/samples/
  # (c) RequeueAfter abwarten (siehe nächster Punkt).
  ```
- **Periodischer Re-Run:** `spec.interval` steuert den
  RequeueAfter-Wert (Default 5m, Bounds 30s–24h). Der Reconciler
  setzt den Wert auch im Skip-Pfad, sodass ein CR in terminaler
  Phase spätestens nach dem Intervall erneut reconciled wird. Siehe
  [cr-examples.md §5](cr-examples.md#5-specinterval-verhalten) für
  das Klassifikations-Verhalten.
- **Deinstallation:**
  ```bash
  kubectl delete -k deploy/manifests/
  ```
  Die CRD-Löschung kaskadiert auf alle `OpenDeskPreflightCheck`-CRs
  cluster-weit.

---

## 8. Alternative: Installation via Helm-Chart

Ab v0.2 steht der Operator zusätzlich als Helm-Chart unter
[`deploy/charts/k-deskflight/`](../../deploy/charts/k-deskflight/)
bereit (`LH-NF-016`, `LH-SST-010`, [ADR 0005](../plan/adr/0005-helm-chart-nicht-im-mvp.md)).
Der Chart leitet seine Templates 1:1 aus `deploy/manifests/` ab und
deckt dieselben Resources wie die Kustomize-Default-Installation (§2)
ab; die Wahl zwischen den beiden Pfaden ist eine Distributions-, keine
Funktions-Entscheidung.

### 8.1 Voraussetzungen

Zusätzlich zu §1:

- Helm 3.x auf dem Client (Validierung gegen `kubeVersion: ">=1.34.0-0"`
  aus [`deploy/charts/k-deskflight/Chart.yaml`](../../deploy/charts/k-deskflight/Chart.yaml)).
- Cluster-Admin-Rechte (CRD-Install, ClusterRole/ClusterRoleBinding).

### 8.2 Default-Installation

Zwei Install-Wege:

- **Repository-Checkout** (heute funktionsfähig, weiter verfügbar
  auch nach M16) — der unten gezeigte Default-Pfad.
- **OCI-Registry** (ab M16-Closure aufrufbar, [ADR 0015](../plan/adr/0015-helm-chart-distributions-form.md)) —
  weiter unten skizziert.

**Default — Install aus dem Repository-Checkout:**

```bash
helm install k-deskflight deploy/charts/k-deskflight/ \
    --namespace k-deskflight-system \
    --create-namespace \
    --set namespace.create=false
```

**OCI-Install (verfügbar ab v0.2.0 / M16-Closure):**

> Hinweis: Dieser Befehl liefert bis zum ersten erfolgreichen
> Chart-Publish in M16 + der Public-Schaltung des GHCR-Packages
> einen `unauthorized` / `not found`. Form und Argumente sind hier
> ergänzend dokumentiert, damit Anwender nach M16-Closure direkt
> umsteigen können.

```bash
helm install k-deskflight oci://ghcr.io/pt9912/charts/k-deskflight \
    --version 0.2.0 \
    --namespace k-deskflight-system \
    --create-namespace \
    --set namespace.create=false
```

`--version` wird explizit empfohlen. Ohne Flag fällt Helm auf das
im `Chart.yaml` veröffentlichte SemVer-Tag zurück (kein implizites
`latest` wie bei Container-Images); bei mehreren publizierten
Versionen führt der Verzicht aufs Pinning aber zu unbeabsichtigtem
Auto-Upgrade. Semver-Range-Auswahl (`--version "^0.2"`) ist
Helm-3-Standard. Helm 3.8+ ist Pflicht für OCI-Support.

**Hinweis zur Namespace-Mechanik:** das Chart enthält eine
Namespace-Resource (Default `namespace.create=true`), Helm verlangt
aber, dass der `--namespace`-Wert beim Install bereits existiert.
Das ergibt zwei tragfähige Operations-Patterns:

| Pattern | Konfiguration | Wer verwaltet den Namespace? |
| ------- | ------------- | ---------------------------- |
| **A** (oben gezeigt) | `--create-namespace` + `--set namespace.create=false` | Helm |
| **B** | Namespace vorab per `kubectl create ns k-deskflight-system`; `helm install` ohne `--create-namespace`, `namespace.create=false` | Anwender / GitOps |

Pattern A ist die kürzere CLI-Variante; Pattern B passt zu GitOps-
Setups, in denen der Operator-Namespace aus einem separaten
Cluster-Bootstrap-Layer kommt. Beide Patterns deaktivieren das
Chart-Namespace-Template, um die „rendered manifests contain a
resource that already exists"-Kollision zu vermeiden — Helm's
`--create-namespace`-Pre-Apply-Phase und ein chart-eigenes
Namespace-Template würden sonst beide versuchen, denselben
Namespace anzulegen.

Verifikation:

```bash
kubectl -n k-deskflight-system wait \
    --for=condition=Available deployment/k-deskflight-operator \
    --timeout=120s
kubectl get crd opendeskpreflightchecks.k-deskflight.geo-terrain.net
```

### 8.3 Image-Pin und Konfigurations-Overrides

Alle Konfigurations-Optionen via `values.yaml`-Overrides; die
vollständige Schema-Definition liegt in
[`deploy/charts/k-deskflight/values.yaml`](../../deploy/charts/k-deskflight/values.yaml)
mit JSON-Schema-Validierung in
[`values.schema.json`](../../deploy/charts/k-deskflight/values.schema.json).

Häufige Overrides:

```bash
# Pin auf konkreten Image-Tag:
helm install k-deskflight deploy/charts/k-deskflight/ \
    --namespace k-deskflight-system --create-namespace \
    --set namespace.create=false \
    --set image.tag=v0.1.0

# Replicas + Resources:
helm install … \
    --set operator.replicas=1 \
    --set operator.resources.requests.cpu=100m \
    --set operator.resources.limits.memory=512Mi

# Eigener Operator-Namespace:
helm install k-deskflight deploy/charts/k-deskflight/ \
    --namespace my-operators --create-namespace \
    --set namespace.create=false \
    --set namespace.name=my-operators
```

`values.yaml.namespace.name` und der `--namespace`-Flag-Wert
müssen übereinstimmen — das wird vom Chart nicht erzwungen
(`values.schema.json` enthält keinen Cross-Field-Constraint dafür),
sondern liegt in der Verantwortung des Anwenders. Bei Drift
zeigen Service-DNS-FQDN und ClusterRoleBinding-Subjects auf
einen anderen Namespace als der gerenderte Helm-Release-Scope.

### 8.4 Betriebsmodus: Cluster-Wide vs. Namespace-Scope

Der Chart belichtet das `AR-016`/`AR-017`-Betriebsmodus-Pattern via
`operator.mode`-Toggle (siehe
[`spec/architecture.md` AR-016](../../spec/architecture.md)):

```bash
# Cluster-Wide Mode (Default):
helm install k-deskflight deploy/charts/k-deskflight/ \
    --namespace k-deskflight-system --create-namespace \
    --set namespace.create=false
# → ClusterRole + ClusterRoleBinding, Operator reconciliert
#    OpenDeskPreflightCheck-CRs in allen Namespaces.

# Namespace-Reconcile-Scope Mode:
helm install k-deskflight deploy/charts/k-deskflight/ \
    --namespace k-deskflight-system --create-namespace \
    --set namespace.create=false \
    --set operator.mode=namespace-scope
# → zusätzlich Role + RoleBinding im Operator-Namespace;
#    Deployment bekommt `--namespace=k-deskflight-system`-Arg,
#    Operator reconciliert nur CRs im eigenen Namespace.
```

Die `values.schema.json`-Validierung schließt die Kombination
`operator.mode=namespace-scope` + `rbac.create=false` aus
(Schema-Error vor `helm install`), weil der Operator dann mit
`--namespace`-Arg ohne nötige Role/RoleBinding starten würde
(403-Fehler beim ersten Reconcile).

### 8.5 RBAC extern verwalten

Wer den Operator in eine Umgebung deployt, in der RBAC-Objekte
durch einen separaten Layer (z. B. Argo-CD ApplicationSets,
zentrale Policy-Engine, Compliance-Tooling) verwaltet werden,
schaltet die Chart-RBAC-Erzeugung ab:

```bash
helm install k-deskflight deploy/charts/k-deskflight/ \
    --namespace k-deskflight-system --create-namespace \
    --set namespace.create=false \
    --set rbac.create=false \
    --set serviceAccount.create=false \
    --set serviceAccount.name=k-deskflight-operator
```

In dieser Konfiguration muss der Anwender außerhalb des Charts
sicherstellen:

- Eine ClusterRole mit dem Verb-Set aus
  [`config/rbac/role.yaml`](../../config/rbac/role.yaml) (`AR-015`).
- Ein ClusterRoleBinding, der die ClusterRole an den vorhandenen
  ServiceAccount bindet.
- Im Namespace-Scope Mode zusätzlich Role + RoleBinding nach
  [`AR-016`/`AR-017`](../../spec/architecture.md).
- Der ServiceAccount muss im durch `serviceAccount.name`
  benannten Wert existieren.

Achtung — `operator.mode=namespace-scope` + `rbac.create=false`
ist von der Schema-Validierung blockiert (siehe §8.4): wer
`rbac.create=false` setzt, muss daher entweder bei
`operator.mode=cluster-wide` (Default) bleiben, oder die
Namespace-Scope-RBAC-Objekte ebenfalls extern bereitstellen und
den Mode-Constraint via `--set operator.mode=cluster-wide`
explizit halten. Das Schema setzt keinen Mode implizit — es
verwirft die ungültige Kombination.

### 8.6 Upgrade und Uninstall

```bash
# Upgrade auf neuen Operator-Tag oder Chart-Version:
helm upgrade k-deskflight deploy/charts/k-deskflight/ \
    --namespace k-deskflight-system \
    --set namespace.create=false \
    --set image.tag=v0.2.0

# Uninstall:
helm uninstall k-deskflight --namespace k-deskflight-system
```

**Achtung:** Wenn beim Install `crd.install=true` war (Default),
löscht `helm uninstall` **auch** die CRD und damit alle
`OpenDeskPreflightCheck`-CRs cluster-weit — das Chart rendert die
CRD als reguläres Template und der Helm-Release verwaltet sie.
`kubectl get opendeskpreflightcheck -A -o yaml > backup.yaml` vor
dem Uninstall sichert; CR-Verlust ist nicht reversibel ohne
manuellen Re-Apply der gesicherten YAMLs. Wer CRs zwischen
Releases überleben lassen will, installiert mit
`--set crd.install=false` und verwaltet die CRD separat
(GitOps-Pattern).

---

## 9. Weiterführend

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
