# Slice M7 — Beispielmanifest, Release-Tag v0.1.0

**Status:** In Progress
**Eröffnet:** 2026-05-20
**Geschlossen:** —
**Vorgänger:** [M6 — Metrics-Endpoint, Tests, Doku (Done)](../done/slice-M6-metrics-tests-doku.md)
**Nachfolger:** v0.2-Roadmap (entsteht mit M7-Closure als Sammel-Notiz; M7 ist der letzte MVP-Slice)
**Bezug:**
[Roadmap §3 M7](../../roadmap.md#m7--beispielmanifest-release-tag-v010),
[`spec/architecture.md` §8 (AR-019, AR-020, AR-021), §AR-022, §AR-026](../../../../spec/architecture.md),
[ADR 0005](../../adr/0005-helm-chart-nicht-im-mvp.md),
[ADR 0009](../../adr/0009-k8s-versions-support-und-profile-mindestversionen.md),
[ADR 0011 §2.4–§2.6](../../adr/0011-governance-und-beitragskonventionen.md),
[ADR 0012 §2.8–§2.9](../../adr/0012-quality-gates.md),
[CHANGELOG-Trigger](../open/changelog.md)

---

## 1. Lieferziel

M7 ist der **Release-Slice** für MVP `v0.1.0` (`LH-REL-001`,
`LH-MVP-002`). Lieferziele in einem Satz: Anwender-fertige
Beispielmanifeste, eine erste `CHANGELOG.md`, das `image-publish`-/
`release-guard`-/`image-scan`-Tooling als adaptiertes m-trace-Pattern,
`--leader-elect`-Konfiguration mit Single-Pod-Topologie-Guard
(`AR-026`-Schluss), und der annotierte Git-Tag `v0.1.0` mit
Container-Image auf `ghcr.io/pt9912/k-deskflight:v0.1.0`.

**Roadmap-§3-M7-Bullets, die in diesem Slice fallen:**

- [ ] Beispielmanifeste konsistent mit `LH-PROD-003a`
  (production-Profil) und ein zweites `evaluation`-Profil-Beispiel
  parallel — Anwender ohne Repo-Checkout braucht eine direkt
  applizierbare Vorlage. §2.1.
- [ ] **CHANGELOG-Erstaufbau** nach Keep-a-Changelog 1.1.0,
  **Englisch**, manuelle Pflege pro Release. Inline-Konvention in
  diesem Slice-Plan verankert; **keine eigene ADR** — Pattern ist
  direkte m-trace-Adaption ohne langfristige Bindung außerhalb des
  Files. §2.2.
- [ ] `make image-publish`-Pattern aus m-trace: `image-build VER=`,
  `image-publish-dry-run`, `image-publish-guard`,
  `image-publish` — Approval-Gate via
  `K_DESKFLIGHT_IMAGE_PUBLISH_APPROVED=1`. §2.3.
- [ ] **Trivy Image-Scan vor Tag** (`LH-QG-007`,
  `ADR 0012 §2.9`): `CRITICAL`/`HIGH` brechen, `MEDIUM` wird
  berichtet, Vulnignore mit `expires`-Pflicht. §2.4.
- [ ] **`make release-guard`** als adaptiertes m-trace-Pattern:
  Branch-Check `main`, Dirty-Tree-Check, Tag-Vorabprüfung gegen
  `origin`, manuelle Approval-Variable, Dry-Run-Pfad. §2.5.
- [ ] **v0.1.0-Tag setzen** als annotated Git-Tag mit Release-Notes
  als Tag-Message-Body und auf GitHub als Release-Eintrag mit
  Verlinkung der Release-Notes-Datei. §2.6.
- [ ] **DCO-Compliance-Check vor Merge der Release-PR**
  (`ADR 0011 §2.4`): operative GitHub-App-Aktivierung
  (`probot/dco`) plus Closure-Notiz, dass alle Commits seit
  ADR-0011-Acceptance Sign-off tragen. §2.7.

**Was M7 zusätzlich aus M6-Übergaben übernimmt
(M6 §8 + Roadmap §3 M7):**

- [ ] **Leader-Election scharfschalten** (`AR-026`, `LH-NF-006`-
  operativ): `--leader-elect`-Flag (Default `true`),
  `OPERATOR_EXPECTED_REPLICA_COUNT`/`--expected-replica-count`-Guard
  (Default `1`), `LeaderElectionID = "k-deskflight-operator"`,
  `LeaderElectionNamespace = <Operator-Namespace>`. Single-Pod-
  Topologie-Mismatch-Guard wird beim Start hart durchgesetzt.
  RBAC für `coordination.k8s.io/leases` ist seit M2 verankert
  (`AR-015`), `controller-runtime`-Manager-Option setzt das Flag.
  §2.8.
- [ ] **Lastenheft `LH-PROD-003a` an aktuelle CRD-Feldnamen
  angleichen**: das Beispiel im Lastenheft (`spec/lastenheft.md`
  Zeile 163ff.) führt noch die M2-Pre-Feldnamen (`ingress`/
  `certManager.required`/`storage.requiredClasses`/`resources`).
  Die seit M2 in `api/v1alpha1/opendeskpreflightcheck_types.go`
  verankerten Namen sind `ingressClass.names`,
  `certManager:{}` (kein `required`-Feld), `storageClass.names`/
  `requireDefault`, `clusterResources.minCPU`/`minMemory`. M7
  passt das Lastenheft an die Implementierung an — die
  Implementierung ist die kanonische Source-of-Truth seit
  M2-Closure. §2.9.

**Was M7 nicht macht (Übergaben an v0.2 und folgende):**

- **Helm-Chart** — bewusst out-of-MVP (`ADR 0005`); v0.2 mit
  Folge-ADR zur Chart-Struktur.
- **ServiceMonitor / PodMonitor / PrometheusRule** —
  out-of-MVP per `ADR 0007 §4`; v0.2 mit Helm-Chart-Aktivierung.
- **Eigene Domänen-Metriken** (`LH-NF-008` wörtlich) — v0.2
  (`ADR 0007 §2.3`).
- **Metrics-Endpoint-Authentication** (Auth-Filter / `kube-rbac-
  proxy` / TLS-Cert-Lifecycle) — v0.2 mit Folge-ADR zu
  `ADR 0007` (M6 §8-Übergabe).
- **Externe Service-Checks** (PostgreSQL, S3, DNS, TLS — `LH-F-018..021`)
  — v0.2 / v0.3+ mit Folge-ADR-Pfad zu `ADR 0010`. Trigger lebt in
  `planning/open/external-services-v03-activation.md`.
- **Multi-Maintainer-Modell / `GOVERNANCE.md`** —
  `ADR 0011 §2.7`-Erweiterung, nicht MVP-blockierend.
- **Image-Provenance / SBOM-Publish / Sigstore-Cosign** — Pattern
  out-of-MVP; M7 liefert Trivy-Scan, SBOM-Generierung bleibt v0.2+.

---

## 2. Slice-Entscheidungen

### 2.1 Beispielmanifest-Set

**Entscheidung:** M7 liefert zwei vollständige Anwender-Beispiele
unter `deploy/samples/` (nicht `config/samples/` — das ist die
`controller-gen`-Output-Heimat):

| Datei | Profil | Zweck |
| ----- | ------ | ----- |
| `deploy/samples/cluster-readiness-production.yaml` | `production` | Default-Vorlage für Anwender, deckt alle fünf MVP-Checks ab (`LH-PROD-003a`-Pattern, mit aktuellen CRD-Feldnamen). |
| `deploy/samples/cluster-readiness-evaluation.yaml` | `evaluation` | Lower-Profile-Vorlage mit reduzierten Schwellen — Anwender kann ohne Cluster-Aufbau eine erste passed-Run-Validierung machen. |

**Verhältnis zu existierenden Sample-Dateien:**

- `config/samples/k-deskflight_v1alpha1_opendeskpreflightcheck.yaml`
  ist der **MVP-Smoke-CR** (`name: smoke`, deckt Cluster-State-Stubs
  ab; gegen kind getestet). Bleibt für Test-Pfade aktiv. M7 ergänzt
  Anwender-Samples, ersetzt das Smoke-CR nicht.
- `docs/user/cr-examples.md` (M6-Lieferung) zeigt Inline-Beispiele
  in der Anwender-Doku. M7 hebt die Inline-Beispiele auf eigene
  Dateien unter `deploy/samples/` und verlinkt aus `cr-examples.md`
  dorthin — Anwender muss CR nicht aus Markdown copy-pasten.

**Inhalte:**

- Beide Samples nutzen `apiVersion: k-deskflight.geo-terrain.net/v1alpha1`,
  `kind: OpenDeskPreflightCheck`.
- `metadata.name` deklarativ (`cluster-readiness-production`,
  `cluster-readiness-evaluation`), Namespace `default`.
- Header-Kommentare verlinken auf `docs/user/cr-examples.md`,
  `docs/user/conditions.md` und auf `ADR 0009` (Kubernetes-Min-Version-
  Defaults pro Profile).
- Production: `kubernetesVersion.min: "1.34"` (ADR-0009-Default
  production), `storageClass.names: [default, backup]`,
  `ingressClass.names: [nginx]`, `certManager: {}`,
  `clusterResources.minCPU: "16"`, `minMemory: "64Gi"`,
  `interval: "5m"`.
- Evaluation: `kubernetesVersion.min: "1.33"`
  (ADR-0009-Default evaluation), `storageClass.names: [standard]`,
  `requireDefault: false`, `ingressClass.names: [nginx]`,
  `certManager: {}`, `clusterResources.minCPU: "2"`,
  `minMemory: "4Gi"`, `interval: "5m"`.

**Kustomize-Integration:** Samples werden **nicht** in
`deploy/manifests/kustomization.yaml` referenziert — sie sind
Anwender-Vorlagen, nicht Teil der Operator-Installation. Die Datei
wird per `kubectl apply -f deploy/samples/<file>.yaml` einzeln
appliziert (Doku-Pfad in `installation.md` §6).

### 2.2 CHANGELOG: Keep-a-Changelog 1.1.0, Englisch, manuell

**Entscheidung-Begründung (`planning/open/changelog.md` §2):**

- **Format Keep-a-Changelog 1.1.0** — m-trace-Pattern, etablierte
  Konvention im Operator-/Go-Ökosystem, `## [Unreleased]`-Section
  trägt den laufenden Stand zwischen Releases.
- **Sprache Englisch** — konsistent zu `LH-NF-021` (Commit-/PR-
  Sprache ab ADR-0011-Acceptance). README, CONTRIBUTING und
  CHANGELOG sind das Anwender-/Mitwirkenden-Set, gleicher
  Sprach-Pflichtenkanon.
- **Manuelle Pflege pro Release** — kein `release-please`/
  `git-cliff`-Tooling im MVP. Begründung: das Repository hat
  bislang sieben Slices in zwei Wochen und damit überschaubares
  Commit-Volumen; ein Automat würde mehr Setup-Aufwand bringen
  als Nutzen. Bei v0.2+ kann die Konvention zu `release-please`
  wechseln (Trigger: erstes Quartal mit >50 Commits pro Release).
- **Granularität pro Release-Tag** — nicht pro Slice. Jede
  `## [vX.Y.Z] - YYYY-MM-DD`-Section listet die für Anwender
  sichtbaren Änderungen seit dem letzten Tag.
- **Inline-Konvention statt ADR** — der CHANGELOG-Format-Entscheid
  ist Routine-Setup ohne langfristige Bindung außerhalb der Datei
  selbst. Eine eigene ADR wäre Overhead. Falls v0.2+ auf
  Tooling-Pflege wechselt, entsteht dort die ADR.

**Pflicht-Erstaufnahmen aus `planning/open/changelog.md §4`:**

- **Changed** — `api/v1alpha1.OpenDeskPreflightCheckSpec.Interval`
  ist nicht mehr nullable (`*string` → `string`); Default `5m`
  greift unverändert (Commit `dc4a14d`, M6 Round-2 §4 Befund 6).
  Quellen-Anker in der Entry.

**Pflicht-Inhalte der ersten `## [0.1.0] - 2026-05-XX`-Section:**

- **Added** — kompletter MVP-Funktionsumfang (CRD, Reconciler,
  fünf Checks, RBAC-Selbstprüfung, `/metrics`-Service,
  Anwender-Doku). Pro Bullet ein Slice-Anker.
- **Changed** — der `Interval`-Bruch (siehe oben). Sonst keine,
  da dies das Erst-Release ist.
- **Security** — leer (Erst-Release ohne CVE-Historie).

**Datei-Layout:** `CHANGELOG.md` im Repository-Root,
`scripts/verify-doc-refs.sh`-Geltungsbereich erweitern (das Skript
listet `CHANGELOG.md` bereits unter den optionalen Top-Level-Dateien
in `ADR 0012 §2.10`).

### 2.3 `make image-publish`-Pattern

**Entscheidung:** Direkte Adaption des m-trace-Patterns
(`/Development/m-trace/Makefile` Zeilen 674–695), single-image-
Variante (k-deskflight hat genau ein Operator-Image, kein Multi-
Service-Set wie m-trace):

| Target | Wirkung |
| ------ | ------- |
| `make image-build VER=X.Y.Z` | Baut `ghcr.io/pt9912/k-deskflight:vX.Y.Z` (statt `:go`-Dev-Tag); erfordert `VER`. |
| `make image-publish-dry-run VER=X.Y.Z` | `image-build` + `docker image inspect`-Self-Check + Echo „would push". Kein Push. |
| `make image-publish-guard` | Verifiziert `K_DESKFLIGHT_IMAGE_PUBLISH_APPROVED=1` — sonst exit 2 mit Hinweis. |
| `make image-publish VER=X.Y.Z` | `image-publish-guard` + `image-build` + `docker push`. |

**Approval-Variable:** `K_DESKFLIGHT_IMAGE_PUBLISH_APPROVED=1`
(m-trace nutzt `MTRACE_IMAGE_PUBLISH_APPROVED=1` — gleicher
Pattern, Projekt-Prefix unterschiedlich). Dokumentiert im
`make help`-Block und in der Release-Notes-Sektion der
`CHANGELOG.md`.

**Image-Tagging-Konvention:**

- `ghcr.io/pt9912/k-deskflight:vX.Y.Z` (annotated Git-Tag-Spiegel,
  immutable nach Push).
- `ghcr.io/pt9912/k-deskflight:dev` bleibt der Deployment-
  Manifest-Default (`deploy/manifests/deployment.yaml:38`) für
  Cluster-Smoke-Workflows; Anwender überschreiben per Kustomize-
  Image-Override (siehe `installation.md` §2).
- Kein `:latest`-Tag — Anwender pinnen explizit auf SemVer.

**Bestehender `image-build`-Alias:** `Makefile` Zeile 208
(`image-build: build`) als Alias auf den `:go`-Dev-Tag bleibt für
inner-loop. M7 ersetzt das Target durch eine zweigleisige
Variante: ohne `VER` baut weiter `:go`, mit `VER` baut
`:vX.Y.Z` (m-trace-Pattern). Migration ist additiv, kein Break.

### 2.4 Trivy Image-Scan-Gate

**Entscheidung:** Adaption des m-trace-`image-scan`-Patterns
(`/Development/m-trace/Makefile` Zeilen 617–671), single-image-
Variante:

| Komponente | k-deskflight-Wert |
| ---------- | ----------------- |
| Trivy-Image-Pin | `aquasec/trivy:0.59.1` (m-trace-Match; `ADR 0012 §2.9`). |
| Scan-Target | `ghcr.io/pt9912/k-deskflight:vX.Y.Z` (post-`image-build`). |
| Severity-Policy | `CRITICAL`/`HIGH` brechen (`--exit-code 1`); `MEDIUM` wird im Stdout-Report gezeigt, blockt nicht. |
| Vulnignore-Datei | `.security/.trivyignore` (analog m-trace), per-Eintrag `expires:YYYY-MM-DD`-Kommentar verpflichtend. |
| Vulnignore-Renderer | `scripts/render-trivyignore.sh` — adaptiert von `/Development/m-trace/scripts/render-trivyignore.sh`: liest pro-Image-Quelle aus `.security/.trivyignore.in`, prüft `expires`-Datum gegen `date +%Y-%m-%d`, abgelaufene Einträge brechen den Generator (`ADR 0012 §2.8` Abs. 3). |
| Cache | `.security/.trivy-cache/` (gitignored). |
| Make-Target | `make image-scan VER=X.Y.Z` — `image-build` als Vorgänger, dann Trivy-Run. |
| Bündel | `security-gates` erweitert: `govulncheck + image-scan`. Mit `make security-gates VER=…` als Release-Gate-Bündel; ohne `VER` läuft nur `govulncheck` (Inner-Loop-Pfad bleibt schnell). |

**Erst-Run-Vulnignore:** Wir starten mit leerer
`.security/.trivyignore`. Falls der erste reale Trivy-Run
`HIGH`/`CRITICAL`-Funde liefert, die nicht via Base-Image-Bump
fix-bar sind, werden sie mit `expires`-Pflicht eingetragen und in
den Release-Notes vermerkt — nicht stillschweigend ignoriert.

### 2.5 `make release-guard`

**Entscheidung:** Adaption des m-trace-`release-guard.sh`-Skripts
(`/Development/m-trace/scripts/release-guard.sh`, 85 Zeilen Bash)
nach `scripts/release-guard.sh`. Aufrufmuster:

```text
make release-guard VER=0.1.0
```

**Checks (alle vor Tag-Erzeugung):**

1. **Approval-Variable:** `K_DESKFLIGHT_RELEASE_APPROVED=1`
   gesetzt, sonst Abbruch.
2. **Branch:** `git rev-parse --abbrev-ref HEAD == main`,
   sonst Abbruch (Override `K_DESKFLIGHT_RELEASE_ALLOW_NON_MAIN=1`
   nur für lokale Guard-Tests).
3. **Dirty-Tree:** `git status --porcelain` leer, sonst Abbruch
   (Override `K_DESKFLIGHT_RELEASE_ALLOW_DIRTY=1`).
4. **Tag-Vorabprüfung:** `git ls-remote --tags origin
   refs/tags/v$VER` — Tag darf auf `origin` noch nicht
   existieren (Override `K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE=1`
   nur für lokale Guard-Tests).
5. **Dry-Run-Pfad:** `K_DESKFLIGHT_RELEASE_DRY_RUN=1` echo „dry-run
   ok for v$VER" ohne Tag-Erzeugung.
6. **Gates-Vorabprüfung:** `make gates` + `make security-gates
   VER=$VER` (inkl. Image-Scan) müssen vor dem Tag grün sein.
   Das Skript prüft das **nicht** automatisch (würde Build-Zeit
   verdoppeln), die Reihenfolge in §4 sieht die Gates als
   eigenständigen Schritt vor dem Guard.

**`scripts/test-release-guard.sh`** (adaptiert von
`/Development/m-trace/scripts/test-release-guard.sh`): lokale
Failure-Path-Tests — pro Override-Pfad ein synthetischer
git-state-Mock, der den Guard wie erwartet zum Abbruch oder
Pass-Through zwingt. Wird im CI nicht ausgeführt (m-trace-
Konvention), nur lokal beim Anpassen des Guards.

### 2.6 Tag- und Release-Notes-Mechanik

**Tag-Form:** Annotated Git-Tag `v0.1.0` per `git tag -a v0.1.0
-m "<release-notes-summary>"`. Die Tag-Message-Body trägt eine
Zwei-Absatz-Zusammenfassung des Releases plus Verweis auf
`CHANGELOG.md` und `docs/user/installation.md`.

**Release-Notes-Datei:** Zwei-Pfad-Strategie:

1. **`CHANGELOG.md`** ist die kanonische Quelle der Release-Notes
   (eine `## [0.1.0] - YYYY-MM-DD`-Section). Wird mit dem Tag-
   Commit ausgeliefert.
2. **GitHub-Release** (`gh release create v0.1.0`): Body ist die
   `## [0.1.0]`-Section als Markdown, plus Image-Pull-Snippet
   (`docker pull ghcr.io/pt9912/k-deskflight:v0.1.0`) und einen
   Link auf `docs/user/installation.md`.

**SemVer-Pre-Release-Pfad:** `v0.1.0-rc.1` bleibt zulässig
(`ADR 0011 §2.5`), aber für MVP-`v0.1.0` ohne Pre-Release-
Iteration geplant — kein RC-Tag im aktuellen Slice. Falls Trivy
`CRITICAL`-Funde liefert, die einen Base-Image-Bump erzwingen,
wird das per Commit ohne RC-Tag adressiert.

### 2.7 DCO-Compliance

**Operativ:** `probot/dco` als GitHub-App im Repository
aktivieren (`ADR 0011 §2.4`-Pflicht). Aktivierung ist Repo-
Settings-Click und nicht im Code; Closure-Notiz dokumentiert
Datum und PR-Verifikation.

**Pre-Tag-Check:** `git log --pretty='%H %s%n%b' v0.0.0..HEAD |
grep -c "Signed-off-by:"` muss `>=` der Anzahl Commits seit
ADR-0011-Acceptance sein. Frühere Commits (vor ADR-Acceptance)
sind „grandfathered" laut `ADR 0011 §2.4` und nicht Pflicht.

**Pre-Acceptance-Marker:** ADR 0011 wurde am 2026-05-16
akzeptiert. Bis Acceptance sind ~30 Commits ohne Sign-off
zulässig; ab Acceptance gilt strikt. M7 prüft, dass alle
Code-Commits seit ADR-0011-Acceptance (`feat(skel):`, `feat(crd):`,
…) Sign-off tragen. Falls Lücken: Reviewer-Findings vor Tag.

### 2.8 Leader-Election scharfschalten (`AR-026`)

**Status heute:** `cmd/operator/main.go` ruft
`ctrl.NewManager(cfg, manager.Options{…})` **ohne**
`LeaderElection`-Felder — das aktiviert controller-runtime-
Defaults: `LeaderElection=false`. Das Deployment-Manifest
referenziert keine Leader-Flags.

**M7-Lieferungen:**

| Datei | Inhalt |
| ----- | ------ |
| `cmd/operator/main.go` | Flag-Parsing (`flag.Parse`) für `--leader-elect` (Default `true`), `--expected-replica-count` (Default `1`), `--leader-election-id` (Default `"k-deskflight-operator"`), `--leader-election-namespace` (Default per `POD_NAMESPACE`-Env oder `metav1.NamespaceDefault`). `manager.Options.LeaderElection`/`LeaderElectionID`/`LeaderElectionNamespace`/`LeaderElectionResourceLock` (`"leases"`) gemäß `AR-026`. |
| `cmd/operator/main.go` | Single-Pod-Topologie-Guard: bei `--leader-elect=false` und `--expected-replica-count=1` wird die laufende Pod-Anzahl im Operator-Namespace abgefragt (`clients.Clientset.CoreV1().Pods(ns).List` mit Operator-Label-Selector). Bei >1 aktivem Pod: Start abbrechen mit `Reason=SinglePodTopologyMismatch` (im Log; Operator hat noch keinen Pod-Status zum Schreiben). Werte >1 für `--expected-replica-count` mit `--leader-elect=false` brechen den Start ebenfalls. Werte <1 werden auf `1` normalisiert. |
| `deploy/manifests/deployment.yaml` | Container-`args`-Block ergänzen: `--leader-elect=true`. `env`-Block ergänzen: `POD_NAMESPACE` per Downward-API (`fieldRef.fieldPath: metadata.namespace`). |
| `deploy/manifests/clusterrolebinding.yaml` / `config/rbac/role.yaml` | `coordination.k8s.io/leases`-Rechte sind seit M2 via Marker an `Reconciler.Reconcile` aktiv (`AR-015`). M7 verifiziert nur die `rbac_consistency_test.go`-Passes; keine Manifest-Änderung nötig. |
| `internal/hexagon/application/leader_topology.go` (neu) | Single-Pod-Topologie-Guard als testbare Funktion: `EnforceSinglePodTopology(ctx, podsAPI corev1.PodsGetter, ns, labelSelector, expected, leaderElect) error`. Tests in `leader_topology_test.go` mit `fake.NewSimpleClientset`. |
| `scripts/cluster-smoke.sh` | Step 9d (neu): nach `operator-http-smoke.sh` Lease-Existenz-Check via `kubectl get leases -n k-deskflight-system k-deskflight-operator -o jsonpath='{.spec.holderIdentity}'` — Vorhandensein attestiert, dass Leader-Election scharf ist. |

**Disclaimer:** Multi-Replica-Aktivierung (Deployment `replicas: >1`)
bleibt v0.2-Trigger — M7 stellt nur sicher, dass das **Pattern**
korrekt verdrahtet ist und beim Default-`replicas: 1` produktiv
läuft. Multi-Pod-Active-Standby ist erst mit
`ADR 0014`-Folge-ADR scharf (v0.2-Trigger entsteht in M7-Closure).

### 2.9 Lastenheft `LH-PROD-003a`-Sync

**Befund:** `spec/lastenheft.md` Zeilen 167–194 (Beispiel-CR im
Abschnitt `LH-PROD-003a`) verwendet die in M1/M2 vorgesehenen
Feldnamen, die seit M2 in `api/v1alpha1/opendeskpreflightcheck_types.go`
**umbenannt** sind:

| Lastenheft (aktuell) | Implementation (kanonisch seit M2) | Quelle |
| -------------------- | ---------------------------------- | ------ |
| `ingress.required` / `ingress.className` | `ingressClass.names: [<class>]` | `opendeskpreflightcheck_types.go:103-105` |
| `certManager.required` | `certManager: {}` (Feld ohne `required`) | `opendeskpreflightcheck_types.go:150` (Empty-Spec-Type) |
| `storage.requiredClasses` | `storageClass.names`/`requireDefault` | `opendeskpreflightcheck_types.go:89-96` |
| `resources.minCpu` / `minMemory` | `clusterResources.minCPU` / `minMemory` | `opendeskpreflightcheck_types.go:125-132` |

**Entscheidung:** Die Implementierung ist die kanonische
Quelle — M2-Closure hat das festgeschrieben. M7 passt das
Lastenheft-Beispiel `LH-PROD-003a` an die aktuellen CRD-
Feldnamen an, in einem `docs(spec)`-Commit. `LH-PROD-003b`
(Zielbild-Beispiel ab v0.2) wird im selben Commit aktualisiert,
weil dieselbe Namens-Diskrepanz dort liegt.

**Begründung:** Anwender, die `LH-PROD-003a` als Vorlage
benutzen, würden mit der alten Form von der CRD-Schema-
Validierung abgewiesen (`spec.ingress` → unbekanntes Feld).
M7 ist der natürliche Zeitpunkt, das anzugleichen, weil hier
die Anwender-fertigen Samples entstehen.

**Lastenheft-Versionsmarker:** Lastenheft-Patch-Hebung
(`Stand: 1.x.y → 1.x.(y+1)`) per ADR-2-Mechanik — Routine-
Edit ohne ADR-Trigger.

---

## 3. Datei-Inventar

Neu im Repository:

| Pfad | Zweck | Vorlage |
| ---- | ----- | ------- |
| `CHANGELOG.md` | Keep-a-Changelog 1.1.0, Englisch, manuell. Erste `## [0.1.0]`-Section bei Tag-Commit. | `m-trace/CHANGELOG.md` |
| `deploy/samples/cluster-readiness-production.yaml` | Production-Profil-CR, alle fünf MVP-Checks aktiviert. | `LH-PROD-003a` (angepasst gemäß §2.9) |
| `deploy/samples/cluster-readiness-evaluation.yaml` | Evaluation-Profil-CR, reduzierte Schwellen. | analog production |
| `scripts/release-guard.sh` | Pre-Release-Konsistenzprüfung. | `m-trace/scripts/release-guard.sh` |
| `scripts/test-release-guard.sh` | Lokale Failure-Path-Tests für den Guard. | `m-trace/scripts/test-release-guard.sh` |
| `scripts/render-trivyignore.sh` | Vulnignore-Generator mit `expires`-Pflicht. | `m-trace/scripts/render-trivyignore.sh` |
| `.security/.trivyignore.in` | Quell-Format mit `expires`-Kommentaren. Initial leer (Header-Kommentar mit Format-Beispiel). | `m-trace/.security/.trivyignore.in` (falls vorhanden) |
| `.gitignore` (Update) | Erweitert um `.security/.trivy-cache/` und `.security/.trivyignore` (generiert). | bestehend |
| `internal/hexagon/application/leader_topology.go` | Single-Pod-Topologie-Guard für `--leader-elect=false`. | M7-eigen, `AR-026` |
| `internal/hexagon/application/leader_topology_test.go` | Tests mit `fake.NewSimpleClientset`. | M7-eigen |
| `Makefile` (Update) | Neue Targets `image-build VER=`, `image-publish-dry-run`, `image-publish-guard`, `image-publish`, `image-scan`, `release-guard`, `release-guard-test`. `security-gates` erweitert um `image-scan` wenn `VER` gesetzt. | `m-trace/Makefile` |
| `cmd/operator/main.go` (Update) | Flag-Parsing für `--leader-elect`, `--expected-replica-count`, `--leader-election-{id,namespace}`. Manager-Options entsprechend. Topologie-Guard-Aufruf. | `AR-026` |
| `deploy/manifests/deployment.yaml` (Update) | Container-`args`: `--leader-elect=true`. `env`: `POD_NAMESPACE` per Downward-API. | bestehend |
| `spec/lastenheft.md` (Update) | `LH-PROD-003a` und `LH-PROD-003b`-Beispiele auf aktuelle CRD-Feldnamen angleichen (§2.9). Versionsmarker im Header. | bestehend |
| `docs/user/cr-examples.md` (Update) | Inline-Beispiele auf `deploy/samples/<file>.yaml`-Verlinkungen umstellen. | bestehend |
| `docs/user/installation.md` (Update) | §6-Erweiterung: Default-Sample applizieren (`kubectl apply -f deploy/samples/cluster-readiness-production.yaml`). | bestehend |

**Keine Anpassung** in dieser Slice: `README.md`/`README.de.md`,
`CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`, `LICENSE`
(`LH-AK-014`-Pflicht ist seit M1 erfüllt). Das M7-Closure-Datum
trägt einen impliziten README-Stand-Marker — sollten Folge-
Reviewer-Findings README-Edits erzwingen, sind sie additive Commits
in derselben Slice-Aktivität.

---

## 4. Reihenfolge der Umsetzung

Jeder Schritt = ein eigener Commit (`ADR 0011 §2.2`). Doku-Reviews
nach K-1-Konvention (siehe `planning/in-progress/README.md`).

1. **`spec/lastenheft.md`** §2.9-Sync — `LH-PROD-003a`/`-003b`
   auf aktuelle CRD-Feldnamen. `docs(spec):`-Commit.
   `make doc-refs` grün. Kein Code-Review nötig (reine
   Doku-Sync), aber `code-reviewer`-Subagent-Run analog K-1.
2. **Beispielmanifeste** `deploy/samples/*.yaml` (zwei Dateien)
   anlegen. `docs(samples):`-Commit. Smoke-Apply gegen den
   `cluster-smoke`-Cluster: `kubectl apply -f deploy/samples/
   cluster-readiness-production.yaml`, danach
   `kubectl wait --for=condition=KubernetesVersionReady=true
   opendeskpreflightcheck/cluster-readiness-production --timeout=60s`.
   Sample-Apply via Mini-Smoke-Pfad attestieren (siehe §7 #2).
3. **`docs/user/cr-examples.md`** und **`installation.md`**
   Updates: Verlinkungen auf neue Sample-Dateien. K-1-Doku-
   Review verpflichtend.
4. **Leader-Election scharfschalten** (`AR-026`-Schluss):
   - `cmd/operator/main.go`: Flag-Parsing + Manager-Options.
   - `internal/hexagon/application/leader_topology.go` + Tests.
   - `deploy/manifests/deployment.yaml`: `--leader-elect=true`-arg
     und `POD_NAMESPACE`-Downward-API.
   - `scripts/cluster-smoke.sh`: Step 9d Lease-Check.
   - `feat(operator):`-Commit. `make gates` + `make cluster-smoke`
     grün.
5. **`Makefile`** erweitern: `image-build VER=`, `image-publish-
   dry-run`, `image-publish-guard`, `image-publish`. `make
   image-publish-dry-run VER=0.1.0-test` lokal verifizieren
   (kein echter Push). `feat(make):`-Commit.
6. **Trivy-Pattern** (`scripts/render-trivyignore.sh`,
   `.security/.trivyignore.in`, `Makefile` `image-scan`-Target):
   adaptieren. `make image-scan VER=0.1.0-test` lokal — kein
   `CRITICAL`/`HIGH` erwartet, sonst Iteration. `feat(security):`-
   Commit.
7. **`security-gates`-Erweiterung** im `Makefile`:
   `security-gates` ruft `image-scan` zusätzlich, wenn `VER`
   gesetzt ist (additiv, ohne `VER` läuft nur `govulncheck`).
   `feat(make):`-Commit.
8. **`release-guard`** (`scripts/release-guard.sh`,
   `scripts/test-release-guard.sh`, `Makefile`-Targets):
   adaptieren. `make release-guard-test` lokal grün.
   `feat(release):`-Commit.
9. **`CHANGELOG.md`** anlegen: `## [Unreleased]` plus erste
   `## [0.1.0] - 2026-05-XX`-Section (Datum wird mit Tag-Commit
   final). Pflicht-Inhalte aus §2.2 + slice-Liste M1–M7.
   `docs(changelog):`-Commit. K-1-Doku-Review verpflichtend.
10. **DCO-Bot-Aktivierung** (operativ, kein Commit) + Closure-
    Notiz in `cms`-Section: probot/dco aktiviert per Datum
    `2026-05-XX`, alle Commits seit ADR-0011-Acceptance
    (`abc1234`) attestiert.
11. **GitHub-Repository public-stellen** (operativ) — schließt
    `LH-AK-014` final (GHSA-Pfad aktiv). Closure-Notiz mit Datum.
12. **Roadmap M7 auf Done ziehen** (`docs(plan)`-Commit):
    `planning/in-progress/slice-M7-release-v0.1.0.md` wandert nach
    `done/`; `roadmap.md` §2-Tabelle und §7 auf
    `Done` + Closure-Sammel-Notiz; `planning/open/changelog.md`
    wandert nach `planning/done/` (Trigger gelöst); `planning/
    open/external-services-v03-activation.md` bleibt für v0.3+.
13. **Release-Gates-Vorlauf**: `make gates` + `make
    security-gates VER=0.1.0` + `make cluster-smoke` lokal
    grün. CI-Run auf `main`-Branch (oder Release-PR) grün.
14. **`make release-guard VER=0.1.0`** grün (mit
    `K_DESKFLIGHT_RELEASE_APPROVED=1`).
15. **Annotated Tag** `git tag -a v0.1.0 -m "<release-summary>"`
    + `git push origin v0.1.0`.
16. **`make image-publish VER=0.1.0`** (mit
    `K_DESKFLIGHT_IMAGE_PUBLISH_APPROVED=1`) — pushed
    `ghcr.io/pt9912/k-deskflight:v0.1.0`.
17. **GitHub-Release** `gh release create v0.1.0
    --notes-file <CHANGELOG-section>` — Body aus
    `CHANGELOG.md`-Section + Pull-Snippet.
18. **Closure-Notiz** in §10 dieses Slice-Plans (Verifikations-
    Ergebnis, Lessons learned, Out-of-Scope-Übergaben).

---

## 5. Lastenheft-Kennungen

**Primär adressiert:**

- `LH-MVP-002` — MVP-Vollständigkeit; M7 ist der Sammel-Closure.
- `LH-REL-001` — Version 0.1 (Tag, Image, Release-Notes).
- `LH-AK-013` — Dokumentation (Erst-Auslieferung über
  Beispielmanifeste und CHANGELOG; Anwender-Doku selbst war M6).
- `LH-AK-014` — Open-Source-Veröffentlichung möglich
  (GHSA-Pfad aktiv, README/CONTRIBUTING/CODE_OF_CONDUCT/SECURITY
  seit M1, CHANGELOG via M7).
- `LH-PROD-003a` — MVP-Beispiel (Lastenheft-Sync + zwei
  Anwender-Samples).
- `LH-QG-007` — Image-Scan (Trivy-Gate).
- `LH-QG-009` — Gate-Bündelung (`security-gates` erweitert um
  `image-scan` wenn `VER` gesetzt).

**Sekundär adressiert / verifiziert:**

- `LH-NF-006` — Minimalrechte operativ (Leader-Election-Lease-
  RBAC seit M2, M7 verifiziert via Pattern-Aktivierung).
- `LH-NF-013` — Dokumentation (CHANGELOG als Anwender-Sicht).
- `LH-NF-021` — Sprachregel (CHANGELOG Englisch).
- `LH-PRI-001` — Muss-Anforderungen für MVP (M7-Closure
  bestätigt Vollständigkeit).
- `LH-AK-001..-016` — Traceability-Matrix-Sammelschluss in §7.

---

## 6. Architekturartefakte

**Erfüllt durch M7:**

- `AR-020` (Makefile-Target-Anker) — `image-build VER=`,
  `image-publish`, `release-guard` jetzt scharf statt nur
  geplant.
- `AR-022` (GHCR-Image-Publish-Approval-Gate) — `image-publish-
  guard` aktiviert.
- `AR-026` (Leader-Election und Replica-Modell) —
  `--leader-elect`-Flag scharf, Single-Pod-Topologie-Guard
  produktiv. M7 deckt **MVP-Scope**; Multi-Replica-Aktivierung
  bleibt v0.2-Trigger.

**Verifiziert (kein Change):**

- `AR-015` (RBAC-Konsolidierung) — Lease-Marker seit M2 aktiv,
  M7 attestiert via `rbac_consistency_test.go` und Cluster-Smoke.
- `AR-021` (CI-Workflow) — `security-gates` erweitert; M1-Stand
  der `gates`-Definition bleibt.

**Out-of-Scope-Übergabe an v0.2:**

- `AR-008` (Mehrere konfigurierte Klassen pro Check) — M4 hat
  das Multi-Class-Pattern für `storageClass.names`/`ingressClass.
  names` umgesetzt. Keine M7-Pflicht.
- `AR-027` (Health-/Metrics-Probe-Topologie mit Leader-Filter) —
  Readiness-Leader-Filter ist M7-Pflicht zusammen mit
  Leader-Election; falls Implementierungs-Scope wachsen sollte,
  Übergabe an v0.2 mit `AR-027`-Folge-ADR (M7-Closure entscheidet).

---

## 7. Verifikation (Abnahmekriterien)

**Pflicht-Pfade:**

1. `make image-build VER=0.1.0` produziert
   `ghcr.io/pt9912/k-deskflight:v0.1.0` (lokal, ohne Push).
2. `kubectl apply -f deploy/samples/cluster-readiness-
   production.yaml` gegen den `cluster-smoke`-Cluster
   transitioniert die CR in `status.phase: Passed` mit fünf
   Conditions `True` (Re-Attest `LH-AK-001..-009/-011`).
3. `make image-publish-dry-run VER=0.1.0` ohne Approval-
   Variable schlägt mit Exit 2 fehl; mit
   `K_DESKFLIGHT_IMAGE_PUBLISH_APPROVED=1` läuft `would push`
   durch.
4. `make image-scan VER=0.1.0` zeigt **keinen** ungeklärten
   `CRITICAL`/`HIGH`-Fund. `MEDIUM`-Funde werden im Output
   gezeigt und in der CHANGELOG-`[0.1.0]`-Section unter
   `Security` referenziert.
5. `make security-gates VER=0.1.0` (Bündel: `govulncheck +
   image-scan`) grün.
6. `make release-guard VER=0.1.0` grün mit
   `K_DESKFLIGHT_RELEASE_APPROVED=1`; ohne Variable Exit 2.
7. `make release-guard-test` läuft alle Failure-Paths sauber
   durch (lokaler Pfad, nicht CI).
8. `make cluster-smoke` grün — inklusive neuer Step-9d Lease-
   Existenz-Check.
9. `make doc-refs` grün — alle neuen Doku-Verlinkungen
   (CHANGELOG, Beispielmanifeste, cr-examples.md-Updates,
   installation.md-Updates) auflösen.
10. `git ls-remote --tags origin refs/tags/v0.1.0` zeigt den
    annotated Tag nach Push.
11. `ghcr.io/pt9912/k-deskflight:v0.1.0` pullbar von einem
    nicht-authentifizierten Public-Klienten (Repo + GHCR-
    Sichtbarkeit `public`).
12. **Traceability-Matrix-Sammelschluss:** alle
    `LH-AK-001..-016` haben einen Closure-Eintrag in
    Roadmap §3 + jeweiligem Slice-Closure (`done/slice-MX-…md`).
    M7-Closure verlinkt die Matrix-Quelle.

**Slice-übergreifende Re-Attests** (bewusst, weil M7 die Sammel-
Closure ist):

- `LH-AK-001..-009` — via `deploy/samples/`-Apply gegen Cluster-
  Smoke (Schritt 2).
- `LH-AK-010` — robust gegen Per-Check-Fehler, durch M5-Tests
  bereits attestiert; M7 verifiziert via `make test` grün.
- `LH-AK-011` — Conditions vorhanden, M3-Re-Attest.
- `LH-AK-012` — Keine Secret-Leaks, M5-Re-Attest via `make
  test` (Aufruf-Pattern-Tests bestehen).
- `LH-AK-013` — Doku vorhanden + Beispielmanifeste.
- `LH-AK-014` — Open-Source-Veröffentlichung (GHSA + CHANGELOG +
  Public-Repo).
- `LH-AK-015` — Minimalrechte dokumentiert; `rbac_consistency_
  test.go` grün.
- `LH-AK-016` — RBAC-Selbstprüfung wirksam, M5-Re-Attest.

---

## 8. Out-of-Scope (geht in v0.2 oder später)

- **Helm-Chart** — `ADR 0005`-Schluss, v0.2 mit Folge-ADR.
- **ServiceMonitor / PodMonitor / PrometheusRule** —
  `ADR 0007 §4`-Schluss, v0.2 mit Helm-Chart-Aktivierung.
- **Eigene Domänen-Metriken** (`LH-NF-008` wörtlich) — v0.2,
  `ADR 0007 §2.3`.
- **Metrics-Endpoint-Authentication** — M6 §8-Übergabe, v0.2
  mit Folge-ADR zu `ADR 0007`.
- **Externe Service-Checks** (`LH-F-018..021`) — v0.2 / v0.3+,
  Trigger in `planning/open/external-services-v03-activation.md`.
- **Multi-Maintainer-Modell / `GOVERNANCE.md`** —
  `ADR 0011 §2.7`-Erweiterung, nicht MVP-blockierend.
- **SBOM-Publish / Sigstore-Cosign / Image-Provenance-Attestation**
  — v0.2-Trigger, M7 nutzt nur Trivy-Scan.
- **Multi-Replica-Aktivierung im Deployment-Default**
  (`replicas: >1`) — `AR-026`-Pattern ist scharf, aber
  Default-Replica bleibt `1`. v0.2 entscheidet pro `ADR 0014`-
  Folge.
- **`release-please` / `git-cliff`-Tooling-Pflege für CHANGELOG**
  — v0.2-Trigger, M7 bleibt manuell.
- **Mutation-Tests / Benchmarks / Fuzz-Tests** (`LH-QG-011a..c`)
  — opt-in, eigene ADR-Pflicht bei Aktivierung.

---

## 9. Risiken und Mitigation

- **Trivy meldet `CRITICAL`/`HIGH` auf `distroless/static-debian12:
  nonroot`** zum Release-Zeitpunkt: Mitigation primär
  Base-Image-Bump (Routine, kein ADR — `ADR 0012 §2.9`); falls
  kein Bump verfügbar, Vulnignore-Eintrag mit `expires` und
  Eintrag in `Security`-Section der CHANGELOG. Nur Funde mit
  dokumentiertem Workaround dürfen ignored werden.
- **GHCR-Push-Berechtigung fehlt** (Personal Access Token nicht
  vorhanden): `image-publish-guard` schlägt mit klarer Meldung
  fehl. Mitigation: PAT-Setup vor Tag, dokumentiert im
  Closure-Notiz-`§10`.
- **DCO-Sign-off-Lücken in Commits seit ADR-0011-Acceptance**
  (Reviewer-Finding möglich): Mitigation per `git rebase` /
  `git commit --amend -s` nur auf nicht-gepushten Commits;
  gepushte Commits bleiben gemäß `ADR 0011 §2.4`-Grandfathering-
  Klausel unverändert, dafür Closure-Notiz mit Liste der
  Lücken-Commits.
- **Leader-Election-Verdrahtung bricht den `cluster-smoke`-Pfad**
  (Lease-Race beim ersten Reconcile): Mitigation per
  `LeaderElectionReleaseOnCancel: true` (controller-runtime-
  Empfehlung) und `LeaderElectionRetryPeriod: 2s` (Default).
  Falls trotzdem flaky, Step 9d des Cluster-Smoke wartet bis
  zu 30 s auf Lease-Auftauchen statt sofortigem Read.
- **`LH-PROD-003a`-Sync verändert Lastenheft-Versionsstand**:
  Mitigation per Versionsmarker-Hebung im Lastenheft-Header
  (Routine-Edit, ADR-2-Mechanik); Commit-Body dokumentiert die
  Feldnamens-Reihe.
- **`v0.1.0`-Tag wird mit Tippfehler in der Tag-Message gepushed**:
  Mitigation per `release-guard`-Dry-Run vor echtem Push; falls
  Tag schon gepushed ist, kein force-overwrite — stattdessen
  `v0.1.1`-Patch-Tag mit korrekter Message (force-push auf Tags
  ist destruktiv und im Slice ausdrücklich verboten).

---

## 10. Closure

— wird nach Abschluss (Schritt 18 in §4) ergänzt: Verifikations-
Ergebnis (§7-Tabelle abgehakt), Lessons learned, Out-of-Scope-
Übergaben an v0.2, Sammel-Notiz-Move der Roadmap nach `done/`,
Closure-Datum.
