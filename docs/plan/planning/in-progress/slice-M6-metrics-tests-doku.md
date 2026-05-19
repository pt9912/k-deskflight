# Slice M6 — Metrics-Endpoint, Tests, Doku

**Status:** In Progress
**Eröffnet:** 2026-05-19
**Vorgänger:** [M5 — RBAC-Selbstprüfung & Robustheit (Done)](../done/slice-M5-rbac-self-check-robustness.md)
**Nachfolger:** [M7 — Beispielmanifest, Release-Tag v0.1.0 (Pending)](../in-progress/roadmap.md#m7--beispielmanifest-release-tag-v010)
**Bezug:**
[Roadmap §3 M6](../in-progress/roadmap.md#m6--metrics-endpoint-tests-doku),
[`spec/architecture.md` §6 (AR-010, AR-010.1), §9 (AR-024, AR-025), §AR-027](../../../../spec/architecture.md),
[ADR 0007](../../adr/0007-prometheus-metrik-scope-im-mvp.md),
[ADR 0012 §2.4–2.6](../../adr/0012-quality-gates.md),
[ADR 0013](../../adr/0013-cluster-smoke-platform.md)

---

## 1. Lieferziel

`/metrics`-Endpoint scrapebar machen (Service + RBAC, `ADR 0007 §2.4`),
Integrationstests gegen `kind` (`make cluster-smoke`) erweitert um pro
MVP-Check je einen passed- und einen failed-Pfad (`LH-QG-002`, vom
Anwender bewusst statt `envtest` gewählt — §2.1), Anwender-Doku unter
`docs/user/` mit Installation, CR-Beispielen für `evaluation` und
`production`, Conditions-Katalog und Troubleshooting (`LH-AK-013`,
`LH-NF-013`, `LH-QA-004`), und vollständige Strikt-Schaltung der drei
M6-Pflicht-Gates (`LH-QG-003` Coverage 90 %, `LH-QG-004` Boundary,
`LH-QG-006` `govulncheck`).

**Roadmap-§3-M6-Bullets, die in diesem Slice fallen:**

- [ ] `/metrics`-Endpoint via `Service`-Objekt exposed und im
  Cluster-Smoke via Service-DNS reachable (`LH-SST-004`, `ADR 0007
  §2.1`, `§2.4`). **ServiceAccount-RBAC für Scrape als
  Pattern-Asset** (ClusterRole mitgeliefert, **kein** Binding):
  funktional in M6 ohne Wirkung, weil der Endpoint
  controller-runtime-Default-unauthentisiert läuft —
  **funktionale Auth-/RBAC-Absicherung ist v0.2** mit eigener
  ADR-Folge zu `ADR 0007` (siehe §2.2.1 + §8). Aktueller Stand:
  Endpoint läuft bereits auf `:8080`
  (`cmd/operator/main.go:43-44, 73-77`), `Service`-Wiring und
  ClusterRole-Pattern fehlen noch. §2.2.
- [ ] Integrationstests (`kind via cluster-smoke`): **vier** MVP-Checks
  bekommen je einen passed- und einen failed-Case
  (KubernetesVersion, StorageClass, IngressClass, ClusterResources —
  `LH-QG-002`, Re-Attest für `LH-AK-005`, `-006`, `-007`, `-009`).
  **cert-manager** (`LH-AK-008`) bekommt im automatischen Smoke
  **keinen** failed-Case — Begründung in §2.8 (`cluster-state-
  stubs.yaml` setzt den Stub global; ein per-CR-Ausschalten ist
  ohne M4-Smoke-Refactor nicht möglich). M4-Closure (Unit-Test 7
  Fälle + Cluster-Smoke passed) bleibt für `LH-AK-008` gültig.
  §2.1 + §2.8.
- [ ] Anwender-Doku in `docs/user/` ausarbeiten: Installation
  (raw manifests), CR-Beispiele evaluation+production,
  Conditions-Katalog mit Reason/Severity, Troubleshooting
  (`LH-AK-013`, `LH-NF-013`, `LH-QA-004`). §2.4.
- [ ] **Coverage-Gate strikt** auf 90 % über produktive
  Pakete (`LH-QG-003`); `make coverage-gate` blockt PRs. §2.5.
- [ ] **`govulncheck` strikt** als Pflicht-Gate (`LH-QG-006`);
  funktionsbasiertes Scanning gegen Go-Vulnerability-Datenbank. §2.6.
- [ ] **Architektur-Boundary strikt**: alle `depguard`-Regeln aus M2
  bleiben aktiv und brechen den Build bei Layer-Verletzung
  (`LH-QG-004`). §2.7.

**Was M6 als zusätzliche Pflicht aus M5 übernimmt
(M5 §10.3-Übergaben):**

- [ ] **`Spec.Interval`-Implementierung** (`AR-010`, `LH-F-025`) —
  zusammen mit Anwender-Doku, weil das Feld Anwender-Default-
  Erwartungen prägt und nicht ohne CR-Beispiel ausgeliefert werden
  soll. §2.3.
- [ ] **Cluster-Smoke-Failed-Szenario mit eingeschränkter RBAC**
  (Operator-ServiceAccount ohne `storage.k8s.io/storageclasses,
  verbs=list`) — M5 §10.3 hat das explizit auf M6-Manual-Test
  verschoben; in M6 ziehen wir es als Teil des Failed-Pfad-Smoke-Sets
  in §2.8 hinein.

**Was M6 noch nicht macht (Roadmap-§3-eng, Übergaben an M7/v0.2):**

- **Beispielmanifest-Vollständigkeit und v0.1.0-Tag** — gehören
  zu M7 (`LH-MVP-002`, `LH-AK-014`). M6 liefert die Doku-Sample-CRs
  unter `docs/user/`, M7 hebt sie auf `deploy/manifests/samples/` und
  zieht Release-Notes.
- **Leader-Election scharf** (`AR-026`) — M7-Release-Hardening.
  RBAC für Leases ist bereits in M2 verankert; `--leader-elect=true`
  bleibt MVP-Default, aber die Konfiguration via Flag/Env entsteht in
  M7. §8.
- **Eigene Domänen-Metriken** (`LH-NF-008` wörtlich) — v0.2
  (`ADR 0007 §2.3`). M6 deckt nur die controller-runtime-
  Framework-Defaults.
- **ServiceMonitor / PodMonitor / PrometheusRule** —
  Prometheus-Operator-Stack-Pattern; explizit out-of-scope in
  `ADR 0007 §4`. v0.2 zusammen mit Helm-Chart.
- **Domain-Metrik-Pre-Registrierung** (`kdeskflight_*`-Stub am
  controller-runtime-Registry) — v0.2; M6 dokumentiert in der
  Anwender-Doku nur, dass dieser Prefix für künftige eigene Metriken
  reserviert ist.

---

## 2. Slice-Entscheidungen

### 2.1 Integrationstest-Tooling: `kind via cluster-smoke` (nicht `envtest`)

**Entscheidung:** Die M6-Integrationstests laufen über den existierenden
`make cluster-smoke`-Pfad gegen `kind`, nicht über `envtest`.

**Begründung:**

- `LH-QG-002` listet beide Optionen wörtlich gleichwertig:
  „Integrationstests laufen gegen `envtest` oder einen `kind`-Cluster."
  Beide sind Lastenheft-konform.
- Die `kind`-Infrastruktur ist bereits voll funktionsfähig
  (`Makefile:158-204`, `scripts/cluster-smoke.sh` 196 Zeilen,
  `.github/workflows/cluster-smoke.yml`, `hack/cluster-smoke/
  cluster-state-stubs.yaml`). Der M5-Closure-Run gegen
  kindest/node:v1.34.0 hat alle fünf MVP-Conditions=True attestiert
  (Run-URL [26036254191](https://github.com/pt9912/k-deskflight/actions/runs/26036254191)).
- `kind` ist dem `envtest` operational überlegen: echter API-Server +
  kubelet + Scheduler + Storage-Provisioner — `envtest` simuliert
  nur `kube-apiserver`+`etcd`. Damit decken die M6-Tests automatisch
  auch `LH-AK-002` (Operator startbar im echten Cluster).
- `cmd/operator/main.go` ist nicht im Coverage-Selektor
  (`COVERPKG=./internal/...`, Dockerfile `coverage`-Stage Zeile 144).
  `cluster-smoke` ist deshalb der einzige Pfad, der den Wiring-Code
  attestiert. Eine zusätzliche `envtest`-Schicht für die fünf
  Adapter-Implementierungen würde die Wiring-Lücke nicht schließen,
  und die Adapter-Unit-Tests (`fake.NewSimpleClientset`) decken die
  reine Adapter-Logik bereits zu ≥ 90 %.

**Folge:** `spec/architecture.md` AR-024 („Integration-Tests via
`envtest`") wird im M6-Closure mit einer Übergabe-Notiz auf v0.2
verschoben — sobald die externen Service-Checks (`LH-F-018..021`,
`ADR 0010`) anstehen und parametrisierte Per-Check-Integrationstests
mit Fault-Injection brauchen. M6 dokumentiert das in §6 als bewusste
Abweichung und im Closure-Notiz als Out-of-Scope-Übergabe.

### 2.2 `/metrics`-Endpoint-Exposition (Service + RBAC + Smoke)

**Status heute:**

- Endpoint läuft auf `:8080` (`cmd/operator/main.go:44, 76`).
- `containerPort: 8080` ist im Deployment exposed
  (`deploy/manifests/deployment.yaml:46-48`).
- **Es gibt kein `Service`-Objekt** in `deploy/manifests/`.
- **Es gibt keine `nonResourceURLs: ["/metrics"]`-RBAC-Regel** für
  Prometheus-Scrape-ServiceAccounts.

**M6-Lieferungen:**

| Datei | Inhalt |
| ----- | ------ |
| `deploy/manifests/service.yaml` (neu) | `Service` `k-deskflight-operator-metrics` mit `port: 8080 (TCP) → metrics`, `selector: app.kubernetes.io/{name,component}: k-deskflight,operator`, `ClusterIP`. |
| `deploy/manifests/metrics-clusterrole.yaml` (neu) | `ClusterRole k-deskflight-metrics-scrape` mit `nonResourceURLs: ["/metrics"]`, `verbs: ["get"]`. Bewusst **kein** Binding mit Default-`prometheus`-ServiceAccount — Anwender bindet selbst (Doku-Pattern, `docs/user/installation.md`). **Disclaimer (siehe §2.2.1):** Die Rolle hat in M6 **keine funktionale Wirkung**, weil der `/metrics`-Endpoint unauthentisiert läuft. Sie ist Pattern-Asset für die in v0.2 geplante Auth-Filter-/`kube-rbac-proxy`-Aktivierung. |
| `deploy/manifests/kustomization.yaml` | Erweitert um `service.yaml` und `metrics-clusterrole.yaml`. |
| `hack/cluster-smoke/metrics-scrape-probe.yaml` (neu) | E2E-Verifikation für Service + Endpoint-Routing: ServiceAccount `metrics-scrape-probe` + Pod `metrics-scrape-probe` (Image `curlimages/curl:8.x`, `sleep infinity`-Command). Der Pod scraped via Service-DNS `http://k-deskflight-operator-metrics.k-deskflight-system.svc.cluster.local:8080/metrics` und attestiert HTTP 200 + Format-Marker (§2.2.2). Optional ein `ClusterRoleBinding`, der die `metrics-scrape`-ClusterRole an die Probe-SA bindet — strukturell mitausgeliefert, funktional in M6 ohne Wirkung. |
| `scripts/operator-http-smoke.sh` (erweitert) | Bereits prüft `/healthz`/`/readyz`/`/metrics` (M3-Stand). M6: drei zusätzliche Assertionen gegen die `/metrics`-Response — (a) **Format**: `# HELP`/`# TYPE`-Marker vorhanden; (b) **Inhalt**: mindestens eine der drei controller-runtime-Standard-Metriken `workqueue_depth`, `rest_client_requests_total`, `controller_runtime_reconcile_total` taucht im Body auf — robust gegen eine umbenannte Einzelmetrik (`||`-OR), aber widerstandsfähig gegen leeren Body trotz Format-Marker; (c) **Sanity**: Body hat ≥ 20 nicht-leere Zeilen (controller-runtime exponiert typisch > 80 Zeilen; 20 ist konservative Untergrenze gegen den Sub-Sekunden-Race „Manager-Init noch nicht durch"). Pfad: Port-Forward zum Operator-Pod (M3-Stand bleibt). |
| `scripts/cluster-smoke.sh` (Step 9b, neu) | Nach `operator-http-smoke.sh` (Port-Forward-Pfad): zweite Verifikation aus dem Cluster heraus via `kubectl exec metrics-scrape-probe -- curl -sf http://k-deskflight-operator-metrics.k-deskflight-system.svc:8080/metrics`. Damit ist **das neue Service-Objekt** funktional attestiert (Service-DNS auflöst, Selector matched Pod-Labels, Endpoint-Routing funktioniert). |

**RBAC-Pattern:** Wir liefern die `ClusterRole` mit, aber **kein**
`ClusterRoleBinding` mit konkretem ServiceAccount — der Anwender bindet
in seinem Monitoring-Namespace. Begründung: Wir kennen den
Monitoring-ServiceAccount nicht (kann `prometheus-k8s`, `victoriametrics`,
`grafana-agent`, `vector`, etc. sein), und das Pattern „ClusterRole
mitliefern, Binding extern" ist Standard im Operator-Ökosystem
(`cert-manager`, `external-dns`, `metrics-server` verfahren so).

#### 2.2.1 Disclaimer: ClusterRole-Wirkung in M6

Der `/metrics`-Endpoint wird in M6 **unauthentisiert** belassen
(controller-runtime-Default; kein Auth-Filter, kein `kube-rbac-proxy`-
Sidecar). Damit hat die `nonResourceURLs: ["/metrics"]`-ClusterRole in
M6 **keine funktionale Wirkung** — jeder Pod im Cluster kann das
Service-DNS scrapen, auch ohne Binding.

**Warum trotzdem ausliefern?** Die ClusterRole ist
Pattern-/Vorlagen-Asset für Anwender, die `kube-rbac-proxy` oder den
controller-runtime-`FilterProvider` selbst aktivieren. Das mitgelieferte
Manifest spart ihnen Recherche-Zeit. Die `installation.md`-Doku macht
den Disclaimer explizit.

**Endpoint-Authentication insgesamt** (Auth-Filter-/`kube-rbac-proxy`-
Aktivierung samt TLS-Cert-Lifecycle und Webhook-Token-Pfad) ist
ADR-pflichtig und v0.2-Slice — siehe §8 Out-of-Scope-Eintrag
„Metrics-Endpoint-Authentication".

#### 2.2.2 E2E-Verifikation für das neue Service-Objekt

`scripts/operator-http-smoke.sh` (M3-Stand) attestiert `/metrics` via
**Port-Forward** direkt zum Operator-Pod — das prüft den Endpoint
selbst, **nicht** das neue Service-Objekt. Damit das Service-Objekt
ehrlich attestiert ist, kommt in `scripts/cluster-smoke.sh` ein
zusätzlicher Step 9b dazu:

1. `kubectl apply -f hack/cluster-smoke/metrics-scrape-probe.yaml`
   (Probe-Pod im `default`-Namespace).
2. `kubectl wait pod metrics-scrape-probe --for=condition=Ready
   --timeout=60s`.
3. Drei Asserts aus dem Probe-Pod gegen die Service-DNS-Response
   (alle müssen Exit-Code 0 liefern):

   ```bash
   BODY=$(kubectl exec metrics-scrape-probe -- curl -sf \
     http://k-deskflight-operator-metrics.k-deskflight-system.svc:8080/metrics)
   # (a) Format-Marker:
   printf '%s\n' "$BODY" | grep -qE '^# (HELP|TYPE) '
   # (b) Inhalts-Beweis (mindestens eine controller-runtime-Standard-Metrik):
   printf '%s\n' "$BODY" | grep -qE '^(workqueue_depth|rest_client_requests_total|controller_runtime_reconcile_total)( |\{)'
   # (c) Sanity-Mindestlänge (controller-runtime exponiert typisch > 80 Zeilen):
   [ "$(printf '%s\n' "$BODY" | grep -vc '^$')" -ge 20 ]
   ```

Das attestiert sechs Eigenschaften in **einem** Curl-Aufruf:

- Service-DNS löst auf (DNS-Suchpfad funktioniert über Namespaces).
- Service-Selector matched mindestens ein Backend (sonst Endpoints
  leer → Connection refused).
- Service-Port 8080 routet auf den korrekten Container-Port.
- `/metrics`-Response ist im Prometheus-Format (`# HELP`/`# TYPE`-
  Marker vorhanden).
- Response hat **Inhalt** (nicht nur Header) — eine der drei
  Standard-Metriken muss da sein. Schutz gegen den Race „HTTP 200,
  leerer Body, weil Manager-Init noch nicht durch".
- Response-Länge ≥ 20 Zeilen — zweite Sanity-Linie gegen
  Mini-Response-Regressionen.

Die ClusterRole wird in diesem Pfad **nicht** funktional getestet
(siehe §2.2.1), aber **strukturell auf Inhalt** verifiziert (nicht
nur auf Existenz — sonst wäre eine kaputte Regel wie
`nonResourceURLs: ["/foo"]` oder `verbs: ["*"]` formal grün). Step
9c führt einen `jq`-Check aus:

```bash
kubectl get clusterrole k-deskflight-metrics-scrape -o json \
  | jq -e '.rules[] | select(.nonResourceURLs[]? == "/metrics") | select(.verbs == ["get"])' \
  > /dev/null
```

Der `-e`-Flag macht `jq` bei leerem Match-Set zum Exit-Code 1. Damit
ist garantiert: mindestens eine Rule listet `/metrics` in
`nonResourceURLs` **und** beschränkt sich auf das Verb `get`. Eine
Drift wie zusätzliche Verben (`["get", "list"]`), erweiterte Pfade
(`["/metrics", "/admin"]`) oder Wildcard (`["*"]`) bricht den Check
— Pattern-Asset bleibt semantisch sauber, auch ohne funktionale
Auth-Filter-Aktivierung.

**Generated-Drift-Konsequenz:** `service.yaml` und
`metrics-clusterrole.yaml` sind handgeschrieben und nicht von
`controller-gen` erzeugt — sie liegen unter `deploy/manifests/`, nicht
unter `config/`. Damit löst der Drift-Check (`make
generated-drift-check`) keine False-Positives aus.

### 2.3 `Spec.Interval` und `RequeueAfter` (AR-010, LH-F-025)

Aus M5 §10.3 übergeben. Implementierung minimal entlang der
`AR-010`-Vorgaben:

| Aspekt | Wert |
| ------ | ---- |
| API-Feld | `Spec.Interval *string` ohne `+kubebuilder:validation:Pattern` — Begründung in §2.3.1. |
| Default | `5m` (`architecture.md:578`, `AR-010`). |
| Bounds | `min=30s`, `max=24h` — Werte außerhalb werden auf den nächsten erlaubten Wert geklemmt. |
| Parse-Fehler | Normalisierung auf `5m`-Default; `Status.Conditions[ConfigurationInvalid]=True` mit Severity `warning`, Reason `IntervalNormalized`. **Kein** Reconcile-Abbruch (`AR-010.1` CR-Spec-Scope bleibt liveness-sicher). |
| Reconcile-Ausgang | `return ctrl.Result{RequeueAfter: normalizedInterval}, nil` am Ende eines erfolgreichen Laufs. |
| `OPERATOR_STRICT_CONFIG`-Wirkung | Keine — CR-Spec-Scope ist nicht restart-blockierend (`AR-010.1`). |

#### 2.3.1 API-Feld-Format ohne Schema-Pattern

Wir entscheiden uns für **`Spec.Interval *string` ohne
`+kubebuilder:validation:Pattern`**, weil ein Pattern und der
Normalisierer einander widersprechen würden:

- `AR-010` fordert wörtlich „Nicht parsebare Werte und Werte
  außerhalb des Bereichs werden auf den nächstgelegenen erlaubten
  Wert geklemmt". Das setzt voraus, dass invalide Strings den
  CRD-Validator **passieren** und den Reconciler erreichen — sonst
  ist der Normalisierer toter Code.
- Ein striktes Pattern wie `^[0-9]+(s|m|h)$` würde gleichzeitig
  legitime `time.ParseDuration`-Formate (`1h30m`, `90s`, `2h30m45s`)
  am CRD-Validator abweisen — der Anwender hätte ein gutes Format
  und bekäme trotzdem `kubectl apply`-Fehler.
- `metav1.Duration` ist aus demselben Grund untauglich: es bricht
  beim CRD-Schema-Validating, bevor der Reconciler die
  `ConfigurationInvalid`-Warning schreiben kann.

**Konsequenz:** Die Validierung liegt vollständig im Reconciler-
Normalisierer (`NormalizeInterval`); das CRD-Schema akzeptiert
jeden String als `Spec.Interval`-Wert. Der typische Anwender-
Trade-off „Fehler früh in `kubectl apply` vs. Fehler später als
Status-Warning" entscheiden wir bewusst pro **Warning**, weil das
liveness-sichere Verhalten aus `AR-010.1` zwingend ist.

Helper-Funktion `NormalizeInterval(raw string) (time.Duration,
*ConditionWarning)` in `internal/hexagon/application/interval.go`
(neue Datei). Reconciler ruft das vor `runChecks` auf und übergibt
das Result an die Status-Aggregation.

**Tabellen-Test-Cases** (alle laufen jetzt deterministisch durch
den Reconciler, weil das Schema keinen mehr abweist):

| Eingabe | Ergebnis | Warning |
| ------- | -------- | ------- |
| `nil`/leer | `5m` (Default — kein Wert vom Anwender) | nein |
| `0s` | `30s` (clamp min, `< min`) | ja, `IntervalNormalized` |
| `15s` | `30s` (clamp min, `< min`) | ja, `IntervalNormalized` |
| `30s` | `30s` | nein |
| `5m` | `5m` | nein |
| `1h30m` | `1h30m` (gültiges `time.ParseDuration`-Format) | nein |
| `24h` | `24h` | nein |
| `25h` | `24h` (clamp max, `> max`) | ja, `IntervalNormalized` |
| `abc` | `5m` (parse-fail → Default — kein interpretierbarer Wert) | ja, `IntervalNormalized` |
| `-5m` | `30s` (clamp min, `< min`) | ja, `IntervalNormalized` |

**Klassifikations-Regel:**

- **Parse-Erfolg + Wert in `[min, max]`** → unverändert übernehmen.
- **Parse-Erfolg + Wert `< min` oder `> max`** → clamp auf die
  verletzte Grenze + Warning. Das gilt auch für negative Durations
  (`time.ParseDuration("-5m")` parsed erfolgreich zu `-5min`, ist
  also `< min` → clamp auf `min`).
- **Parse-Fehler** (`time.ParseDuration` gibt `err != nil` zurück
  — z. B. `"abc"`, `""` nach Pointer-Dereferenzierung, `"5"` ohne
  Einheit) → Default `5m` + Warning. Hier ist „nächstgelegener
  erlaubter Wert" nicht definiert; AR-010 spricht von „Default,
  wenn Feld leer ist" als Default-Anker für nicht-interpretierbare
  Eingaben.

### 2.4 `docs/user/`-Struktur

**Status heute:** `docs/user/README.md` ist ein 18-Zeilen-Stub mit
„Aktuell keine Einträge".

**M6-Lieferungen:**

| Datei | Inhalt | Lastenheft/AR |
| ----- | ------ | ------------- |
| `docs/user/README.md` | Erweitert: Inhaltsverzeichnis mit Verweisen auf die vier neuen Sub-Dokumente. | `LH-AK-013`, `LH-NF-013`. |
| `docs/user/installation.md` (neu) | Schritt-für-Schritt-Installation via raw manifests (`kubectl apply -k deploy/manifests/`); Namespace-Override-Pattern via Kustomize-Overlay (schließt **`AR-OP-005`** aus `architecture.md`); Image-Pin-Override; Prometheus-Scrape-Binding-Pattern für die in §2.2 mitgelieferte `ClusterRole`. | `LH-AK-013`, `LH-NF-013`, `AR-OP-005`. |
| `docs/user/cr-examples.md` (neu) | Zwei vollständige CR-Beispiele plus Differenzen-Kommentar: ein `evaluation`-CR (lax, niedrige Schwellen, `Interval: 30m`) und ein `production`-CR (strikt, hohe Schwellen, `Interval: 5m`). Profile-Default-Auswertung (`LH-PROF-002`/`LH-PROF-003`) wird in den Beispielen sichtbar gemacht. | `LH-PROF-002`, `LH-PROF-003`, `LH-F-025`. |
| `docs/user/conditions.md` (neu) | Vollständiger Conditions-Katalog: pro `ConditionType*`-Konstante (KubernetesVersionReady, StorageClassReady, IngressClassReady, CertManagerInstalled, ClusterResourcesReady, plus die generischen ReconcileError/ConfigurationInvalid) die Reasons mit Severity (critical/warning/info), Bedeutung und Anwender-Action. Quelltext-Referenz: `internal/adapter/check/{name}.go` Reason-Konstanten + `internal/hexagon/application/runner.go` Synthetic-Reasons (Timeout/ReconcileTimeout/ReconcileCanceled/InternalError/RBACInsufficient/RBACCheckFailed/IntervalNormalized). | `LH-QA-004`, `LH-AK-011`. |
| `docs/user/troubleshooting.md` (neu) | Typische **M6-realistische** Fehlerbilder mit Diagnose-Kommando + Lösungs-Schritt: (1) Operator startet nicht → fehlende ClusterRole am Operator-SA; (2) alle Checks `Unknown` → fehlende RBAC-Permissions → SAR-Pfad; (3) cert-manager-Check `warning` → keine `cert-manager.io`-API-Gruppe gefunden; (4) **`/metrics` connection refused** → Service-Selector matched keinen Pod, oder Operator-Pod nicht `Ready`; (5) **`/metrics` DNS-Auflösung schlägt fehl** → CoreDNS-Issue oder falscher Namespace im FQDN; (6) **`/metrics` HTTP 200 aber leerer Body / Verbindung sofort geschlossen** → controller-runtime-Manager nicht gestartet, Probe-Liveness aber grün (Race); (7) **NetworkPolicy blockt Scrape** → Anwender hat im Monitoring-Namespace eine restriktive Default-Deny-Policy. **Explizit als v0.2-Eintrag markiert** (eigener Doku-Abschnitt mit Hinweis): (8) **Metrics-Endpoint HTTP 401** → tritt **erst nach v0.2-Auth-Filter-Aktivierung** auf (siehe §8); Lösungs-Schritt ist dann ClusterRoleBinding für Prometheus-SA — in M6 ist dieser Fall nicht reproduzierbar, weil der Endpoint unauthentisiert läuft (§2.2.1). | `LH-AK-013`, `LH-QA-004`. |

**Reihenfolge-Hinweis:** `docs/user/conditions.md` ist der größte
Eintrag (~30 Reasons über 5 Checks + 7 Synthetic-Reasons). Er sollte
**zuletzt** geschrieben werden, weil sich Reason-Konstanten während
der `Spec.Interval`-Arbeit (§2.3) noch erweitern können
(`IntervalNormalized` als neuer Reason).

### 2.5 Coverage-Gate strikt (`LH-QG-003`, `ADR 0012 §2.4`)

**Status:** Die `THRESHOLD ?= 90`-Variable steht bereits im Makefile
(Zeile 24); slice-M5 §10.2 hat 94.7 % über alle `internal/`-Pakete
attestiert. Der Coverage-Range-Selektor schließt `cmd/`-Wiring
bewusst aus (`architecture.md` AR-004, `Dockerfile`-coverage-Stage
Zeile 144). M6-Aufgabe ist deshalb keine Hebung, sondern eine
**Strikt-Verifikation**: dass die neuen M6-Code-Pfade (Interval-
Normalisierung, optionale Service/RBAC-Manifest-Tests) den 90 %-
Floor nicht unterlaufen.

**M6-Lieferungen:**

- `internal/hexagon/application/interval_test.go` (neu, ≥ 95 %
  Coverage über `interval.go`).
- Reconciler-Test-Erweiterung für `RequeueAfter`-Pfad und
  `ConfigurationInvalid`-Pfad (mindestens 2 zusätzliche Cases in
  `reconciler_test.go`).
- `make coverage-gate` läuft im PR-Gate weiter strikt; CI-Run
  attestiert das mit URL in §7.

**Was M6 nicht ändert:** Die Schwelle bleibt 90 %; `ADR 0012 §2.4`
nennt das Ziel ≥ 95 %, aber jede Hebung ist ADR-pflichtig
(`LH-QG-003`) und liegt explizit außerhalb des M6-Scopes.

### 2.6 `govulncheck` strikt (`LH-QG-006`, `ADR 0012 §2.6`)

**Status:** `make security-gates` ist seit M2 aktiv, `govulncheck
v1.1.4` ist im Makefile gepinnt (Zeile 150-153), CI-Workflow läuft
ihn parallel zu `make gates` (M5 §10.5 attestiert ihn am
2026-05-18 grün).

**M6-Lieferungen:**

- **Verifikation auf voller Code-Basis:** `make security-gates`
  bleibt grün, auch nach M6-Code-Wachstum (Interval, Service-Manifest-
  Tests).
- **Vulnignore-Konvention im README dokumentieren** (`ADR 0012
  §2.8`): falls ein Vulnignore-Eintrag aufgenommen werden muss, trägt
  er ein `expires`-Datum. Aktuell ist die Vulnignore-Datei nicht
  vorhanden (kein Bedarf) — der M6-Closure-Punkt verifiziert das
  weiterhin.
- **Anwender-Doku-Hinweis** in `docs/user/installation.md`: dass
  der Operator-Image-Tag (M7) bei `CRITICAL`-Vuln-Findings vom
  Release-Gate gestoppt wird (`LH-QG-007` Trivy ist M7, aber
  `LH-QG-006` ist heute aktiv).

### 2.7 Architektur-Boundary strikt (`LH-QG-004`)

**Status:** `.golangci.yml` Zeile 54-106 hält fünf depguard-Regeln,
M5 §10.2 hat sie auf der vollständigen Code-Basis nach Reconciler-/
Runner-/Adapter-Wachstum grün attestiert (`0 issues`). M6-Aufgabe
ist erneut Verifikation, nicht Neudefinition.

**M6-Lieferungen:**

- `make lint` bleibt grün auf der M6-Code-Basis (inkl.
  `interval.go`/`interval_test.go` in `application/`).
- Wenn der `Interval`-Normalisierer eine neue Abhängigkeit braucht
  (z. B. `k8s.io/apimachinery/pkg/util/wait`), muss vorher die
  `application`-Layer-depguard-Regel geprüft werden.
  **Voraussichtliche Lösung:** Wir nutzen ausschließlich
  `time.ParseDuration` (Standard-Library), damit kein neues k8s-
  Import in `application/` landet — `application` darf laut
  depguard `domain-isolation` keine k8s-Pakete importieren.

### 2.8 Per-Check passed + failed im Cluster-Smoke

**Status:** `scripts/cluster-smoke.sh` läuft heute mit **einem**
Sample-CR (`smoke` in `default`, alle fünf Checks passed). Step-8
asserted die fünf Conditions auf `status=True`. Der failed-Pfad
fehlt für jeden Check.

**M6-Strategie:** Mehrere CRs in einem Smoke-Lauf — nicht mehrere
Smoke-Läufe. Begründung: ein kind-Cluster aufzusetzen kostet 60-90 s
(`ADR 0013 §2.5`); fünf Smoke-Läufe würden die CI-Laufzeit von
~3 min auf ~20 min heben. Mehrere CRs in einem Lauf bringt die Last
auf zwei Reconcile-Zyklen.

**Konkretes Layout:**

| Sample-CR | Datei | Erwartung |
| --------- | ----- | --------- |
| `smoke` (bestehend) | `hack/cluster-smoke/cluster-state-stubs.yaml`-bezogene CR aus dem M4-Stand. | Phase=Passed, alle fünf Conditions True. (Bleibt unverändert.) |
| `smoke-failed-version` | `hack/cluster-smoke/failed-crs/version.yaml` (neu) | `kubernetesVersion.min: "99.99"` → Condition `KubernetesVersionReady=False`, Reason `KubernetesVersionTooOld`, Phase=Failed. |
| `smoke-failed-storage` | `hack/cluster-smoke/failed-crs/storage.yaml` (neu) | `storageClass.names: ["nonexistent"]` → Condition `StorageClassReady=False`, Reason `StorageClassMissing`. |
| `smoke-failed-ingress` | `hack/cluster-smoke/failed-crs/ingress.yaml` (neu) | `ingressClass.names: ["nonexistent"]` → Condition `IngressClassReady=False`, Reason `IngressClassMissing`. |
| `smoke-failed-resources` | `hack/cluster-smoke/failed-crs/resources.yaml` (neu) | `clusterResources.minCPU: "999"` → Condition `ClusterResourcesReady=False`, Reason `InsufficientCPU`. |
| `smoke-failed-rbac` | (manueller Pfad in `docs/user/troubleshooting.md`, nicht im CI-Smoke) | Operator-SA ohne `storage.k8s.io/storageclasses,verbs=list` → Condition `StorageClassReady=Unknown`, Reason `RBACInsufficient`. M5 §10.3-Übergabe. |

**cert-manager failed-Pfad (Ausnahme — §1 verweist hierauf):** Nicht
im automatischen Smoke, weil das M4-Closure die cert-manager-Stubs
explizit ins `cluster-state-stubs.yaml` (cluster-global) legt — den
Stub temporär wegzunehmen würde **alle** anderen CRs im selben Smoke-
Lauf auch failen lassen (StorageClass-Stub und IngressClass-Stub
sind kein Problem, weil sie pro CR-Konfiguration adressierbar sind;
die cert-manager-API-Gruppe ist es nicht). Den failed-Pfad sauber
zu produzieren würde einen M4-Smoke-Refactor verlangen
(Stubs aus `cluster-state-stubs.yaml` raus, in CR-Beispiel-Files
rein), der außerhalb des M6-Scopes liegt. Deshalb:

- **`LH-AK-008` bleibt durch M4-Stand attestiert** (Unit-Test 7 Fälle
  in `adapter/check/certmanager_test.go` + Cluster-Smoke passed-Case
  gegen Stub-CRD). M6 re-attestiert es nicht.
- `troubleshooting.md` bekommt einen Eintrag „cert-manager-Check
  warning", der den failed-Pfad als Diagnose-Szenario textuell
  beschreibt (Reason `CertManagerMissing`).
- v0.2-Übergabe: wenn die M4-Stub-Architektur ohnehin überarbeitet
  wird (z. B. weil v0.2 ClusterIssuer-Detail-Validierung `LH-F-014`
  bringt), kann der failed-Pfad nachgezogen werden.

**Implementation-Strategie für `scripts/cluster-smoke.sh` (Step 6b
mit Status-Kollateralschutz):** Nach Step 6 (Sample-CR `smoke`
appliziert + erwartete Phase=Passed konvergiert) iteriert Step 6b
**seriell** über die vier `failed-crs/*.yaml`-Files. Pro CR:

1. `kubectl apply -f hack/cluster-smoke/failed-crs/<name>.yaml`.
2. `kubectl wait opendeskpreflightcheck/<name> --for=jsonpath=
   '{.status.phase}'=Failed --timeout=60s`.
3. Assertion: die für diesen CR erwartete Condition steht auf
   `status=False` mit dem dokumentierten Reason.
4. **Erst dann** wird die nächste failed-CR appliziert.

**Eindeutige `metadata.name` pro CR** (verhindert Status-Mischung
zwischen Reconcile-Zyklen): `smoke`, `smoke-failed-version`,
`smoke-failed-storage`, `smoke-failed-ingress`,
`smoke-failed-resources`. Jeder Name belegt eine eigene OPDC-
Ressource im `default`-Namespace; controller-runtime-Watch
trackt sie separat, Status-Updates pro CR sind voneinander
isoliert.

**Kein `kubectl delete` zwischen den Cases nötig**, weil die CRs
unterschiedliche Namen tragen und der MVP-Reconciler keine
inter-CR-Abhängigkeiten kennt (jede OPDC-CR wird unabhängig
reconciled). Falls die Step-6b-Loop in einer späteren Slice
inter-CR-Effekte einführt (z. B. ConfigMap-Report-Aggregation in
v0.2, `LH-F-028`), muss die Strategie auf Delete-zwischen-Cases
gehoben werden — das ist v0.2-Folge-Reaktion, kein M6-Bruch.

Step-8-Assertion bekommt ein zweites Set Erwartungen aus den
failed-CRs (eine Erwartungs-Tabelle pro CR-Name → Phase + erwartete
Conditions/Reasons).

**RBAC-failed-Pfad als Out-of-CI-Smoke:** Das `smoke-failed-rbac`-
Szenario braucht eine separate ServiceAccount + ClusterRole-Mutation;
das ist im CI-Smoke nicht trivial ohne den ganzen Cluster neu
aufzusetzen. M6 dokumentiert es als manuellen Reproduktions-Schritt
in `troubleshooting.md` und als M6-Closure-Verifikation per Hand
(Plan §7 #12), nicht als CI-Pflicht.

---

## 3. Datei-Inventar

### 3.1 Neue Code-/API-Dateien

| Pfad | Zweck |
| ---- | ----- |
| `internal/hexagon/application/interval.go` | `NormalizeInterval(raw string) (time.Duration, *ConditionWarning)` — clamp/default/parse-fail-Pfade gemäß `AR-010`. Nur `time.ParseDuration` aus stdlib, damit `application`-Layer-depguard nicht bricht (§2.7). |
| `internal/hexagon/application/interval_test.go` | Tabellen-Test (10 Cases gemäß §2.3.1-Tabelle): leer→5m, 0s→30s, 15s→30s, 30s→30s, 5m→5m, 1h30m→1h30m, 24h→24h, 25h→24h, „abc"→5m + Warning, „-5m"→30s + Warning. |

### 3.2 Erweiterte Code-/API-Dateien

| Pfad | Änderung |
| ---- | -------- |
| `api/v1alpha1/opendeskpreflightcheck_types.go` | `Spec` bekommt `Interval *string` **ohne** `+kubebuilder:validation:Pattern` (Begründung §2.3.1) und mit Default-Marker `// +kubebuilder:default="5m"`. Generated-Drift-Lauf nötig (`make manifests`). |
| `internal/hexagon/application/reconciler.go` | `Reconcile` ruft am Anfang `NormalizeInterval(cr.Spec.Interval)`, übergibt das Result an die Status-Aggregation (ConfigurationInvalid-Warning kommt als zusätzliche Condition rein), und am Ende `return ctrl.Result{RequeueAfter: normalizedInterval}, nil`. |
| `internal/hexagon/application/reconciler_test.go` | Drei neue Cases: `TestReconcileIntervalDefaulted` (leer→5m + RequeueAfter), `TestReconcileIntervalNormalized` (parse-fail→Warning-Condition + default-RequeueAfter), `TestReconcileIntervalClamped` (25h→24h-RequeueAfter + Warning). |
| `deploy/manifests/kustomization.yaml` | Resources-Liste erweitert um `service.yaml` und `metrics-clusterrole.yaml` (§2.2). |
| `scripts/operator-http-smoke.sh` | Zusätzliche Assertion auf generischen `# HELP`-/`# TYPE`-Marker im `/metrics`-Response (Prometheus-Format-Verifikation; robust gegen library-Drift, §9). |
| `scripts/cluster-smoke.sh` | Step 6b (serielles Apply + `kubectl wait` pro CR aus `hack/cluster-smoke/failed-crs/*.yaml`, §2.8), Step 8 (zweite Assertion-Phase für failed-Cases), Step 9b (Probe-Pod-Apply + `kubectl exec`-curl via Service-DNS, §2.2.2), Step 9c (`jq`-Inhaltscheck der `metrics-scrape`-ClusterRole, §2.2.2-Code-Block). |
| `hack/cluster-smoke/cluster-state-stubs.yaml` | Unverändert. |

### 3.3 Neue Deploy-Manifest-Dateien

| Pfad | Inhalt |
| ---- | ------ |
| `deploy/manifests/service.yaml` | `Service k-deskflight-operator-metrics`, ClusterIP, Port 8080 → Target `metrics`. |
| `deploy/manifests/metrics-clusterrole.yaml` | `ClusterRole k-deskflight-metrics-scrape` mit `nonResourceURLs: ["/metrics"]`, `verbs: ["get"]`. **Kein** Binding (Anwender bindet selbst, Pattern in `docs/user/installation.md` dokumentiert). |

### 3.4 Neue Cluster-Smoke-Failed-CRs und Probe-Pod

| Pfad | Inhalt |
| ---- | ------ |
| `hack/cluster-smoke/failed-crs/version.yaml` | OPDC `metadata.name: smoke-failed-version` (default-Namespace) mit `kubernetesVersion.min: "99.99"` → Reason `KubernetesVersionTooOld`. |
| `hack/cluster-smoke/failed-crs/storage.yaml` | OPDC `metadata.name: smoke-failed-storage` mit `storageClass.names: ["nonexistent"]` → Reason `StorageClassMissing`. |
| `hack/cluster-smoke/failed-crs/ingress.yaml` | OPDC `metadata.name: smoke-failed-ingress` mit `ingressClass.names: ["nonexistent"]` → Reason `IngressClassMissing`. |
| `hack/cluster-smoke/failed-crs/resources.yaml` | OPDC `metadata.name: smoke-failed-resources` mit `clusterResources.minCPU: "999"` → Reason `InsufficientCPU`. |
| `hack/cluster-smoke/metrics-scrape-probe.yaml` | E2E-Verifikation für Service + Endpoint-Routing (§2.2.2): ServiceAccount + Pod `metrics-scrape-probe` (`curlimages/curl:8.x`, `sleep infinity`); optional ClusterRoleBinding für die `k-deskflight-metrics-scrape`-ClusterRole. Probe scraped via Service-DNS in Step 9b von `scripts/cluster-smoke.sh`. |

### 3.5 Neue Anwender-Doku

| Pfad | Inhalt |
| ---- | ------ |
| `docs/user/README.md` (erweitert) | ToC mit Verweisen auf die vier Sub-Dokumente; Hinweis auf `kdeskflight_*`-Prefix-Reservierung für v0.2-Domänen-Metriken (`ADR 0007 §2.3`). |
| `docs/user/installation.md` | Raw-manifest-Installation, Namespace-Override-Pattern (schließt `AR-OP-005`), Image-Pin-Override, Prometheus-Scrape-Binding-Pattern. |
| `docs/user/cr-examples.md` | Zwei vollständige CR-Beispiele (`evaluation`/`production`) plus Differenzen-Kommentar. |
| `docs/user/conditions.md` | Conditions-Katalog: 5 ConditionTypes × Reason-Liste + 7 Synthetic-Reasons (Timeout, ReconcileTimeout, ReconcileCanceled, InternalError, RBACInsufficient, RBACCheckFailed, IntervalNormalized) + 2 generische (ReconcileError, ConfigurationInvalid). |
| `docs/user/troubleshooting.md` | Mindestens 5 typische Fehlerbilder mit Diagnose-Kommando und Lösungs-Schritt (Operator-Start, alle Checks Unknown, cert-manager-warning, Metrics-401, Interval-Warning). |

### 3.6 RBAC-/Generated-Drift-Konsequenzen

`Spec.Interval`-Hinzufügung an `OpenDeskPreflightCheck`-Spec **bricht
nicht** die `+kubebuilder:rbac:`-Marker (die zielen auf Verben/
Ressourcen, nicht auf Spec-Felder). Aber:

- `make manifests` muss laufen, weil das CRD-OpenAPI-Schema durch
  das neue Feld wächst.
- `make generated-drift-check` muss in demselben Commit grün sein.
- Cluster-Smoke verifiziert, dass die alte CR `smoke` (ohne
  `Interval`-Feld) weiter `Phase=Passed` liefert (Backward-
  Compatibility — `Interval` ist optional mit Default).

### 3.7 Audit-/Konsistenz-Tests

Keine neuen Audit-Tests in M6. Die bestehenden
(`adapter/check/rbac_consistency_test.go`,
`application/destructive_audit_test.go` aus M5) decken Drift weiter
ab. M6 verifiziert nur, dass sie grün bleiben.

---

## 4. Reihenfolge der Umsetzung

Jeder Schritt = ein eigener Commit; lokal `make gates` grün ziehen.
**Lektion aus M5 §10.4** („Reihenfolge-Bruch wiederholt sich"):
Reconciler-Pfade und Wiring/Beispiele bleiben im selben Commit, damit
`make cluster-smoke` nicht zwischendurch rot wird.

1. **`Spec.Interval`-API + Normalisierer + Reconciler-RequeueAfter
   in einem Commit.** `api/v1alpha1/opendeskpreflightcheck_types.go`
   (neues Feld + kubebuilder-Marker), `internal/hexagon/application/
   interval.go` (neu), `interval_test.go` (neu), `reconciler.go`
   (Aufruf von `NormalizeInterval` + `RequeueAfter`),
   `reconciler_test.go` (drei neue Cases). `make manifests` + `make
   generated-drift-check`. Smoke-CR `smoke` bleibt unverändert (ohne
   `Interval`-Feld) und muss `Phase=Passed` liefern.

2. **Service-Manifest + Metrics-ClusterRole + Probe-Pod-Stub +
   Kustomization-Update + http-smoke-Erweiterung + Cluster-Smoke-
   Step-9b in einem Commit.** `deploy/manifests/service.yaml`,
   `metrics-clusterrole.yaml`, `kustomization.yaml` (Resources
   erweitert), `hack/cluster-smoke/metrics-scrape-probe.yaml`,
   `scripts/operator-http-smoke.sh` (`# HELP `-Marker-Assertion),
   `scripts/cluster-smoke.sh` (Step 9b für Service-DNS-E2E gemäß
   §2.2.2). Lokal `make cluster-smoke` grün: Port-Forward-Pfad und
   Service-DNS-Pfad liefern beide HTTP 200 + Format-Marker.

3. **Failed-Cluster-Smoke-CRs + `scripts/cluster-smoke.sh`-Loop in
   einem Commit.** Vier `hack/cluster-smoke/failed-crs/*.yaml`-Files,
   Step-6b-Loop in `cluster-smoke.sh`, Step-8-Assertion erweitert.
   Lokal `make cluster-smoke` deckt nun **fünf CRs** ab (1 passed-
   Bestand `smoke` + 4 failed-Cases version/storage/ingress/
   resources; cert-manager-failed ist M6-Ausnahme gemäß §1+§2.8).
   Laufzeit bleibt akzeptabel (vier zusätzliche Reconcile-Zyklen,
   nicht vier zusätzliche Cluster-Bootstraps).

4. **Anwender-Doku Teil 1: Installation + CR-Beispiele.**
   `docs/user/README.md` erweitert (ToC),
   `docs/user/installation.md` (neu, schließt `AR-OP-005`),
   `docs/user/cr-examples.md` (neu, zwei Beispiele evaluation+
   production). `make doc-refs` grün — alle Querverweise müssen
   auflösen.

5. **Anwender-Doku Teil 2: Conditions-Katalog.**
   `docs/user/conditions.md` (neu, 5 ConditionTypes + 9 generische/
   synthetische Reasons). Quelltext-Validierung als **strukturierter
   grep** vor Commit:

   ```bash
   # Reason-Konstanten aus Code extrahieren (Pattern matched die
   # M5-Konvention `ReasonXxx = "..."` und `ConditionTypeXxx = "..."`):
   grep -rhoE '(Reason|ConditionType)[A-Z][a-zA-Z]+ *= *"[^"]+"' \
     internal/adapter/check/ internal/hexagon/application/ \
     | sed -E 's/.*"([^"]+)"/\1/' | sort -u > /tmp/reasons-code.txt
   # Reasons aus Doku extrahieren (Pattern matched die in §2.4
   # vereinbarte Doku-Konvention `**Reason:** Xxx`):
   grep -hoE '\*\*Reason:\*\* *[A-Z][a-zA-Z]+' \
     docs/user/conditions.md \
     | sed -E 's/.*Reason:\*\* *//' | sort -u > /tmp/reasons-doc.txt
   diff /tmp/reasons-code.txt /tmp/reasons-doc.txt
   ```

   Damit ist die Validierung an Code-Konstanten gebunden (nicht an
   freitext-Substrings), und die Doku-Konvention `**Reason:** Xxx`
   ist im Header von `conditions.md` als Pflicht-Schema verankert.
   **Kein neuer Audit-Test im M6-Scope** (sonst Scope-Creep) — die
   strukturierte grep-Variante reicht, solange die Reason-Liste
   unter ~50 Einträgen bleibt. **v0.2-Übergabe:** sobald die Liste
   größer wird oder externe Service-Checks (`LH-F-018..021`) neue
   Reasons einführen, wird der grep-Check durch einen `go/parser`-
   Audit-Test im `internal/audit/markers/`-Mini-Package abgelöst
   (Pattern wie M5 `rbac_consistency_test.go`).

6. **Anwender-Doku Teil 3: Troubleshooting.**
   `docs/user/troubleshooting.md` (neu, 7 M6-realistische
   Fehlerbilder inkl. dem RBAC-eingeschränkt-Smoke aus M5
   §10.3-Übergabe + 1 explizit als v0.2-Eintrag markierter
   Auth-401-Fall, siehe §2.4 für die Liste).

7. **Slice-Closure.** Slice-Datei nach `done/` ziehen,
   Roadmap-Status auf M6=Done, Closure-Notiz mit Verifikations-
   Ergebnis und CI-Run-URLs. Roadmap-§7-Statuszeile aktualisieren
   („M1–M6 geschlossen, M7 weiterhin Pending").

**Erwartete Commit-Zahl:** 6 Code-/Doku-Commits + 1 Closure-Commit.
Mehr als M5 wegen der Doku-Stückelung — aber jedes Stück lässt sich
isoliert reviewen.

---

## 5. Lastenheft-Kennungen

**Pflichtkennungen aus Roadmap §3 M6:**

`LH-SST-004` (Prometheus-Format, `ADR 0007 §2.1` — M6 erfüllt
Endpoint-Reachability + Service-Routing + Prometheus-Format;
**funktionale Auth-/RBAC-Absicherung des Endpoints ist v0.2**
mit eigener ADR-Folge, siehe §2.2.1 + §8),
`LH-NF-008` (Eigene Metriken — M6 erfüllt nur den Endpoint-Anteil,
voll v0.2 gemäß ADR 0007 §2.1),
`LH-NF-010` (Testbarkeit — cluster-smoke deckt das ab),
`LH-NF-013` (Dokumentation),
`LH-QA-002` (reproduzierbare Ergebnisse — durch envtest-/kind-
deterministische Cluster-State-Stubs erfüllt; M6 verifiziert nur),
`LH-QA-004` (transparente Bewertung — Conditions-Katalog ist der
operative Anker),
`LH-QG-002` (Tests — `make cluster-smoke` ist die kind-Variante des
Lastenheft-Erfüllungspfads),
`LH-QG-003` (Coverage-Gate strikt 90 %),
`LH-QG-004` (Architektur-Boundary strikt),
`LH-QG-006` (Vulnerability-Scan strikt),
`LH-AK-013` (Dokumentation vorhanden — Hauptverifikationsanker).

**Zusätzliche Kennungen aus den M5-Übergaben:**

`LH-F-025` (Wiederholintervall — `Spec.Interval` + RequeueAfter,
§2.3),
`LH-AK-016` (RBAC-Selbstprüfung wirksam — der M5-`failed-rbac`-
Smoke-Pfad wird in `troubleshooting.md` als manueller Reproduktions-
Schritt verankert, §2.8).

**Re-Attest-Kennungen aus M3/M4 via Failed-Cluster-Smoke
(§2.8):**

`LH-AK-005` (KubernetesVersion failed),
`LH-AK-006` (StorageClass failed),
`LH-AK-007` (IngressClass failed),
`LH-AK-009` (ClusterResources failed).

**Nicht re-attestiert in M6** (M4-Stand bleibt gültig, Begründung
§1+§2.8): `LH-AK-008` (cert-manager — failed-Pfad nur Unit-Test
in `adapter/check/certmanager_test.go`, kein automatischer Cluster-
Smoke-Case).

**Indirekt erfüllt durch Doku-Anker (§2.4):**

`LH-PROF-002` (`evaluation`-Profil — über das CR-Beispiel),
`LH-PROF-003` (`production`-Profil — über das CR-Beispiel),
`LH-AK-011` (Conditions vorhanden — Conditions-Katalog ist die
Erfüllung der „dokumentiert"-Lesart),
`LH-AK-015` (Minimalrechte dokumentiert — `installation.md`
verweist auf das mitgelieferte ClusterRole-Set und die ergänzende
metrics-ClusterRole).

---

## 6. Architekturartefakte

Erfüllt aus `spec/architecture.md`:

- `AR-010` Wiederholintervall — scharf via `NormalizeInterval` +
  `RequeueAfter`-Aufruf im Reconciler (§2.3).
- `AR-010.1` Konfigurationsklassifizierung — `Spec.Interval` als
  CR-Scope-Konfiguration (nicht restart-blockierend), bleibt
  liveness-sicher (§2.3).
- `AR-OP-005` (Anwender-Overridebarkeit des Default-Operator-
  Namespace) — geschlossen via `docs/user/installation.md`
  (Kustomize-Overlay-Pattern, §2.4).

**Bewusste Abweichung von AR-024 — Begründung im Plan §2.1:**

- `AR-024` „Integration-Tests via `envtest`" wird in M6 **nicht**
  eingelöst. Die `kind via cluster-smoke`-Variante ist
  lastenheft-konform (`LH-QG-002` listet beide gleichwertig) und
  operational stärker (echter API-Server + kubelet). Übergabe an
  v0.2 zusammen mit `LH-F-018..021` externen Service-Checks, die
  parametrisierte Fault-Injection brauchen werden.

Vorbereitet, aktiv ab späterer Slice:

- `AR-024` Integration-Tests via `envtest` — v0.2 (siehe oben).
- `AR-026` Leader-Election — M7 (RBAC für Leases bereits in M2; in
  M6 wird `--leader-elect=true` als MVP-Default in
  `docs/user/installation.md` nur dokumentiert).
- `AR-OP-006` OTel-Tracing — v0.2.
- `AR-OP-007` Conversion-Webhook für künftige Versionssprünge —
  Folge-ADR.
- `AR-OP-008` Harte Namespace-/Tenant-Isolation — v0.2.

---

## 7. Verifikation (Abnahmekriterien)

1. **`make build`** baut den erweiterten Reconciler + Interval-
   Normalisierer.
2. **`make lint`** grün — `interval.go` bleibt `application`-layer-
   konform (nur `time.ParseDuration` aus stdlib).
3. **`make test`** grün; alle neuen Unit-Tests bestehen (interval-
   Tabelle, drei neue Reconciler-Cases).
4. **`make coverage-gate`** grün bei Threshold 90 % strikt
   (`LH-QG-003`, `ADR 0012 §2.4`). Harte Bedingung ist `≥ 90 %`;
   die in §2.5 genannte Erwartungsschätzung („Coverage bleibt im
   M5-Bereich 94.7 %") ist Prognose, kein Abnahmekriterium.
5. **`make doc-refs`** grün — alle neuen `docs/user/`-Querverweise
   auflösen, ADR-/Architektur-/Lastenheft-Referenzen stimmen.
6. **`make generated-drift-check`** grün — die `Spec.Interval`-
   Hinzufügung erzeugt CRD-Schema-Drift, der per `make manifests`
   nachgezogen wird; nach dem `manifests`-Lauf ist kein weiterer
   Drift.
7. **`make gates`** grün (Bundle) auf GitHub-Actions, attestiert
   per Run-URL in der Closure-Notiz.
8. **`make security-gates`** grün — `govulncheck` ohne Findings;
   `LH-QG-006` strikt aktiv (ist seit M2 grün, M6 verifiziert nur).
9. **`make cluster-smoke` mit fünf CRs grün** (`LH-QG-002`,
   `LH-AK-005`/`-006`/`-007`/`-009`-Re-Attest):
   - `smoke` (passed-Bestand): Phase=Passed, fünf Conditions True.
   - `smoke-failed-version`: Phase=Failed, KubernetesVersionReady=
     False, Reason=KubernetesVersionTooOld.
   - `smoke-failed-storage`: Phase=Failed, StorageClassReady=False,
     Reason=StorageClassMissing.
   - `smoke-failed-ingress`: Phase=Failed, IngressClassReady=False,
     Reason=IngressClassMissing.
   - `smoke-failed-resources`: Phase=Failed, ClusterResourcesReady=
     False, Reason=InsufficientCPU.

   **Nicht enthalten** (Begründung §1+§2.8): cert-manager-failed-CR
   — M4-Stand bleibt für `LH-AK-008` gültig.
10. **`/metrics`-Endpoint im Smoke reachable, Port-Forward-Pfad**
    (`LH-SST-004`, `ADR 0007 §2.1`): `scripts/operator-http-smoke.sh`
    liefert HTTP 200 vom `/metrics`-Pfad direkt am Operator-Pod
    **und** alle drei Assertionen aus §2.2-Tabelle bestehen:
    (a) `# HELP`/`# TYPE`-Format-Marker, (b) mindestens eine der
    controller-runtime-Standard-Metriken (`workqueue_depth`,
    `rest_client_requests_total`, `controller_runtime_reconcile_total`)
    im Body, (c) Body ≥ 20 nicht-leere Zeilen. Schutz gegen
    library-Versions-Drift (a + b im OR), gegen leeren-Body-Race
    (b + c) und gegen Mini-Response-Regressionen (c).
11. **`/metrics`-Service via Service-DNS reachable, E2E-Pflicht**
    (§2.2.2): Probe-Pod scraped via `kubectl exec` + `curl` auf
    Service-DNS-FQDN. Alle drei Asserts aus §2.2.2-Code-Block
    bestehen: (a) Format-Marker, (b) Standard-Metrik im Body, (c)
    Body ≥ 20 nicht-leere Zeilen. Damit ist das **neue Service-
    Objekt** funktional attestiert (Service-DNS auflöst, Selector
    matched ≥ 1 Backend, Port 8080 routet korrekt) **plus** der
    Endpoint hat semantisch validen Inhalt (nicht nur HTTP-200-Header
    ohne Body). Strukturell wird
    zusätzlich verifiziert: `kubectl get svc k-deskflight-operator-
    metrics -n k-deskflight-system` existiert; die
    `k-deskflight-metrics-scrape`-ClusterRole **nicht nur auf
    Existenz, sondern auf Regel-Inhalt** geprüft (Step 9c, §2.2.2
    Code-Block): `jq -e` muss mindestens eine Rule finden, die
    `/metrics` in `nonResourceURLs` listet und `verbs == ["get"]`
    hat — schützt gegen Drift wie zusätzliche Verben, erweiterte
    Pfade oder Wildcard.

    **Out-of-Scope dieses Abnahmepunkts — explizit nicht erfüllt
    durch M6 (Übergabe v0.2, siehe §8 „Metrics-Endpoint-
    Authentication"):** funktionale Wirkung der `metrics-scrape`-
    ClusterRole. Der Endpoint läuft controller-runtime-Default-
    unauthentisiert (§2.2.1); ein Probe-Pod ohne ClusterRoleBinding
    kann ebenfalls scrapen. Die mitgelieferte ClusterRole ist
    Pattern-Asset für eine v0.2-ADR-Folge zu `ADR 0007`, die
    `kube-rbac-proxy`-Sidecar oder controller-runtime-
    `FilterProvider` aktiviert. **Dieser Abnahmepunkt gilt
    deshalb nicht als „RBAC funktional durchgesetzt", sondern
    als „RBAC-Pattern-Asset ausgeliefert + Service-Routing
    attestiert".**
12. **`LH-AK-013` Dokumentation vorhanden** — `docs/user/`
    enthält vier neue Dateien (`installation.md`, `cr-examples.md`,
    `conditions.md`, `troubleshooting.md`) plus den erweiterten
    `README.md` mit ToC. Alle Conditions-Katalog-Einträge sind
    Quelltext-validiert (grep-Pass).
13. **`Spec.Interval`-Verhalten** (`LH-F-025`, `AR-010`):
    - Leer → Default `5m` → `RequeueAfter=5m`.
    - Parse-fail → Default `5m` + Condition
      `ConfigurationInvalid=True`, Reason=`IntervalNormalized`,
      Severity=`warning`.
    - Out-of-bounds → clamp + Warning-Condition.
    Verifiziert durch `interval_test.go` (Tabelle) und drei neue
    Reconciler-Cases.
14. **`LH-AK-016` RBAC-failed-Smoke als manueller Pfad** —
    M5 §10.3-Übergabe: `troubleshooting.md` dokumentiert den
    Reproduktions-Schritt (Operator-SA temporär aus
    `clusterrolebinding.yaml` entbinden, `kubectl describe opdc
    smoke` zeigt Phase=Unknown + Reason=RBACInsufficient).
    Im M6-Closure attestiert per Hand (kein CI-Smoke).

---

## 8. Out-of-Scope (geht in M7 / v0.2)

- **Beispielmanifest-Vollständigkeit + v0.1.0-Tag** — M7
  (`LH-MVP-002`, `LH-AK-014`). Die M6-CR-Beispiele unter
  `docs/user/cr-examples.md` werden in M7 als kanonische Samples
  nach `deploy/manifests/samples/` gehoben.
- **Trivy-Image-Scan** (`LH-QG-007`) — M7-Release-Gate.
  `govulncheck` (`LH-QG-006`) ist M6, Trivy ist M7.
- **DCO-Compliance-Check** vor Merge — M7-Release-Pflicht
  (`ADR 0011`).
- **Leader-Election scharf via Flag/Env** (`AR-026`) — M7-
  Release-Hardening. M6 dokumentiert nur den MVP-Default
  (`--leader-elect=true` aktiv).
- **`envtest`-basierte Integrationstests** (`AR-024`) — v0.2,
  Begründung in §2.1.
- **Eigene Domänen-Metriken** (`LH-NF-008` wörtlich, `kdeskflight_*`-
  Prefix) — v0.2 (`ADR 0007 §2.3`). M6 reserviert den Prefix nur
  in der Anwender-Doku.
- **ServiceMonitor / PodMonitor / PrometheusRule** — v0.2 zusammen
  mit Helm-Chart (`ADR 0007 §4`, `ADR 0005`).
- **Metrics-Endpoint-Authentication** (Auth-Filter via controller-
  runtime-`FilterProvider` oder `kube-rbac-proxy`-Sidecar samt
  TLS-Cert-Lifecycle und Token-Webhook-Pfad) — v0.2 mit eigener
  ADR-Folge zu `ADR 0007`. M6 liefert die `nonResourceURLs`-
  ClusterRole als Pattern-Asset mit, aber funktional bleibt der
  Endpoint unauthentisiert (§2.2.1).
- **`OPERATOR_STRICT_CONFIG`-Wirkung auf `Spec.Interval`** — bleibt
  laut `AR-010.1` ausdrücklich **nicht** restart-blockierend;
  Runtime-Config-Anteil (`RECONCILE_TIMEOUT_SECONDS`,
  `CHECK_TIMEOUT_SECONDS`, etc.) ist v0.2-Slice gemäß `AR-010.1`-
  Übergabe.
- **OTel-Tracing-Spans im Reconcile-Pfad** (`AR-OP-006`) — v0.2
  gemäß `ADR 0007 §4`.
- **K8s-Events** (`LH-F-027`) — v0.2; `LogResult`-Sanitize-Hook-
  Pattern aus M5 ist bereits vorbereitet.
- **HTML-Report / kubectl-Plugin** — spätere Versionen
  (`LH-PRI-003`).
- **Conditions-Katalog-Audit-Test** (Quelltext-Reasons ↔ Doku-
  Reasons) — v0.2-Pattern, sobald die Reason-Liste wächst. M6
  verifiziert die Konsistenz nur per grep im Slice-Closure.

---

## 9. Risiken und Mitigation

- **`Spec.Interval` ohne CRD-Schema-Pattern — Fehler erst als
  Status-Warning, nicht als `kubectl apply`-Fehler** (§2.3.1):
  Anwender, die `Spec.Interval: "abc"` setzen, bekommen kein
  Sofort-Feedback vom API-Server — ihr `apply` wird akzeptiert, der
  Operator zeigt die Normalisierung erst nach dem ersten Reconcile
  als `ConfigurationInvalid`-Warning-Condition. **Mitigation:**
  `docs/user/conditions.md` listet `IntervalNormalized` als
  prominenten Eintrag mit Anwender-Action („prüfe `Status.Conditions`
  nach jedem CR-Apply, vor allem bei neuen Anwendern");
  `docs/user/troubleshooting.md` hat einen dedizierten Eintrag
  „Interval-Warning erschienen". **Akzeptierte Konsequenz:**
  `AR-010.1` zwingt zu CR-Spec-Scope-liveness-Sicherheit
  (Reconcile darf nicht durch ungültige Spec-Werte hart brechen);
  diese Anforderung schließt CRD-Schema-Pattern aus, weil dort
  abgewiesene Werte den Normalisierer nie erreichen würden.

- **Cluster-Smoke-Laufzeit bei fünf CRs:** der controller-runtime-
  Watch reagiert auf jeden Apply mit einem Reconcile, parallel
  laufen sie nicht (Single-Replica + kein Worker-Pool im MVP). Jede
  CR = ein Reconcile = bis zu 30 s per-Check-Timeout × bis zu fünf
  Checks = im theoretischen Worst-Case 150 s pro CR. Fünf CRs ×
  150 s = 12.5 min Worst-Case — bleibt unter dem 20-min-CI-Timeout
  des cluster-smoke-Workflows. **Mitigation:** Die failed-CRs sind
  so gewählt, dass jeder Check **trotz Fail** in O(1)-API-Calls
  zurückkommt (Lookup-Treffer, kein Retry). Realistische Laufzeit
  pro CR ≤ 10 s, gesamt für fünf CRs ≤ 60 s zusätzlich zum
  bestehenden Smoke-Lauf. Reserveplan: wenn das im CI länger
  dauert, kann der `cluster-smoke`-Job auf 30 min hochgezogen
  werden — Plan-Bruch nicht erforderlich.

- **`Service`-Objekt-Selector mismatched mit Deployment-Pod-Labels:**
  trivial bei Tippfehler. **Mitigation:** Die Selector-Labels werden
  1:1 aus dem Deployment-Template kopiert (`app.kubernetes.io/name:
  k-deskflight`, `app.kubernetes.io/component: operator`); der
  neue `scripts/cluster-smoke.sh` Step 9b (§2.2.2) deckt den Fall
  funktional ab — wenn der Selector nichts matched, sind die
  Endpoints leer, der Probe-Pod-`curl` bekommt Connection refused
  und der Smoke wird rot.

- **`metrics-scrape`-ClusterRole als Schein-Sicherheit** (§2.2.1):
  Anwender könnten aus dem Vorhandensein der ClusterRole schließen,
  dass der `/metrics`-Endpoint authentisiert sei — er ist es nicht.
  Wer im Cluster Netzwerk-Zugriff auf den Service hat, kann scrapen,
  ohne Token-Binding. **Mitigation:** `installation.md` und der
  Manifest-Kommentar in `deploy/manifests/metrics-clusterrole.yaml`
  machen den Disclaimer explizit („In M6 ohne Wirkung, Pattern-Asset
  für künftige `kube-rbac-proxy`/FilterProvider-Aktivierung — siehe
  v0.2-ADR-Folge zu `ADR 0007`"). NetworkPolicy als zusätzlicher
  Schutz ist Anwender-Pflicht (Pattern in `troubleshooting.md`).
  **Akzeptierte Konsequenz:** der unauthentisierte Endpoint ist
  controller-runtime-Default und entspricht `ADR 0007 §2.4`, das
  keinen Auth-Filter im MVP fordert; M6 hebt das nicht.

- **Prometheus-Format-Marker-Assertion in `operator-http-smoke.sh`
  ist zu spezifisch:** `controller_runtime_reconcile_total` ist eine
  konkrete controller-runtime-Metrik, die von der Library-Version
  abhängen kann. **Mitigation:** Wir matchen auf das allgemeinere
  `# HELP `-Marker-Pattern (jede Prometheus-Format-Response beginnt
  mit `# HELP` und/oder `# TYPE`-Comments). Wenn das Pattern in einer
  zukünftigen controller-runtime-Version wegfällt, ist der Smoke
  bewusst rot — das ist eine berechtigte Regression.

- **Conditions-Katalog driftet von Code** — grep ist nicht
  AST-genau: ein Substring-Match auf einen Reason-Constant könnte
  auch in einem unzusammenhängenden Code-Kommentar oder Variablen-
  Namen anschlagen, und ein Rename eines Reason-Constants ohne
  Doku-Update bleibt durch die schwache Sicht-Tiefe von grep
  potenziell unbemerkt. **Mitigation in M6:**
  (a) Der grep-Pattern in §4 Step 5 zielt **auf das M5-
      Code-Konvention-Muster** `(Reason|ConditionType)[A-Z][a-zA-Z]+
      *= *"..."` (linker Seite Constant-Name, rechter Seite String-
      Literal) — damit sind freitext-Kollisionen ausgeschlossen,
      solange die M5-Naming-Konvention eingehalten wird.
  (b) Die Doku-Seite hat Pflicht-Schema `**Reason:** <Name>` im
      Header verankert — auch das ist strukturiert, nicht freier
      Text.
  (c) `docs/user/conditions.md` enthält im Header den Pflicht-
      Hinweis „Quelle der Wahrheit:
      `internal/adapter/check/{name}.go` + `internal/hexagon/
      application/runner.go`; bei neuen Reasons hier ergänzen".
  **v0.2-Übergabe:** sobald die Reason-Liste > ~50 Einträge oder
  externe Service-Checks (`LH-F-018..021`) neue Reasons einführen,
  wird der grep durch einen `go/parser`-Audit-Test im
  `internal/audit/markers/`-Mini-Package (M5-Pattern, siehe
  `rbac_consistency_test.go`) ersetzt. Der Wechsel ist nicht
  M6-Scope, weil das Mini-Package erst nach Reason-Listen-Wachstum
  die manuelle Pflege ablöst — vorher ist der Audit-Test Overhead
  ohne Schutzwirkung.

- **`docs/user/cr-examples.md` deckt nur zwei Profile, nicht alle
  Profile-Permutationen:** `evaluation` und `production` sind aktuell
  die einzigen MVP-Profile (`LH-PROF-002`/`LH-PROF-003`); `custom`
  und `k3s`/`scs`/`airgapped` sind v0.2+ (`LH-PROF-001`/`-004`).
  **Mitigation:** Der Header in `cr-examples.md` benennt den Scope
  explizit; v0.2-Profile bekommen eigene Beispiel-Einträge, sobald
  sie implementiert sind.

- **Anwender bindet die mitgelieferte `metrics-ClusterRole` an einen
  zu permissiven ServiceAccount:** unsere Rolle ist auf
  `nonResourceURLs: ["/metrics"], verbs: ["get"]` minimiert, aber
  Anwender könnten sie an `cluster-admin` binden. **Mitigation:**
  `docs/user/installation.md` zeigt das saubere Pattern (Bind nur an
  den konkreten Prometheus-/VictoriaMetrics-/Vector-ServiceAccount).
  Kein technisches Lock — Anwender-Pflicht.

- **`Interval` und controller-runtime-Watch-Trigger doppeln sich:**
  controller-runtime triggert ohnehin bei jeder CR-Mutation einen
  Reconcile. `RequeueAfter: 5m` heißt dann „spätestens nach 5 min
  erneut, früher wenn ein Watch reinkommt". **Risiko:** kein
  Verhaltensbruch — `controller-runtime`-Scheduler dedupliziert
  korrekt. **Mitigation:** in `docs/user/conditions.md` erklären,
  dass das Intervall ein **Maximum**, kein Fixwert ist.

- **`make manifests`-Lauf auf einer Maintainer-Maschine ohne
  `controller-gen` schlägt fehl:** das M6-Code-Wachstum braucht
  einen `make manifests`-Lauf (CRD-Schema-Drift wegen
  `Spec.Interval`). **Mitigation:** Makefile-Target ruft
  `controller-gen` über Docker (m-trace-Pattern); CI-Pipeline kann
  das selbst tun. Bei lokal-fail: README-Hinweis auf
  `controller-gen`-Version aus M2-Closure.

---

## 10. Closure

Wird beim Slice-Abschluss befüllt (analog M5 §10): Geliefertes Datei-
Set mit Commit-Map, Verifikations-Ergebnis pro §7-Item, CI-Run-URLs,
Out-of-Scope-Übergaben an M7/v0.2, Lessons learned.

**Pflicht-Formulierungen für die Closure-Notiz** (damit
Abnahme-Sprache nicht zwischen Plan und Closure driftet):

- **§7 #11** Closure-Eintrag muss die RBAC-Wirkung als
  „**Pattern-Asset ausgeliefert; funktional ungeschützt — Auth-/
  RBAC-Absicherung Übergabe an v0.2 mit eigener ADR-Folge zu
  `ADR 0007`**" formulieren. **Nicht** zulässig: „RBAC vorhanden",
  „RBAC eingerichtet", „RBAC passend" — diese Formulierungen
  suggerieren funktionale Durchsetzung, die in M6 nicht stattfindet
  (§2.2.1).
- **§7 #4** Closure-Eintrag muss den realen Coverage-Wert nennen
  (analog M5: „94.7 %"), nicht nur „grün bei 90 %". Die in §2.5
  genannte Erwartung ist Prognose, nicht Abnahme.
- **§7 #13** Closure-Eintrag muss die `Interval`-Entscheidung
  explizit referenzieren: „§2.3.1-Klassifikations-Regel (Parse-
  Erfolg + `< min/> max` → clamp; Parse-Fehler → Default) ist im
  Code 1:1 implementiert; `-5m` clamped auf `30s`, nicht auf
  Default `5m`."

Die Pflicht-Formulierungen sind Konsequenz der Plan-Review-Befunde
(Round 3, 2026-05-19) und stellen sicher, dass die Closure-Sprache
mit der bewussten Scope-Begrenzung in §1 + §2.2.1 + §8 + §2.3.1
konsistent bleibt.
