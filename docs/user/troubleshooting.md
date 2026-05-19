# Troubleshooting

> **Adressat:** Cluster-Betreiber, die den Operator deployen und mit
> unerwartetem Verhalten (CrashLoop, `Unknown`-Conditions,
> `/metrics`-Probleme) konfrontiert sind.
>
> **Stand:** MVP v0.1. Acht typische Fehlerbilder + ein
> v0.2-Vorgriff. Pro Eintrag: Symptom → Diagnose-Kommando →
> Lösungs-Schritt. Diagnose-Pfade sind so geschrieben, dass sie
> in einer Standard-`kubectl`-Umgebung funktionieren.

---

## 1. Operator-Pod startet nicht / CrashLoopBackOff

**Symptom:**

```
kubectl -n k-deskflight-system get pods
NAME                                    READY   STATUS             RESTARTS   AGE
k-deskflight-operator-5f5749f6c8-xxxxx  0/1     CrashLoopBackOff   3          2m
```

**Diagnose:**

```bash
kubectl -n k-deskflight-system logs deployment/k-deskflight-operator --tail=50
kubectl -n k-deskflight-system describe pod -l app.kubernetes.io/component=operator
```

Suche nach `forbidden: User "system:serviceaccount:k-deskflight-system:k-deskflight-operator"`
oder `failed to create manager: failed to discover server version` —
beide Symptome zeigen fehlende RBAC am Operator-ServiceAccount.

**Lösungs-Schritt:**

- Prüfe, dass die ClusterRoleBinding aus
  [`deploy/manifests/clusterrolebinding.yaml`](../../deploy/manifests/clusterrolebinding.yaml)
  appliziert ist:
  ```bash
  kubectl get clusterrolebinding k-deskflight-operator -o yaml
  ```
- Falls der `subjects[0].namespace` nicht zum tatsächlichen Operator-
  Namespace passt (z. B. nach Namespace-Override gemäß
  [installation.md §4](installation.md#4-namespace-override-ar-op-005)),
  Override-Patch fixen.
- Re-Apply: `kubectl apply -k deploy/manifests/`.

---

## 2. Alle Checks `Unknown` mit Reason `RBACInsufficient`

**Symptom:** CR-Status zeigt fünf `Unknown`-Conditions, jede mit
`reason: RBACInsufficient`. Phase = `Unknown`.

```bash
kubectl describe opdc smoke | grep -A 2 'Reason:'
```

**Diagnose:**

Die ClusterRole `k-deskflight-operator-cluster` ist appliziert, aber
deckt nicht alle benötigten Verben ab — z. B. weil ein Anwender sie
manuell reduziert hat. Reconciler ruft `SelfSubjectAccessReview` pro
Check, kriegt `denied` zurück, und ersetzt das Check-Result durch
`Unknown/RBACInsufficient` (slice-M5 §2.3). **Unterschied zum
Reason `RBACCheckFailed`:** dort ist nicht die Permission das
Problem, sondern das SAR-Subsystem selbst (`authorization.k8s.io`-
API nicht erreichbar).

Reproduzieren als Diagnose-Schritt:

```bash
# Manuelle Verifikation pro Check-Ressource (als Operator-SA):
kubectl auth can-i list storageclasses.storage.k8s.io \
  --as=system:serviceaccount:k-deskflight-system:k-deskflight-operator
kubectl auth can-i list ingressclasses.networking.k8s.io \
  --as=system:serviceaccount:k-deskflight-system:k-deskflight-operator
kubectl auth can-i list nodes \
  --as=system:serviceaccount:k-deskflight-system:k-deskflight-operator
```

Erwartet: alle drei liefern `yes`. Wenn `no`, dann fehlt die jeweilige
Rule.

**Lösungs-Schritt:**

- Kompletten Manifest-Set re-applizieren (deckt die ClusterRole aus
  `config/rbac/role.yaml` mit ab, gleicher Pfad wie in
  [installation.md §2](installation.md#2-default-installation)):
  ```bash
  kubectl apply -k deploy/manifests/
  ```
- Falls eine Org-Policy Cluster-weite ClusterRoles einschränkt,
  reicht eine Namespace-scoped Role nicht — der Operator braucht
  Cluster-Reads (Nodes, StorageClasses, IngressClasses sind
  cluster-scoped). LH-AK-015 + die operativ-Whitelist im
  `rbac_consistency_test.go` (slice-M5 §10.6) dokumentieren das
  Minimum.

**Manueller Reproduktions-Pfad** (slice-M5 §10.3-Übergabe an M6):
Wer den `Unknown/RBACInsufficient`-Pfad gezielt reproduzieren will,
kann temporär die ClusterRoleBinding aufheben — **und braucht
gleichzeitig eine Spec-Mutation**, weil der Reconciler bei
terminaler Phase ohne Spec-Diff skipt (`isAlreadyReconciled` in
[`internal/hexagon/application/reconciler.go`](../../internal/hexagon/application/reconciler.go)).

```bash
kubectl delete clusterrolebinding k-deskflight-operator
# Spec-Patch bumpt metadata.generation und triggert einen Reconcile.
# Eine harmlose interval-Änderung reicht; Annotation-Bump allein
# greift nicht, weil Annotationen die generation nicht ändern.
kubectl patch opdc smoke --type=merge -p '{"spec":{"interval":"6m"}}'
kubectl describe opdc smoke
# Erwartet: Phase=Unknown, Conditions mit Reason=RBACInsufficient.
# Anschließend RBAC wieder herstellen — Re-Apply triggert keinen
# weiteren Reconcile auf der gleichen Generation; entweder eine
# zweite Spec-Mutation oder RequeueAfter-Intervall (≤ 6m) abwarten.
kubectl apply -f deploy/manifests/clusterrolebinding.yaml
```

---

## 3. cert-manager-Check `warning` mit Reason `CertManagerMissing`

**Symptom:** Status-Phase = `Warning`; eine Condition zeigt
`CertManagerInstalled=False`, `reason: CertManagerMissing`,
`severity: warning`. Andere Conditions bleiben `True`.

```bash
kubectl get opdc smoke -o jsonpath='{.status.conditions[?(@.type=="CertManagerInstalled")]}'
```

**Diagnose:**

Die API-Gruppe `cert-manager.io` ist im Cluster nicht registriert.
**Nicht** critical: für viele OpenDesk-Deployments ist cert-manager
optional (eigene TLS-Termination per Ingress-Controller oder externer
Cert-Issuer).

Diagnose:

```bash
kubectl api-resources --api-group=cert-manager.io
```

Leer? Dann ist cert-manager wirklich nicht installiert.

**Lösungs-Schritt:**

- **Wenn cert-manager gewünscht ist:**
  [Upstream-Installation](https://cert-manager.io/docs/installation/)
  durchführen; nach erfolgreichem Deploy schreibt der nächste
  Reconcile-Lauf des Operators die Condition auf `True` um.
- **Wenn cert-manager bewusst weggelassen wird:** Den Warning-Status
  als erwartet akzeptieren. ClusterIssuer-Detail-Validierung
  ([LH-F-014](../../spec/lastenheft.md)) kommt mit v0.2 — dort gibt
  es einen Konfigurations-Schalter, um den cert-manager-Check
  optional zu machen.

---

## 4. `/metrics` connection refused (Service-Pfad)

**Symptom:** Ein Prometheus-Scrape gegen
`http://k-deskflight-operator-metrics.k-deskflight-system.svc:8080/metrics`
liefert `connection refused`.

**Diagnose:**

```bash
# Existiert der Service?
kubectl -n k-deskflight-system get svc k-deskflight-operator-metrics

# Matched der Selector mindestens einen Pod?
kubectl -n k-deskflight-system get endpoints k-deskflight-operator-metrics
# Erwartet: ADDRESSES-Spalte zeigt mindestens eine Pod-IP

# Ist der Operator-Pod Ready?
kubectl -n k-deskflight-system get pods -l app.kubernetes.io/component=operator
```

Wenn `ENDPOINTS` leer: Service-Selector matched keinen Pod.

**Lösungs-Schritt:**

- Falls der Operator-Pod nicht Ready ist → siehe §1 (CrashLoopBackOff)
  oder §6 (Manager-Init-Race).
- Falls der Selector nicht matched (z. B. nach Pod-Label-Customization):
  Selector im Service-Manifest mit den Pod-Labels abgleichen
  ([`deploy/manifests/service.yaml`](../../deploy/manifests/service.yaml)
  + [`deploy/manifests/deployment.yaml`](../../deploy/manifests/deployment.yaml)).
  Default-Labels: `app.kubernetes.io/name: k-deskflight`,
  `app.kubernetes.io/component: operator`.

---

## 5. `/metrics` DNS-Auflösung schlägt fehl

**Symptom:** Scrape-Pod loggt `getaddrinfo failed` oder
`temporary failure in name resolution` für den Service-FQDN.

**Diagnose:**

```bash
# Aus einem Pod im selben/anderen Namespace:
kubectl run dns-test --rm -it --image=busybox:1.36 -- \
  nslookup k-deskflight-operator-metrics.k-deskflight-system.svc.cluster.local
# Erwartet: Adresse aus dem Service-CIDR, kein NXDOMAIN
```

**Lösungs-Schritt:**

- **CoreDNS-Pod-Status prüfen:**
  ```bash
  kubectl -n kube-system get pods -l k8s-app=kube-dns
  ```
  Pods sollten alle `Running` sein.
- **FQDN-Suffix prüfen:** Default-Cluster-Domain ist
  `cluster.local`. Custom-Cluster (k3s, scs) können das auf
  `cluster.cluster.local` o. ä. setzen — der Scrape-FQDN muss dann
  entsprechend angepasst werden.
- **Cross-Namespace-Suchpfade:** aus dem `default`-Namespace
  funktioniert sowohl `…svc:8080` (Kurzform via search-path) als
  auch der volle FQDN. Aus fremden Namespaces ohne search-path muss
  der volle FQDN benutzt werden.

---

## 6. `/metrics` HTTP 200 mit leerem Body (Manager-Init-Race)

**Symptom:** Scrape kriegt `200 OK` zurück, aber Body ist leer oder
nur ein paar Header-Zeilen (`# HELP go_goroutines …`) ohne weitere
Metriken.

**Diagnose:**

Der controller-runtime-`Healthz`-Endpoint auf Port 8081 antwortet
schon, bevor der Manager intern alle Metriken registriert hat —
liveness-Probe ist grün, aber `/metrics` ist noch nicht voll
populated. Race-Fenster ist typisch unter 1 Sekunde nach Pod-Start;
in Test-Setups (kind, ressourcen-knappe Nodes) länger.

> **Hinweis:** Der Operator-Container nutzt
> `gcr.io/distroless/static-debian12:nonroot` (`Dockerfile`-runtime-
> Stage) — es ist weder `wget` noch eine Shell verfügbar.
> `kubectl exec` direkt im Operator-Pod liefert „executable not
> found". Diagnose-Pfade müssen das Cluster-API oder einen
> separaten Side-Pod nutzen.

Empfohlene Diagnose über die Cluster-API (kein Side-Pod nötig):

```bash
# Body via kubectl-proxy beziehen, nicht-leere Zeilen zählen
kubectl -n k-deskflight-system get --raw \
    '/api/v1/namespaces/k-deskflight-system/services/k-deskflight-operator-metrics:8080/proxy/metrics' \
    | grep -vc '^$'
# Erwartet: > 80 (controller-runtime + go_* + process_* + workqueue_*)
```

Alternativ über einen einmaligen curl-Pod (z. B. wenn der eigene
RBAC-Stand kein `services/proxy` erlaubt):

```bash
kubectl run scrape-probe --rm -it --restart=Never \
    --image=curlimages/curl:8.10.1 -- \
    curl -s http://k-deskflight-operator-metrics.k-deskflight-system.svc:8080/metrics \
    | grep -vc '^$'
```

`scripts/operator-http-smoke.sh` schützt sich mit einer
≥ 20-Zeilen-Sanity-Linie + einer Inhalts-Probe auf
`go_goroutines` / `process_cpu_seconds_total` (slice-M6 §2.2.2).
Externe Scraper sollten ähnlich defensiv sein.

**Lösungs-Schritt:**

- Scrape-Intervall geringfügig nach Pod-Restart verschieben (z. B.
  durch Prometheus-Operator `scrapeTimeout`-Konfiguration).
- Wenn das Problem persistiert: Operator-Pod-Logs auf
  Manager-Init-Fehler prüfen (`manager starting — blocking on signal
  handler` muss kommen). Bei einem hängenden Init bleibt `/metrics`
  dauerhaft leer — Bug-Report öffnen.

---

## 7. NetworkPolicy blockt Scrape

**Symptom:** Aus einem Monitoring-Pod kommt `connection refused`
oder Timeout, obwohl §4 alle grün ist (Service vorhanden, Endpoints
gefüllt, Operator-Pod Ready).

**Diagnose:**

```bash
# Welche NetworkPolicies sind im Operator-Namespace aktiv?
kubectl -n k-deskflight-system get networkpolicy
# Welche treffen auf den Operator-Pod?
kubectl -n k-deskflight-system describe networkpolicy
```

Wenn eine `default-deny`-Policy im `k-deskflight-system`-Namespace
aktiv ist, blockt sie Cross-Namespace-Ingress auf Port 8080.

**Lösungs-Schritt:**

Eine `NetworkPolicy` ergänzen, die Scrape aus dem Monitoring-Namespace
explizit erlaubt:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-metrics-scrape
  namespace: k-deskflight-system
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/component: operator
  policyTypes:
    - Ingress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: monitoring
      ports:
        - port: 8080
          protocol: TCP
```

NetworkPolicy ist **Anwender-Pflicht** (siehe
[installation.md §5](installation.md#5-prometheus-scrape-binding-pattern-asset)) —
der MVP liefert keine NetworkPolicy mit, weil die Stack-spezifischen
Labels (`monitoring`/`observability`/`prometheus`/…) nicht vorab
bekannt sind.

> **Hinweis zum `kubernetes.io/metadata.name`-Label:** dieses Label
> wird seit Kubernetes 1.22 automatisch auf jeden Namespace gesetzt
> (`NamespaceDefaultLabelName`-Feature). Der Operator-Floor ist 1.34
> ([ADR 0009 §2.2](../plan/adr/0009-k8s-versions-support-und-profile-mindestversionen.md)),
> die NetworkPolicy oben ist also kompatibel mit jeder vom Operator
> unterstützten Cluster-Version. Für Cluster mit manueller Label-
> Konvention statt dessen `name: monitoring` o. ä. verwenden.

---

## 8. `ConfigurationInvalid` mit `IntervalUnparseable` / `IntervalClampedMin` / `IntervalClampedMax`

**Symptom:** Phase = `Warning` (statt `Passed`); eine zusätzliche
Condition `ConfigurationInvalid=True` mit einem der drei
`IntervalXxx`-Reasons.

**Diagnose:**

```bash
kubectl get opdc smoke -o jsonpath='{.status.conditions[?(@.type=="ConfigurationInvalid")]}{"\n"}'
# Beispiel:
# {"type":"ConfigurationInvalid","status":"True","reason":"IntervalUnparseable",
#  "message":"spec.interval \"abc\" is not a valid Go duration ...; falling back to default 5m0s",...}
```

Der `spec.interval`-Wert wurde vom Reconciler normalisiert; der
Operator läuft mit dem normalisierten Wert weiter (kein Lifeness-
Bruch — AR-010.1).

**Lösungs-Schritt:**

`spec.interval` auf einen erlaubten Wert korrigieren. Format:
[`time.ParseDuration`](https://pkg.go.dev/time#ParseDuration) —
gültig sind z. B. `30s`, `5m`, `1h30m`, `24h`. Bounds: `[30s, 24h]`.

Detail-Mapping aus dem Reason:

| Reason | Korrekte Aktion |
| ------ | --------------- |
| `IntervalUnparseable` | Format auf `time.ParseDuration`-konformen String korrigieren — Beispiele: `30s`, `5m`, `1h30m`. |
| `IntervalClampedMin` | Wert auf ≥ `30s` heben oder Clamp-Verhalten akzeptieren. |
| `IntervalClampedMax` | Wert auf ≤ `24h` senken oder Clamp-Verhalten akzeptieren. |

Siehe [conditions.md §7](conditions.md#7-conditiontype-configurationinvalid-cr-spec-scope)
und [cr-examples.md §5](cr-examples.md#5-specinterval-verhalten).

---

## 9. (v0.2-Vorgriff) `/metrics` HTTP 401

> **Dieser Eintrag tritt in MVP v0.1 NICHT auf.** Der Operator
> liefert den `/metrics`-Endpoint unauthentisiert (controller-runtime-
> Default; siehe
> [installation.md §5](installation.md#5-prometheus-scrape-binding-pattern-asset)
> und [Slice-M6 §2.2.1](../plan/planning/in-progress/slice-M6-metrics-tests-doku.md)).
> Der Eintrag ist Vorgriff für die v0.2-Auth-Filter-Aktivierung
> (eigene ADR-Folge zu
> [ADR 0007](../plan/adr/0007-prometheus-metrik-scope-im-mvp.md)).

**Symptom (v0.2+):** Scrape liefert HTTP 401 zurück.

**Diagnose:**

```bash
# /metrics ist eine nonResourceURL — `kubectl auth can-i` mit dem
# Pfad-Argument, nicht mit --subresource:
kubectl auth can-i get /metrics \
  --as=system:serviceaccount:monitoring:prometheus-k8s
# oder ein Token-direkter Aufruf:
curl -H "Authorization: Bearer $(kubectl create token prometheus-k8s -n monitoring)" \
  https://k-deskflight-operator-metrics.k-deskflight-system.svc:8080/metrics
```

**Lösungs-Schritt (v0.2+):** ClusterRoleBinding für den
Prometheus-/Monitoring-ServiceAccount auf die mitgelieferte
`k-deskflight-metrics-scrape`-ClusterRole anlegen — Beispiel-Manifest
in [installation.md §5](installation.md#5-prometheus-scrape-binding-pattern-asset).
Die ClusterRole ist in v0.1 bereits ausgeliefert; sie wird mit der
Auth-Filter-Aktivierung in v0.2 funktional wirksam.

---

## 10. Weiterführend

- [`installation.md`](installation.md) — Operator-Setup, RBAC,
  Metrics-Service.
- [`cr-examples.md`](cr-examples.md) — CR-Beispiele mit
  Profile-Default-Erklärung.
- [`conditions.md`](conditions.md) — Reason-Codes pro Condition.
- [Slice-M5](../plan/planning/done/slice-M5-rbac-self-check-robustness.md)
  — RBAC-Selbstprüfung und Per-Check-Recover-Pfade.
- [ADR 0007](../plan/adr/0007-prometheus-metrik-scope-im-mvp.md) —
  Metrik-Scope und Auth-Filter-Übergabe an v0.2.
