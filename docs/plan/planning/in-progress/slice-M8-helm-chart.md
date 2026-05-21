# Slice M8 — Helm-Chart als Distributions-Pfad

**Status:** In Progress
**Eröffnet:** 2026-05-21
**Geschlossen:** —
**Vorgänger:** [Slice M7 — Release v0.1.0](../done/slice-M7-release-v0.1.0.md) (v0.1-Closure-Slice)
**Nachfolger:** M9 — Kubernetes-Events bei Phasen-Übergängen
**Bezug:** [Roadmap §3 M8](roadmap-0.2.md),
[`spec/lastenheft.md` `LH-NF-016`, `LH-SST-010`, `LH-PRI-002`](../../../../spec/lastenheft.md),
[`spec/architecture.md` §3 `AR-001..AR-005`, §8 `AR-019..AR-022`](../../../../spec/architecture.md),
[ADR 0005 §4](../../adr/0005-helm-chart-nicht-im-mvp.md),
[ADR 0011 §2.5](../../adr/0011-governance-und-beitragskonventionen.md),
[ADR 0012 §2.1–§2.11](../../adr/0012-quality-gates.md),
[ADR 0013](../../adr/0013-cluster-smoke-platform.md),
[ADR 0014 §2.7](../../adr/0014-v0.2-scope-schnitt.md)

---

## 1. Lieferziel

Erster v0.2-Distributionspfad: ein **Helm-Chart** unter
`deploy/charts/k-deskflight/`, das die bestehenden Manifeste aus
`deploy/manifests/` als Templates kapselt, anwenderkonfigurierbar
über `values.yaml` + `values.schema.json` macht und über
`make helm-lint`/`make helm-template`-Targets quality-gated wird.

Cluster-Smoke (`ADR 0013`) wird um einen zweiten Install-Pfad
erweitert: zusätzlich zum bestehenden
`kubectl apply -f deploy/manifests/` läuft ein `helm install`-Pfad
parallel und attestiert dasselbe Funktionsset (CRD-Install,
Operator-Startup, fünf MVP-Checks, `/metrics`-Scrape).

Eine **Folge-ADR `ADR 0015`** entscheidet die Distributions-Form
(traditionelles Helm-Repository vs. OCI-Registry über GHCR) im
Rahmen dieses Slices — `ADR 0005 §4` hat sie als „kommt mit der
v0.2-Roadmap" angemerkt; M8 löst diese offene Stelle.

**Kein fachlicher Code-Eingriff** am Operator selbst — der Helm-Chart
ist reine Distributions-Verpackung. Die einzige Code-Veränderung
betrifft die Image-Tag-Konvention in `values.yaml.image.tag`
(default: leerer String, fällt auf `Chart.appVersion` zurück).

---

## 2. Slice-Entscheidungen

### 2.1 Chart-Layout unter `deploy/charts/k-deskflight/`

Pfad fixiert: `deploy/charts/k-deskflight/` (singular, unter dem
bestehenden `deploy/`-Schirm; konsistent mit `deploy/manifests/`
und `deploy/samples/`). Damit ist der Chart-Name kanonisch
`k-deskflight`.

Chart-Layout:

```text
deploy/charts/k-deskflight/
├── Chart.yaml
├── values.yaml
├── values.schema.json
├── README.md
├── templates/
│   ├── _helpers.tpl
│   ├── crd.yaml
│   ├── namespace.yaml         (optional, via values.namespace.create)
│   ├── serviceaccount.yaml
│   ├── clusterrole.yaml
│   ├── clusterrolebinding.yaml
│   ├── role.yaml
│   ├── rolebinding.yaml
│   ├── deployment.yaml
│   ├── service.yaml           (Metrics-Service `LH-AK-013`)
│   ├── metrics-clusterrole.yaml  (Pattern-Asset für Prometheus-Operator)
│   └── NOTES.txt
└── .helmignore
```

### 2.2 Template-Generierung aus `deploy/manifests/`

Die bestehenden Manifeste aus `deploy/manifests/` (M2/M5/M6/M7-Output)
sind die **kanonische Quelle**. Die Helm-Templates leiten ihre Struktur
1:1 ab und werden als gerenderte Variante mit
`{{ .Values…}}`-Substitutionen versehen.

**Ausnahme: Image-Tag-Pin.** Der Operator-Image-Tag in
`deploy/manifests/deployment.yaml` folgt dem letzten veröffentlichten
Release-Tag. Wenn `Chart.yaml.appVersion` ein neueres Release referenziert,
gewinnt **die Chart-Seite** — beide Werte werden zusammen gehoben (idealerweise
im Release-Slice, vgl. M16 für v0.2.0). Vor Step 5 lag der Manifest-Tag noch
auf `:dev` (M2-Platzhalter); Step 5 hat ihn auf `:v0.1.0` gehoben, weil das
v0.1.0-Release ausgeliefert ist (vgl. step-5-Review M-3).

**Doppelpflege-Strategie:** `deploy/manifests/` bleibt parallel zum
Chart als roher Install-Pfad bestehen (`ADR 0005 §3` —
„Anwender, die im MVP zwingend Helm benötigen, können das
Manifest-Set über `kubectl apply -k` (Kustomize) oder eigene
Wrapper installieren"). In v0.2 öffnen wir den Helm-Chart **zusätzlich**;
beide Pfade müssen synchron bleiben. Ein neues Quality-Gate
`make helm-manifests-sync` rendert `helm template` mit Default-
`values.yaml` und vergleicht das Resultat strukturell mit
`deploy/manifests/` — bei Drift schlägt das Gate fehl (Pattern
aus `ADR 0013`-Smoke-Stubs übertragen).

### 2.3 `values.yaml`-Schema und Konfigurierbarkeit

`values.yaml`-Felder (Default-Werte in Klammern):

```yaml
namespace:
  create: true          # (true) Namespace-Resource selbst rendern
  name: k-deskflight-system

image:
  repository: ghcr.io/pt9912/k-deskflight
  tag: ""               # ("") leer → fällt auf Chart.appVersion zurück
  pullPolicy: IfNotPresent

operator:
  mode: cluster-wide        # AR-016/AR-017: cluster-wide | namespace-scope
  replicas: 1
  leaderElect: true
  expectedReplicaCount: 1   # AR-026 Single-Pod-Topologie-Guard
  resources:
    requests: { cpu: 100m, memory: 128Mi }
    limits:   { cpu: 500m, memory: 256Mi }

metrics:
  enabled: true
  port: 8080
  service:
    type: ClusterIP
  clusterRolePattern:
    create: true        # Pattern-Asset für Prometheus-Operator
                        # (v0.1 explizit unauthenticated, `ADR 0007 §3`)

rbac:
  create: true

serviceAccount:
  create: true
  name: ""              # leer → vom Helm-Release-Namen abgeleitet
  annotations: {}

crd:
  install: true         # (true) CRD via Helm rendern;
                        # auf false setzen wenn CRD separat verwaltet wird
```

Begründung der Auswahl:
- **`namespace.create`** — operative Wahl zwischen „Helm verwaltet
  Namespace" und „Operator-Namespace existiert via GitOps".
- **`image.tag` leer als Default** — verhindert versehentliches
  Pinning auf eine veraltete Version; `Chart.appVersion` ist die
  Default-Quelle.
- **`crd.install`** — Standardpraxis für Operator-Charts; ermöglicht
  GitOps-Workflows mit zentralem CRD-Management.
- **`operator.leaderElect` + `expectedReplicaCount`** — bewusst exponiert,
  weil `AR-026` einen Single-Pod-Topologie-Guard fordert (Multi-Replica-
  HA-Tuning ist v0.2-out-of-scope, aber das Feld muss da sein, damit
  Anwender später Replicas konfigurieren können ohne Chart-Upgrade
  zu warten).
- **`operator.mode`** — implementiert das `AR-016`/`AR-017`-Versprechen,
  Namespace-Reconcile-Scope Mode „ab v0.2 zusätzlich per Helm-Values"
  zu exponieren. `cluster-wide` (Default) rendert nur ClusterRole +
  ClusterRoleBinding (v0.1-Default); `namespace-scope` rendert
  zusätzlich Role + RoleBinding im Operator-Namespace und setzt
  `--namespace=<namespace.name>` an den Deployment-Args. Die ClusterRole
  bleibt in beiden Modi aktiv für cluster-weite Read-Ressourcen
  (`nodes`, `storageclasses`, `ingressclasses`, CRDs).

### 2.4 `values.schema.json` — JSON-Schema-Validierung

`values.schema.json` validiert die `values.yaml`-Struktur clientseitig
(Helm 3+ supportet das nativ). Pflicht-Validierungen:

- `image.repository`: non-empty string
- `image.pullPolicy`: enum `{IfNotPresent, Always, Never}`
- `operator.mode`: enum `{cluster-wide, namespace-scope}`
- `operator.replicas`: integer ≥ 1
- `operator.expectedReplicaCount`: integer ≥ 1
- `metrics.service.type`: enum `{ClusterIP, NodePort, LoadBalancer}`
- `namespace.name`: Kubernetes-DNS-Label-Constraint (Lowercase
  alphanumerisch, Bindestriche im Inneren erlaubt; entspricht
  RFC 1123 Label) — als JSON-Schema-`pattern` im Schema-File
  selbst hinterlegt, nicht in dieser Prosa-Doku

Damit fängt `helm install` Fehler vor dem Server-Round-Trip ab.

### 2.5 Helm-Quality-Gate

Zwei neue `make`-Targets, eingehängt in den `make gates`-Pfad
(`ADR 0012 §2.1` zählt `make gates` als Sammel-Eintrag — M8 erweitert):

- **`make helm-lint`** — `helm lint deploy/charts/k-deskflight/`
  über das Helm-Build-Image laufen lassen. Verifiziert Chart-
  Syntax, Schema-Konformität und Best-Practice-Hinweise.
- **`make helm-template`** — `helm template` mit Default-`values.yaml`
  + drei Test-Values-Overlays (siehe §4 Reihenfolge): rendert, prüft
  YAML-Wohlgeformtheit über `kubectl --dry-run=client -o yaml`-Parse.
- **`make helm-manifests-sync`** — siehe §2.2: strukturelle
  Drift-Detektion zu `deploy/manifests/`. **Step-2-Review-Heads-up:**
  das Sync-Gate muss Helm-Meta-Labels (`helm.sh/chart`,
  `app.kubernetes.io/version`, `app.kubernetes.io/managed-by`)
  explizit von der Vergleichsbasis exkludieren — der Chart fügt sie
  als zusätzliche Labels in Pod-Template und Resource-Metadaten ein,
  die Manifeste tragen sie nicht. Ohne Exklusion läuft das Gate rot.

Optional, **nicht** in M8: `chart-testing` (`ct lint`/`ct install`)
und `helm-docs`. Beide sind Quality-of-Life-Tools; ihre Aktivierung
wird auf M16 (Release-Slice) verschoben, sobald die Distributions-
Form geklärt ist.

### 2.6 Versionierung in `Chart.yaml`

- **`Chart.yaml.version`** — Chart-Schema-Version, folgt eigenständigem
  SemVer. Initial: `0.1.0` (erster Chart, Schemaversion 0.1).
- **`Chart.yaml.appVersion`** — Operator-Image-Version, folgt der
  Operator-SemVer. Initial: `0.1.0` (zeigt erstmal auf v0.1.0-Operator,
  bis M16 den `v0.2.0`-Tag bumpt).

Die beiden Versionen sind bewusst entkoppelt: ein Chart-Patch
(z. B. `values.yaml`-Default-Korrektur ohne Operator-Änderung)
bumpt nur `Chart.yaml.version`, nicht `appVersion`. M16 bumpt beide
auf `0.2.0`, sobald der Operator-Tag steht.

### 2.7 Cluster-Smoke-Erweiterung um `helm install`-Pfad

`scripts/cluster-smoke.sh` (M2-Origin, in M6 erweitert) erhält einen
zusätzlichen Modus `INSTALL_MODE=helm` (vs. `INSTALL_MODE=manifests`,
heutiger Default). Im `helm`-Modus:

1. `kind` provisioniert (wie heute).
2. `helm install k-deskflight deploy/charts/k-deskflight/
   --namespace k-deskflight-system --create-namespace
   --set image.tag=<smoke-tag>` ersetzt das `kubectl apply -f
   deploy/manifests/`.
3. Restliche Smoke-Schritte (CRD-Wait, Operator-Pod-Wait,
   Sample-CR-Apply, Phase-Wait, `/metrics`-Scrape, vier failed-CR-
   Szenarien aus M6) bleiben unverändert.

Beide Modi (`manifests` und `helm`) laufen im CI als separate
Matrix-Jobs (parallel); lokale Entwickler können wahlweise einen
oder beide ausführen. Die Co-Failure-Gegenprobe aus M6 (`docs/plan/
planning/done/slice-M6-metrics-tests-doku.md §4 Step-Round-2`)
gilt für beide Modi.

### 2.8 Distributions-Form-Folge-ADR (`ADR 0015`)

`ADR 0005 §4` hat die Wahl „traditionelles Helm-Repository vs.
OCI-Registry über GHCR" als Folge-Themen-Slot offen gelassen. M8
löst diese Stelle mit einer **neuen ADR `0015 — Helm-Chart-
Distributions-Form`** ein.

**Entscheidungs-Vorgriff (Slice-intern, ADR muss noch geschrieben):**
OCI-Registry über GHCR ist der vermutete Vorzug, weil
- der Container-Image-Pfad bereits über `ghcr.io/pt9912/k-deskflight`
  läuft (`slice-M7 §10.2 §10.5 #11b`);
- OCI-Registries sind seit Helm 3.8 stabile Default-Distribution;
- ein separates Helm-Repository (z. B. `helm.k-deskflight.io`) wäre
  zusätzliche Infrastruktur ohne Anwendervorteil;
- GHCR unterstützt anonymes Pull, was `slice-M7 §10.5 #11b`
  für das Image bereits abgesichert hat.

Der finale ADR-Text mit Optionen/Trade-offs entsteht in §4
Reihenfolge **vor** dem Chart-Publish-Schritt.

---

## 3. Datei-Inventar

| Pfad | Art | Quelle |
| ---- | --- | ------ |
| `deploy/charts/k-deskflight/Chart.yaml` | neu | Helm-Standard |
| `deploy/charts/k-deskflight/values.yaml` | neu | §2.3 |
| `deploy/charts/k-deskflight/values.schema.json` | neu | §2.4 |
| `deploy/charts/k-deskflight/README.md` | neu | Chart-Anwender-Doku |
| `deploy/charts/k-deskflight/.helmignore` | neu | Helm-Standard |
| `deploy/charts/k-deskflight/templates/_helpers.tpl` | neu | Standard-Helper-Funktionen (`name`, `fullname`, `labels`) |
| `deploy/charts/k-deskflight/templates/crd.yaml` | abgeleitet | aus `deploy/manifests/crd.yaml` |
| `deploy/charts/k-deskflight/templates/namespace.yaml` | abgeleitet | aus `deploy/manifests/namespace.yaml` |
| `deploy/charts/k-deskflight/templates/serviceaccount.yaml` | abgeleitet | aus `deploy/manifests/rbac.yaml` |
| `deploy/charts/k-deskflight/templates/clusterrole.yaml` | abgeleitet | aus `config/rbac/role.yaml` (controller-gen-Output, `AR-015` MVP-Minimum) |
| `deploy/charts/k-deskflight/templates/clusterrolebinding.yaml` | abgeleitet | aus `deploy/manifests/clusterrolebinding.yaml` |
| `deploy/manifests/role.yaml` | **neu** | M8 schließt die `AR-016`-Lücke aus v0.1. Namespaced Role mit `AR-016 §1028`-Verben (`opendeskpreflightchecks` + `/status`); nicht im Default-`kustomization.yaml`, weil Cluster-Wide-Default unverändert bleibt. |
| `deploy/manifests/rolebinding.yaml` | **neu** | M8, dito. Bindet die Role an den Operator-SA im Operator-Namespace. Auch nicht im Default-`kustomization.yaml`. |
| `deploy/charts/k-deskflight/templates/role.yaml` | abgeleitet | aus `deploy/manifests/role.yaml`; conditional auf `values.operator.mode == "namespace-scope"` |
| `deploy/charts/k-deskflight/templates/rolebinding.yaml` | abgeleitet | aus `deploy/manifests/rolebinding.yaml`; conditional dito |
| `deploy/charts/k-deskflight/templates/deployment.yaml` | abgeleitet | aus `deploy/manifests/deployment.yaml` |
| `deploy/charts/k-deskflight/templates/service.yaml` | abgeleitet | aus `deploy/manifests/service.yaml` (M6) |
| `deploy/charts/k-deskflight/templates/metrics-clusterrole.yaml` | abgeleitet | aus `deploy/manifests/metrics-clusterrole.yaml` (M6 Pattern-Asset) |
| `deploy/charts/k-deskflight/templates/NOTES.txt` | neu | Anwender-Hinweise nach Install |
| `Makefile` | erweitert | neue Targets `helm-lint`, `helm-template`, `helm-manifests-sync`, `chart-sync-crd` |
| `scripts/helm-manifests-sync.sh` | neu | Drift-Gate-Skript, läuft im helm-tools-Container |
| `scripts/chart-sync-crd.sh` | neu | Sync der CRD-Inline-Copy nach `make manifests` (slice-M8 step-5-Review H-2) |
| `Dockerfile` (`tools`-Stage) | erweitert | `helm` und `kubectl` bereits vorhanden; ggf. Version-Pin nachjustieren |
| `scripts/cluster-smoke.sh` | erweitert | `INSTALL_MODE=helm` (§2.7) |
| `.github/workflows/ci.yaml` | erweitert | zweite Matrix-Dimension `install-mode: [manifests, helm]` für Cluster-Smoke-Job |
| `docs/plan/adr/0015-helm-chart-distributions-form.md` | neu | §2.8 Folge-ADR |
| `docs/user/installation.md` | erweitert | neuer §-Block „Installation via Helm-Chart" |
| `CHANGELOG.md` | erweitert | `[Unreleased] Added`: Helm-Chart |
| `docs/plan/planning/in-progress/roadmap-0.2.md` | erweitert | Slice-Status-Anriss in §2 mit Closure-Link beim M8-Abschluss |

---

## 4. Reihenfolge der Umsetzung

1. **Chart-Skelett anlegen.** `Chart.yaml`, `values.yaml`,
   `values.schema.json`, `.helmignore`, `templates/_helpers.tpl`,
   `templates/NOTES.txt`. `helm lint deploy/charts/k-deskflight/`
   läuft grün auf Skelett-Ebene.
2. **Templates aus `deploy/manifests/` ableiten.** Pro Manifest-
   Datei ein Template; `{{ .Values… }}`-Substitution für die in
   §2.3 ausgewiesenen Felder. Pro Schritt
   `helm template deploy/charts/k-deskflight/ | kubectl apply
   --dry-run=client -f -` zur Wohlgeformtheits-Prüfung.
3. **Drei Test-Values-Overlays.** `deploy/charts/k-deskflight/
   test-values/{default,minimal,full}.yaml` als Test-Eingaben für
   `helm-template`:
   - `default.yaml` — leeres File (alle Werte aus `values.yaml`).
   - `minimal.yaml` — `crd.install: false`, `namespace.create: false`
     (GitOps-Szenario).
   - `full.yaml` — alle Felder explizit gesetzt (Coverage-Maximum).
4. **`make`-Targets in `Makefile`**. Strukturelle Aktivierung der
   drei Helm-Gates:
   - `helm-lint` (existiert seit Step 1) → in `make gates` einhängen.
   - `helm-template` (existiert seit Step 3) → in `make gates`
     einhängen.
   - `helm-manifests-sync` → als Target neu anlegen, ruft
     `scripts/helm-manifests-sync.sh` auf. Skript ist in Step 4
     bewusst nur ein **Stub** (exit 1 mit klarer Hinweis-Message,
     Skript-Header dokumentiert den geplanten Algorithmus). **Bewusst
     NICHT** in `make gates` aufgenommen, weil CI sonst transient rot
     wird; Einhängung in `gates` erst in Step 5 nach Grün-Stand.
   - Alle vier neuen Helm-Targets (`helm-tools-image`, `helm-lint`,
     `helm-template`, `helm-manifests-sync`) landen in der
     `.PHONY`-Liste.
   - CI-Workflow (`.github/workflows/ci.yml`) ruft weiterhin
     `make gates` ohne Einzelschritt-Auflistung; der Job-Name in
     Z. 30 wird mitgepflegt, damit das GitHub-Actions-Pane den
     erweiterten Scope sichtbar zeigt.

   **Step-4-Review-Heads-up für Step 5:** das Stub-Skript läuft
   in Step 4 direkt auf dem Host (nur `cat` + `exit`). Sobald
   Step 5 das Skript um `helm`/`kubectl`/`yq` erweitert, muss
   der Aufruf auf das Pattern `docker run … $(IMAGE):helm-tools
   bash scripts/helm-manifests-sync.sh` umgestellt werden — sonst
   bricht K-2 (Docker-only). Wahrscheinlich ist die `helm-tools`-
   Stage um `kubectl` und `yq` zu erweitern oder eine eigene
   `chart-sync-tools`-Stage anzulegen. Entscheidung in Step 5.
5. **`make helm-manifests-sync` zum Grün-Stand bringen.** Wenn
   Drift gegen `deploy/manifests/` auftritt: zuerst Drift-Ursache
   verstehen, dann entscheiden ob Template oder Original-Manifest
   die kanonische Quelle ist. Default-Vorrang: `deploy/manifests/`
   (existiert seit M2/M5/M6).

   **Step-5-Closure-Notiz (2026-05-21):**
   - `helm-tools`-Stage um `kubectl` (für `kubectl kustomize`) und
     `yq` (für YAML-Normalisierung) erweitert. Pin-Hebung Routine
     ohne ADR (`YQ_VERSION ?= v4.44.5`). yq als Bare-Binary über
     HTTPS — Tarball-Variante existiert nicht für alle Releases,
     gleiches Vertrauensmodell wie kind in der smoke-Stage.
   - Skript `scripts/helm-manifests-sync.sh` rendert beide Seiten
     (`helm template` + `kubectl kustomize --load-restrictor=
     LoadRestrictionsNone`), normalisiert über yq (Helm-Meta-Labels
     drop, controller-gen-Annotation drop, `creationTimestamp` drop,
     leere/null-Docs filtern, Map-Keys rekursiv sortieren, Resources
     nach (kind, namespace, name) sortieren), diff -u.
   - Vier Drift-Befunde im Erst-Lauf, alle vor Aktivierung gelöst:
     1. Duplikat-Label `app.kubernetes.io/component: metrics-scrape`
        im Chart — fix: Labels in `templates/metrics-clusterrole.yaml`
        inline statt über den `k-deskflight.labels`-Helper.
     2. Chart-`ClusterRole` hatte Labels, controller-gen-Output nicht
        — fix: Labels in `templates/clusterrole.yaml` entfernt mit
        Kommentar zur AR-007-Begründung.
     3. Image-Tag-Drift `:dev` (Manifest-Platzhalter) vs.
        `:0.1.0` (Chart-appVersion) — fix: `deploy/manifests/
        deployment.yaml` auf `:v0.1.0` gehoben (Operator ist seit
        2026-05-20 released, Platzhalter nicht mehr nötig); Helper
        `k-deskflight.imageRef` brückt zusätzlich die appVersion-vs-
        Image-Tag-`v`-Präfix-Konvention (Helm: ohne v, GHCR: mit v).
     4. Leerer Leading-Doc (helm-Conditional-Render-Artefakt) — fix:
        `map(select(. != null and (.kind // "") != ""))` in der
        yq-Sort-Phase filtert null/empty docs.
   - `helm-manifests-sync` jetzt in `make gates` aufgenommen
     (Position hinter `helm-template`); CI-Job-Name in
     `.github/workflows/ci.yml` Z. 30 mitgezogen.
   - **Step-5-Review-Folgen umgesetzt:**
     - H-2: `make chart-sync-crd` jetzt als Target angelegt
       (Skript: `scripts/chart-sync-crd.sh`), Failure-Header von
       `helm-manifests-sync` erwähnt den Sync-Befehl explizit.
     - M-1: yq sha256-Verifizierung über das `checksums`+
       `checksums_hashes_order`-Asset-Paar des yq-Releases
       (robust gegen Spaltenfolge-Änderungen).
     - M-2: Load-Restrictor-Bypass mit präziserer Begründung
       kommentiert (in-repo, container-isoliert, alternative-
       Indirektion ohne Sicherheitsgewinn).
     - M-3: §2.2 um Image-Tag-Pin-Ausnahmeklausel ergänzt.
     - N-1: `[[ -s "$output" ]]`-Sanity-Check nach `normalize`.
     - N-2: Sort-Tiebreaker-Begründung als Inline-Kommentar.
   - **Deferred Future-Concern (Step-5-Review H-1):**
     Die Normalisierungs-Pipeline droppt heute nur eine controller-
     gen-Annotation (`controller-gen.kubebuilder.io/version`).
     Wenn die CRD-Marker in v0.2 zusätzliche generierte
     Annotationen produzieren (z. B. `api-approved.kubernetes.io`),
     kippt das Gate stumm in False-Positive. Refactor auf
     Whitelist-Modell als Trigger-Datei unter
     [`../open/controller-gen-annotation-whitelist.md`](../open/controller-gen-annotation-whitelist.md)
     verankert (M16-Pin wurde im Step-6-Review-Nachgang verworfen,
     weil Release-Slice nicht Polish-Sammler werden darf).
6. **Cluster-Smoke-Erweiterung.** `scripts/cluster-smoke.sh`
   `INSTALL_MODE=helm` umsetzen; CI-Matrix-Dimension
   `install-mode: [manifests, helm]` ergänzen. Beide laufen parallel
   grün.

   **Step-6-Closure-Notiz (2026-05-21):**
   - `Dockerfile` smoke-Stage: helm wird via `COPY --from=helm-tools
     /usr/local/bin/helm` aus der helm-tools-Stage geholt — Single-
     Source-of-Truth-Pinning, keine Helm-Version-Duplikation zwischen
     den zwei Tooling-Images.
   - `Makefile`: `INSTALL_MODE ?= manifests` als neue Variable;
     `cluster-smoke` forwarded sie als env-var; `cluster-smoke-image`
     reicht jetzt `HELM_VERSION` und `YQ_VERSION` zusätzlich durch,
     damit die transitive helm-tools-Stage-Dependency mit denselben
     Pins baut.
   - `scripts/cluster-smoke.sh`: neuer `INSTALL_MODE`-Schalter mit
     Validierung (manifests|helm), Step 3+4 conditional. Im Helm-
     Pfad: `helm install k-deskflight … --create-namespace --set
     namespace.create=false --wait --timeout=120s` (Namespace-Anlage
     übernimmt Helm, Chart-Namespace-Resource ist abgeschaltet —
     sonst Kollision „rendered manifests contain a resource that
     already exists").
   - `.github/workflows/cluster-smoke.yml`: `strategy.matrix.install-
     mode: [manifests, helm]` mit `fail-fast: false`; Job-Name und
     Artefakt-Name enthalten den Modus, damit beide Parallel-Läufe
     getrennt aufgezeichnet werden.
   - Verifikation lokal: beide Modi laufen grün durch alle Steps
     (1–9d), inklusive identischer 669-Zeilen-/metrics-Output und
     Lease-Existenz für Leader-Election. Per-CR-Phase-Assertion
     identisch zwischen den Modi (1× Passed + 4× Failed mit
     erwarteten Reason-Codes).

   **Step-6-Review-Folgen umgesetzt:**
   - M-1: `OPERATOR_NAMESPACE`-Skript-Variable als zentrale Quelle;
     Helm-`--namespace`, Step-5-`-n` und (mittelbar) Cleanup-
     Referenzen nutzen sie statt hart-codierte Strings.
   - M-2: CRD-Wait-Kommentar im Helm-Pfad als „Belt-and-Suspenders"
     präzisiert (helm --wait deckt nur Deployment-Ready, nicht
     CRD-Established).
   - N-1: `--atomic` zum `helm install` für sauberere Rollback-
     Semantik bei Timeouts.
   - N-2: Doppel-Flag-Mechanik (`--create-namespace` +
     `--set namespace.create=false`) als **Smoke-spezifischer
     Workaround** mit Operations-Empfehlungen für (a) reinen
     Chart-Pfad oder (b) reinen Helm-Pfad inline kommentiert.
     Schließt einen Doku-Heads-up für Step 8 (`docs/user/
     installation.md`) auf, das Pattern explizit als nicht-
     produktions-tauglich auszugrenzen.
   - N-3: `LH-NF-016`-Eintrag (Helm-only-Attestierung) im
     Skript-Header-Listing ergänzt.
7. **`ADR 0015 — Helm-Chart-Distributions-Form` schreiben.**
   Status: Accepted. Inhalt: OCI vs. Helm-Repo (Vorgriff §2.8), Publish-
   Flow, Versions-Sync mit `Chart.yaml.version`.
8. **Doku.** `docs/user/installation.md` um Helm-Block erweitern
   (Voraussetzungen, `helm repo add`/`helm pull oci://`, Override-
   Beispiele für `values.yaml`). `deploy/charts/k-deskflight/README.md`
   anlegen. **Step-2-Review-Heads-up:** den Kombi-Fall
   `rbac.create=false` + extern verwaltete RBAC explizit beschreiben,
   damit Anwender nicht versehentlich `operator.mode=namespace-scope`
   ohne Begleit-RBAC versuchen (Schema-Constraint fängt das
   client-seitig ab, aber ein Doku-Block macht die Erwartung explizit).
9. **CHANGELOG.** Unter `## [Unreleased] Added` einen Helm-Chart-
   Eintrag ergänzen (wird mit M16-Closure in die `[0.2.0]`-Section
   verschoben).
10. **Doku-Review.** Per K-1-Konvention (siehe [K-1 in README.md](README.md)) — Doku-Steps 8 und 9
    vor dem Slice-Closure vom `code-reviewer`-Subagent reviewen.
11. **Slice-Closure-Notiz §10.** Geliefertes Datei-Set, attestierte
    `LH-*`-Kennungen, offene Stellen → M9 oder später.

---

## 5. Lastenheft-Kennungen

| Kennung | Bezug | Erfüllung in M8 |
| ------- | ----- | --------------- |
| `LH-NF-016` | „Das Produkt soll per Helm Chart installierbar sein." | Erfüllt — `helm install` als zweiter Distributions-Pfad neben `kubectl apply -f deploy/manifests/`. |
| `LH-SST-010` | „Das Produkt soll optional über ein Helm Repository installierbar sein." | Erfüllt mit Publish-Mechanik aus `ADR 0015` (OCI-Registry über GHCR, vermutlich). Distributions-Adresse: `oci://ghcr.io/pt9912/charts/k-deskflight` (final in ADR). |
| `LH-NF-017` | GitOps-Kompatibilität | Erweitert — Helm-Chart erlaubt Argo-CD / Flux ohne Wrapper. |
| `LH-AK-015` | „dokumentierte, minimale RBAC-Konfiguration" | Bleibt erfüllt — Templates rendern dasselbe RBAC wie `deploy/manifests/`. |
| `LH-PRI-002` (Helm-Chart-Punkt) | v0.2-Soll-Anforderungen | Erfüllt mit Slice-Closure. |

Neue `LH-AK-*`-Abnahmekriterien werden in M8 **nicht** ins
Lastenheft eingepflegt, weil M8 keine fachlich neue
Funktion liefert. Falls Bedarf entsteht (z. B. ein Abnahmekriterium
„Helm-Install grün gegen die in `ADR 0009` unterstützten
Kubernetes-Versionen"), wird das mit dem Slice-Plan-Update
nachgereicht.

---

## 6. Architekturartefakte

- `spec/architecture.md` §AR-019 (Dockerfile-Stages) bleibt unverändert
  — `tools`-Stage hat bereits `helm` (`slice-M2 §3` hatte das angelegt).
- `spec/architecture.md` §AR-020 (Makefile-Target-Anker) wird
  erweitert: `helm-lint`, `helm-template`, `helm-manifests-sync` ans
  bestehende Target-Set anfügen. Update mit M8-Closure.
- `spec/architecture.md` §AR-022 (Image-Tagging und -Distribution)
  bekommt einen Folgesatz, sobald `ADR 0015` steht (Chart-
  Distributions-Form mit demselben Repository-Mechanismus).
- Keine neuen `AR-*`-Kennungen; M8 ist ein Distributions-Slice
  ohne strukturelle Architekturänderung.

---

## 7. Verifikation (Abnahmekriterien)

| Kriterium | Verifikationspfad | Pflicht |
| --------- | ----------------- | ------- |
| `helm lint` grün auf Default-`values.yaml` | `make helm-lint` | ja |
| `helm template` rendert valide YAML für alle drei Test-Overlays | `make helm-template` | ja |
| `helm template` mit Default-Values matched `deploy/manifests/` strukturell | `make helm-manifests-sync` | ja |
| `helm install` im kind-Cluster bringt den Operator in `Ready`-Zustand | `make cluster-smoke INSTALL_MODE=helm` | ja |
| Sample-CR (`deploy/samples/cluster-readiness-evaluation.yaml`) erreicht Phase `Passed` über den Helm-Install-Pfad | `make cluster-smoke INSTALL_MODE=helm` (interne Schritte) | ja |
| `/metrics`-Endpoint im Helm-Install-Pfad abfragbar | Probe-Pod-Curl im Smoke (M6-Mechanik geerbt) | ja |
| CI-Matrix `install-mode: [manifests, helm]` läuft beide Pfade parallel grün | GitHub-Actions-Run | ja |
| `ADR 0015` ist `Accepted` und beschreibt die Distributions-Form | `make doc-refs` + Sichtkontrolle | ja |
| `docs/user/installation.md` hat einen Helm-Block | `make doc-refs` + Code-Review | ja |
| `CHANGELOG.md` `[Unreleased]` hat einen Helm-Chart-Eintrag | Sichtkontrolle | ja |

---

## 8. Out-of-Scope (geht in M9 oder später)

- **Subcharts** (eingebettete Abhängigkeiten wie cert-manager) — v0.2
  hat keine Abhängigkeits-Charts (cert-manager wird als
  Vorbedingung geprüft, nicht installiert; `LH-F-035`).
- **Helm-Hooks** (`pre-install`/`post-install`-Jobs) — v0.2 braucht
  keine Pre-/Post-Logik. Falls später ein CRD-Migrations-Hook
  nötig wird (`AR-OP-007`-Aktivierung, frühestens v1.0), eigene
  Folge-ADR.
- **`chart-testing` (`ct`-CLI) als Quality-Gate** — als Trigger-
  Datei unter
  [`../open/chart-testing-activation.md`](../open/chart-testing-activation.md)
  verankert. M16-Pin im Step-6-Review-Nachgang verworfen, weil
  Release-Tag-Slice nicht zum Polish-Sammler werden darf.
- **`helm-docs`-Automatisierung** der Chart-`README.md` aus
  `values.yaml`-Kommentaren — analog als Trigger-Datei unter
  [`../open/helm-docs-automation.md`](../open/helm-docs-automation.md).
- **Multi-Replica-HA-Tuning** — `expectedReplicaCount`-Feld ist
  exponiert, aber Replicas > 1 werden in v0.2 nicht getestet
  (`AR-026` v0.1-Closure dokumentiert das als v0.2-out-of-scope).
- **Backup/Restore-Mechanik für ConfigMap-Report** — gehört zu
  M10, nicht M8.
- **OpenTelemetry-Tracer-Provider-Konfiguration via `values.yaml`** —
  gehört zu M12.

---

## 9. Risiken und Mitigation

| Risiko | Mitigation |
| ------ | ---------- |
| `helm-manifests-sync` läuft kontinuierlich rot, weil Templates und Manifeste auseinanderlaufen | Default-Vorrang `deploy/manifests/` (kanonische Quelle); Drift wird via Gate früh bemerkt und im selben Commit gelöst. |
| `values.schema.json` ist zu eng und blockt legitime Anwender-Konfigurationen | Schema startet bewusst minimal (nur die in §2.4 gelisteten Felder); Erweiterung pro Bedarfsmeldung. |
| OCI-Distribution scheitert an GHCR-spezifischen Authentifizierungs-Eigenheiten | `ADR 0015` adressiert das explizit; falls OCI-Pfad in §4 Schritt 7 als nicht-tragfähig auffällt, fallback auf traditionelles Helm-Repository (eigenes GitHub-Pages-Hosting). |
| Cluster-Smoke `helm`-Modus läuft langsamer als `manifests`-Modus und blockt CI | Beide Modi laufen als parallele Matrix-Jobs; ein Modus-Hänger blockt nicht den anderen. Per-Modus-Timeout wird in `cluster-smoke.sh` gesetzt. |
| Chart-Version (`Chart.yaml.version`) und Image-Tag (`appVersion`) divergieren versehentlich | Versions-Sync-Note in `Chart.yaml`-Kommentar; M16 prüft beide Felder im Release-Guard. |
| `Subchart`-Wunsch entsteht erst v0.3, dann muss `Chart.yaml.apiVersion: v2` ohnehin schon stehen | `apiVersion: v2` von Anfang an gesetzt (Helm 3 Default); `dependencies:`-Block bleibt leer. |

---

## 10. Closure (—)

> Diese Sektion wird mit dem Slice-Abschluss befüllt.
> Erwartete Inhalte (analog `slice-M1-repo-skeleton.md §10`):
> - 10.1 Geliefertes Datei-Set
> - 10.2 Attestierte `LH-*`-Kennungen und Abnahmekriterien aus §7
> - 10.3 Aktualisierung von Roadmap-§2 (M8-Status auf Done)
> - 10.4 Lessons-Learned (Helm-Build-Toolchain-Eigenheiten,
>   Drift-Pattern, Cluster-Smoke-Modus-Erweiterung)
> - 10.5 Post-Closure-Folgeschritte (Helm-Chart-Publish unter M16,
>   `chart-testing`-Aktivierung)
