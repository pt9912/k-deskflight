# ADR 0015 — Helm-Chart-Distributions-Form: OCI-Registry über GHCR

**Status:** Accepted
**Datum:** 2026-05-21
**Bezug:** [Lastenheft `LH-NF-016`, `LH-SST-010`](../../../spec/lastenheft.md),
[ADR 0001](0001-dokumentations-und-planungsstruktur.md),
[ADR 0002](0002-adr-lifecycle.md),
[ADR 0005 §4](0005-helm-chart-nicht-im-mvp.md),
[ADR 0011 §2.5](0011-governance-und-beitragskonventionen.md),
[ADR 0012 §2.8](0012-quality-gates.md),
[ADR 0014 §2.2](0014-v0.2-scope-schnitt.md),
[Slice M8 §2.8](../planning/in-progress/slice-M8-helm-chart.md)

---

## 1. Kontext

`ADR 0005 §4` hat die Distributions-Form des Helm-Charts als
explizit nicht-MVP-relevanten Folge-Themen-Slot offen gelassen:
„Distributionsform des Helm Charts ab v0.2 (traditionelles
Helm-Repository vs. OCI-Registry; eigene Domain vs. GHCR) — entsteht
mit der v0.2-Roadmap und ggf. einer eigenen ADR."

Slice M8 hat seit Step 1 den Helm-Chart unter
`deploy/charts/k-deskflight/` angelegt, drei Helm-Gates aktiviert
(`make helm-lint`, `make helm-template`, `make helm-manifests-sync`)
und den Cluster-Smoke um einen `INSTALL_MODE=helm`-Pfad erweitert
(Step 6). Slice-M8 Step 7 fixiert jetzt die Distributions-Form,
damit Step 8 (Anwender-Doku) die Install-Snippets vervollständigen
kann und M16 (Release-Slice) die `make chart-publish`-Mechanik
operativ einlöst.

Zwei breit etablierte Optionen für die Helm-Chart-Distribution:

- **Traditionelles Helm-Repository.** Ein HTTP-Server (eigene Domain,
  GitHub Pages, OCI-Static-Hosting) hostet `index.yaml` plus die
  Chart-Tarballs. Anwender-Pfad: `helm repo add k-deskflight <url>`,
  `helm install k-deskflight/k-deskflight`. Konvention bis Helm 3.7;
  funktioniert auch heute noch unverändert.
- **OCI-Registry.** Helm 3.8+ unterstützt native OCI-Distribution
  ([Helm 3.8 Release Notes](https://helm.sh/blog/storing-charts-in-oci/)).
  Anwender-Pfad: `helm install k-deskflight
  oci://<registry>/<path>/<chart> --version X.Y.Z`. Konvergiert mit
  Container-Image-Distribution — beide leben in derselben Registry.

Die Wahl bestimmt operative Konsequenzen für die `make chart-publish`-
Mechanik, die Anwender-Doku in `docs/user/installation.md §8.2`,
das Pre-Publish-Verifikations-Gate (M16) und die ArtifactHub-
Listing-Form. Sie betrifft **nicht** den Chart-Aufbau selbst, nur
den Distributions-Weg.

---

## 2. Entscheidung

### 2.1 Distributions-Form: OCI-Registry über GHCR

**Der Helm-Chart wird über die OCI-Registry-Variante auf
[GitHub Container Registry (GHCR)](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
veröffentlicht.**

Push-Ziel (für `helm push`):

```text
oci://ghcr.io/pt9912/charts
```

Anwender-Referenz (für `helm pull` / `helm install`):

```text
oci://ghcr.io/pt9912/charts/k-deskflight
```

Helm hängt den Chart-Namen aus `Chart.yaml.name` beim Push als
letztes Pfadsegment an; der resultierende GHCR-Package-Name ist
`ghcr.io/pt9912/charts/k-deskflight`. Der `charts/`-Pfad-Bestandteil
trennt den Chart-Package vom Operator-Image (`ghcr.io/pt9912/k-deskflight`),
sodass beide Artefakte ohne Namens-Kollision koexistieren —
konventionell für OCI-Setups, die sowohl Container-Images als auch
Helm-Charts an dieselbe Registry publishen (das cert-manager-Projekt
verwendet z. B. `quay.io/jetstack/charts/cert-manager` als
Chart-OCI-Pfad).

Begründungen:

- **Konsistenz mit der Image-Distribution.** Der Operator-Container
  läuft bereits über GHCR (`ghcr.io/pt9912/k-deskflight`, seit M7).
  Eine zweite Registry für Charts würde zwei Authentication-Wege
  (CI-Workflow + Anwender-Pull) und zwei Lifecycle-Mechaniken
  (Retention, Visibility, Token-Permissions) pflegen lassen. OCI
  über GHCR teilt die existierende Infrastruktur.
- **Kein eigenes Hosting.** Ein traditionelles Helm-Repository
  bräuchte entweder GitHub Pages mit handgepflegtem `index.yaml`
  und einem Publish-Skript-Cron, oder eine eigene Domain mit
  HTTP-Server. Beides ist zusätzliche Infrastruktur ohne Anwender-
  Mehrwert in unserem Setup.
- **Anonymous Pull bereits geklärt.** Slice M7 §10.5 #11b hat das
  GHCR-Package des Operator-Images explizit auf public + anonymous-
  pullable konfiguriert. Derselbe Settings-Pfad wird beim ersten
  Chart-Publish einmal manuell für das neue Chart-Package
  durchlaufen.
- **`helm pull` und `helm install` direkt.** Anwender brauchen kein
  `helm repo add` mehr — der OCI-Pfad funktioniert mit jedem
  Helm-3.8+-Client out of the box.
- **ArtifactHub unterstützt OCI-Repositories nach manueller Anlage.**
  Im Unterschied zu klassischen Helm-Repositories mit `index.yaml`
  entfällt die manuelle Index-Generierung; ArtifactHub indexiert
  dann den OCI-Pfad automatisch, sobald das Repository einmalig
  über die Web-UI eingetragen ist (siehe
  [ArtifactHub-Doku zu OCI-Repositories](https://artifacthub.io/docs/topics/repositories/helm-charts/#oci-based-helm-chart-repositories)).
  Die einmalige Anlage ist M16-Folgearbeit (§4).

### 2.2 Versions-Sync und SemVer-Politik

`Chart.yaml.version` (Chart-Schema-Version) und
`Chart.yaml.appVersion` (Operator-Image-Version) bleiben weiter
**entkoppelt** (Slice M8 §2.6):

- Eine reine Chart-Verbesserung ohne Operator-Änderung (z. B.
  `values.yaml`-Default-Korrektur, README-Update, Template-
  Refactor) bumpt nur `Chart.yaml.version`.
- Ein Operator-Release bumpt beide synchron — `Chart.yaml.version`
  mit dem nächsten Chart-Patch/Minor/Major-Schritt,
  `Chart.yaml.appVersion` auf den neuen Operator-Tag.

**Normative Versions-Sync-Constraint:** Bei jedem getaggten
Operator-Release **muss** gelten:
`Chart.yaml.appVersion == "v<RELEASE_TAG_OHNE_v>"`
**und** `Chart.yaml.version` ist auf den passenden Chart-SemVer-Wert
gehoben (`>=` letzte Chart-Version, in der Regel
`== RELEASE_TAG_OHNE_v`). Die Enforcement-Mechanik (Erweiterung von
`make release-guard` aus slice-M7 §2.5) entsteht mit der M16-
Implementation; diese ADR fixiert nur den Constraint.

M16 setzt damit auf `v0.2.0` für beide Werte und führt den ersten
Chart-Publish unter `Chart.yaml.version: 0.2.0` durch. Spätere
Chart-Patches (z. B. `0.2.1`) ohne Operator-Bump zeigen
`appVersion: "0.2.0"` und `version: 0.2.1` — der Constraint gilt
nur bei Operator-Release-Tags, nicht bei Chart-internen Patches
zwischen Operator-Releases.

### 2.3 Publish-Mechanik

Drei neue `make`-Targets analog zum Image-Publish-Pattern aus
Slice M7 §2.3 (`make image-publish-{dry-run,guard,}`):

| Target | Zweck |
| ------ | ----- |
| `make chart-publish-dry-run VER=X.Y.Z` | Chart-Tarball lokal bauen (`helm package`), Push-Ziel ausgeben, kein Push. |
| `make chart-publish-guard` | Approval-Variable `K_DESKFLIGHT_CHART_PUBLISH_APPROVED=1` prüfen; fail-fast wenn nicht gesetzt. |
| `make chart-publish VER=X.Y.Z` | `helm package` + `helm push` nach `oci://ghcr.io/pt9912/charts`. Setzt `chart-publish-guard` voraus, sowie `K_DESKFLIGHT_REGISTRY_TOKEN` für `helm registry login`. |

Pre-Publish-Verifikation (Pflicht, nicht im `chart-publish`-Target
erzwungen — der Anwender ruft die Gates separat; M16-Release-Guard
kette beide):

1. `make gates` — enthält bereits `helm-lint`, `helm-template`,
   `helm-manifests-sync`. Schließt strukturelle Chart-Korrektheit
   ab.
2. `make cluster-smoke INSTALL_MODE=helm` — verifiziert den
   functional Install-Pfad gegen einen kind-Cluster.

Authentifizierung im CI über `${{ secrets.GITHUB_TOKEN }}` mit
`packages: write`-Berechtigung; lokal über ein Personal Access
Token mit denselben Scopes (Standard-GHCR-Pfad).

Die konkrete Skript-Implementation (parallel zu
`scripts/image-scan.sh`) lebt unter `scripts/chart-publish.sh` und
wird in **M16** geschrieben — diese ADR fixiert nur die Form,
nicht die Implementation.

### 2.4 Anonymous-Pull-Aktivierung

Beim ersten erfolgreichen `helm push` legt GHCR ein **privates**
Package an — wie schon beim Operator-Image-Package in M7
([slice-M7 §10.5 #11b](../planning/done/slice-M7-release-v0.1.0.md));
die Privacy-Default-Politik ist nicht OCI-Chart-spezifisch, sondern
gilt für jedes neu angelegte GHCR-Package. Der Schritt zur
Public-Schaltung läuft einmalig manuell über die GitHub-Package-
Settings-UI:

1. `https://github.com/users/pt9912/packages/container/charts%2Fk-deskflight`
   öffnen. **Die URL ist erst nach dem ersten erfolgreichen
   `helm push` aufrufbar** — solange das Package nicht existiert,
   liefert GitHub einen 404. Reihenfolge im M16-Closure: erst der
   Publish-Run (`make chart-publish VER=0.2.0`), dann die
   Settings-URL.
2. „Package settings" → „Danger Zone" → „Change visibility" →
   `Public`.
3. Settings-Save.

Analog zum Operator-Image-Pfad aus Slice M7 §10.5 #11b. Ab v0.2.0
wird der initiale `helm push` erstmalig laufen; die Public-
Schaltung ist Bestandteil der M16-Closure.

### 2.5 Anwender-Install-Befehl

```bash
# Direkter Install:
helm install k-deskflight oci://ghcr.io/pt9912/charts/k-deskflight \
    --version 0.2.0 \
    --namespace k-deskflight-system \
    --create-namespace \
    --set namespace.create=false

# Oder Pull + lokaler Install:
helm pull oci://ghcr.io/pt9912/charts/k-deskflight --version 0.2.0
helm install k-deskflight ./k-deskflight-0.2.0.tgz \
    --namespace k-deskflight-system \
    --create-namespace \
    --set namespace.create=false
```

**`--version` wird explizit empfohlen.** Ohne Flag versucht Helm
das im `Chart.yaml` veröffentlichte SemVer als Default-Tag
aufzulösen — ein OCI-Pull ohne `--version` ist also nicht-zufällig,
aber bei mehreren publizierten Versionen führt das zu
unbeabsichtigtem Auto-Upgrade beim nächsten Release. Pinning via
`--version` ist die robustere Operations-Variante; semver-Range-
Auswahl (`--version "^0.2"`) ist Helm-3-Standard.

Ein implizites `latest`-Tag wie bei Container-Images gibt es bei
Helm-OCI **nicht** — ohne `--version` wählt Helm das Chart-SemVer,
nicht eine alphabetisch-letzte Tag-Liste.

---

## 3. Konsequenzen

- **Slice M8 §8.2 in `docs/user/installation.md`** wird mit dem
  OCI-Install-Snippet ergänzt, sobald M16 den ersten erfolgreichen
  Publish + Public-Schaltung durchgeführt hat. Der M8-Step-8-
  Review-Hinweis („kommt mit ADR 0015 / M16-Publish") wird damit
  eingelöst — der Doku-Block wandert von „kommt noch" zu
  funktional.
- **`deploy/charts/k-deskflight/README.md`** (Chart-internes
  README) bekommt mit M16 einen Hinweis auf die OCI-Quelle in der
  Source-Sektion.
- **M16-Slice-Plan** schreibt die `scripts/chart-publish.sh`-
  Implementation und die drei `make`-Targets nach dem Image-
  Publish-Muster aus M7 §2.3. Approval-ENVs nutzen den
  `K_DESKFLIGHT_*`-Präfix.
- **`make release-guard`** (slice-M7 §2.5) wird in M16 um eine
  Chart-Version-Prüfung erweitert: vor `git tag -a v0.2.0` muss
  `Chart.yaml.version` und `Chart.yaml.appVersion` mit dem
  Release-Tag übereinstimmen.
- **Trivy-Image-Scan-Pendant für den Chart-Tarball** ist nicht
  vorgesehen — Charts enthalten keine ausführbare Software, die
  Supply-Chain-Risiko-Oberfläche reduziert sich auf die Chart-
  YAML-Templates selbst (die statisch via `helm-manifests-sync`
  attestiert sind) und das referenzierte Operator-Image
  (separat über `make image-scan` gescannt).
- **CHANGELOG `[Unreleased]`-Section** bekommt mit Slice M8 §4 Step 9
  einen Eintrag „added Helm-Chart distributed via
  `oci://ghcr.io/pt9912/charts/k-deskflight`".
- **`AR-022` (Image-Tagging und -Distribution)** in
  `spec/architecture.md` wird mit Slice-Closure um einen
  Folgesatz zur Chart-Distribution ergänzt (Chart-Version-Sync
  mit Operator-Tag, OCI-Pfad-Konvention).
- **Tag-Präfix-Konvention bewusst asymmetrisch:** Image-Tag trägt
  `v`-Präfix (`ghcr.io/pt9912/k-deskflight:v0.2.0`, GHCR-Image-
  Konvention via `_helpers.tpl` `k-deskflight.imageRef`), Chart-
  Version trägt keinen `v`-Präfix (`Chart.yaml.version: 0.2.0`,
  Helm-SemVer-Konvention). Anwender, die `image.tag` explizit
  setzen, kontrollieren den `v` selbst (Helper überspringt das
  Präfix-Prepending bei explizitem Override).

---

## 4. Nicht Gegenstand dieser ADR

- **Chart-Signing via cosign** — eigene Folge-ADR (v0.3+), sobald
  der erste Anwender-Anwendungsfall für signierte Charts entsteht.
  Helm 3.8+ unterstützt OCI-native Cosign-Signaturen; die Aktivierung
  braucht einen Schlüssel-Lifecycle-Plan, der nicht in den M8-Scope
  gehört.
- **chart-testing (`ct lint`/`ct install`) als Quality-Gate** —
  Trigger-Datei [`../planning/open/chart-testing-activation.md`](../planning/open/chart-testing-activation.md);
  Aktivierungs-Anlass siehe dort.
- **helm-docs-Automatisierung der Chart-`README.md`** —
  Trigger-Datei [`../planning/open/helm-docs-automation.md`](../planning/open/helm-docs-automation.md).
- **ArtifactHub-Repository-Anlage** — operative Folgearbeit für M16
  (Repository-Submission über die ArtifactHub-Web-UI, einmaliger
  Schritt nach dem ersten Public-Publish; eigene Slice-Plan-Notiz
  unter `next/` oder M16-internal task).
- **Chart-Repository-Mirror auf eigener Domain** (z. B.
  `helm.k-deskflight.io`) — nicht vorgesehen; ein zukünftiger
  Mehrfach-Distributions-Bedarf würde eine eigene ADR auslösen.
- **Subcharts und Helm-Hooks** — v0.2-out-of-scope per Slice M8 §8;
  Aktivierung mit konkretem Use-Case in v0.3+.
- **Implementation der `scripts/chart-publish.sh` und der drei
  `make chart-publish*`-Targets** — gehört zu M16. Diese ADR
  fixiert die Form (OCI, GHCR, Push-Ziel, Approval-Muster); M16
  schreibt das Skript und wired es ans Makefile.
- **Chart-Version-Bump-Disziplin-Enforcement** — manuelle Pflege
  bleibt in v0.2; eine automatische Version-Bump-Erkennung wäre
  Teil der `chart-testing`-Aktivierung
  ([`../planning/open/chart-testing-activation.md`](../planning/open/chart-testing-activation.md) §3).
- **Tag-Immutability-Protection.** OCI-Registries (GHCR
  eingeschlossen) erlauben standardmäßig Re-Push desselben Tags;
  ein versehentlicher `helm push` einer geänderten `0.2.0` würde
  den Inhalt überschreiben, ohne dass Anwender das anhand des Tags
  erkennen. v0.2 hat dafür keine technische Absicherung; Operations-
  Disziplin („Tag-Re-Use ist verboten") gilt informell. Eine harte
  Immutability-Erzwingung (z. B. via Cosign-Attestation oder
  Registry-Policy) wäre Folge-ADR, ggf. zusammen mit dem
  Chart-Signing-Plan in v0.3+.
