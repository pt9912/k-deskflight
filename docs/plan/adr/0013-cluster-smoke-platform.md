# ADR 0013 — Cluster-Smoke-Plattform für lokale und CI-Attestation

**Status:** Accepted
**Datum:** 2026-05-17
**Bezug:** [Lastenheft](../../../spec/lastenheft.md),
[ADR 0001](0001-dokumentations-und-planungsstruktur.md),
[ADR 0002](0002-adr-lifecycle.md),
[ADR 0009](0009-k8s-versions-support-und-profile-mindestversionen.md),
[ADR 0012](0012-quality-gates.md),
[Roadmap §3 M2/M3](../planning/done/roadmap.md)

---

## 1. Kontext

Mehrere Lastenheft-Abnahmekriterien aus §17 sind strikt
cluster-pflichtig — sie verlangen wörtlich „lässt sich in einem
Kubernetes-Cluster installieren / starten":

- `LH-AK-001` — CRD installierbar.
- `LH-AK-002` — Operator startbar.

Weitere Items aus §17 lassen sich zwar durch fake-client-Unit-Tests
verifizieren (`LH-AK-003`/`-004`/`-005`/`-011`), ein realer
Cluster-Lauf bleibt aber als zusätzliches observational Attest
sinnvoll und für die Roadmap-Slice-Closure-Notizen (M2 §10.5,
M3 §10.5) angefordert.

Bisher gab es im Repository **keinen** verbindlichen Mechanismus, einen
echten Kubernetes-Cluster für solche End-to-End-Smoke-Tests zu
provisionieren. Die §10.5-Attest-Einträge der M2/M3-Closures stehen
mit dem Status „pending — wird mit dem ersten kind/minikube-Lauf
attestiert" — ohne Festlegung welcher Tool-Stack das eigentlich ist.

Drei breit etablierte Optionen für lokale Kubernetes-Cluster-Provisionierung:

- **kind** (Kubernetes IN Docker, SIG-Projekt) — Upstream-Kubernetes
  in Docker-Containern, ~30 s Startzeit, kubebuilder-/OperatorSDK-
  Default.
- **k3d** (k3s in Docker) — Wrapper um die Rancher-k3s-Distribution,
  ~10 s Startzeit, k3s ist bewusst gestrippt (Traefik-Default,
  SQLite-Backend, kein cloud-controller-manager).
- **minikube** (Container- oder VM-basiert) — ältestes Projekt,
  schwerergewichtig, ~60–120 s Startzeit.

Die Wahl hat operative Folgen für die §10.5-Attest-Pfade, für die
GitHub-Actions-CI-Pipeline und für künftige Profil-Validierung
(`LH-PROF-001` zählt `k3s` als eigenständiges Profil-Ziel ab v0.2 auf,
[`ADR 0009 §3`](0009-k8s-versions-support-und-profile-mindestversionen.md)).

---

## 2. Entscheidung

### 2.1 Default-Plattform: kind

Cluster-Smoke-Tests für `LH-AK-001`/`LH-AK-002` und für die
zusätzlichen observational Cluster-Attestate aus M2/M3 §10.5
laufen auf **`kind` (Kubernetes IN Docker)**.

Begründung:

- **Upstream-Kubernetes-Treue.** kind packt unmodifiziertes Upstream-
  Kubernetes (`kindest/node:vX.Y.Z`-Images) in Docker-Container.
  Das matched den Target-Stack aus
  [`ADR 0009 §2.1`](0009-k8s-versions-support-und-profile-mindestversionen.md)
  (K8s 1.32 / 1.33 / 1.34, Default 1.34) ohne Distribution-Drift.
  k3s und minikube hätten distribution-spezifische Eigenheiten
  (k3s: Traefik-Default-Ingress, SQLite-Backend statt etcd; minikube:
  optionale addon-Layer), die bei einem Operator-Smoketest entweder
  bewusst aktiviert oder ignoriert werden müssten.
- **Operator-Ökosystem-Konvergenz.** kubebuilder, Operator-SDK,
  controller-runtime und die meisten Kubernetes-SIG-Operatoren nutzen
  kind als Default-Test-Cluster. Konsistenz mit dem Ökosystem
  reduziert Lese-Last für externe Mitwirkende.
- **CI-Integration.** `helm/kind-action` ist die GitHub-Actions-
  Default-Action für kind-Provisionierung; weit gepflegt, SHA-pinnbar
  analog [`ADR 0011 §2.5`](0011-governance-und-beitragskonventionen.md)/
  [`ADR 0012 §2.8`](0012-quality-gates.md).
- **Image-Loading-Pfad.** `kind load docker-image …` ist ein
  etablierter Mechanismus, lokal gebaute Operator-Images ohne
  Registry-Push in den Cluster zu schieben. Konsistent mit der
  Docker-only-Build-Linie (slice-M1 §2.1).
- **Schwesterprojekt-Konsistenz.** `/Development/m-trace` nutzt
  ebenfalls kind im Test-Layer (vgl. `scripts/test-browser-e2e.sh`,
  `scripts/validate-k8s-examples.sh`); k-deskflight folgt damit der
  bekannten Linie ohne neue Wartungs-Last.

### 2.2 k3d bleibt als Folge-Plattform vorgesehen

`k3d` wird mit der **Aktivierung des `k3s`-Profils** aus
[`LH-PROF-001`](../../../spec/lastenheft.md) (v0.2 oder später)
als **zusätzliche** Cluster-Smoke-Plattform eingeführt — nicht als
Ersatz für kind, sondern als Profil-spezifischer Validierungs-Pfad.
Dann läuft ein paralleler `make cluster-smoke-k3s`-Target gegen k3d
und prüft, ob die Operator-Logik mit den k3s-spezifischen
Distribution-Eigenheiten (Traefik-Ingress, SQLite-Backend) noch
korrekt arbeitet.

Diese ADR bindet die Folge-Aktivierung **nicht** im Detail — der
Trigger entsteht mit dem `k3s`-Profil-Slice und kann eigene
Sub-Entscheidungen verlangen (z. B. Image-Loading-Pattern, k3d-
Versionspin).

### 2.3 minikube wird nicht eingeführt

minikube ist langsamer (~60–120 s Startzeit), schwerergewichtig
(VM- oder DinD-Layer), und decken-aktiviert keine Eigenschaften, die
nicht auch kind liefert. Für lokale GUI-orientierte Entwickler-
Workflows mit Dashboard etc. ist minikube weiterhin etabliert, aber
das ist nicht unser Smoke-Test-Bedarf. Wir nehmen es **nicht** auf
die unterstützte Liste.

### 2.4 Versionspin

Tools werden in der Dockerfile-`smoke`-Stage installiert (siehe §2.6),
deshalb sind die Pins zentral im Dockerfile via `ARG` gesetzt — analog
`ARG GO_VERSION` und `ARG CONTROLLER_GEN_VERSION`.

| Komponente | Pin-Quelle | Hebung |
| ---------- | ---------- | ------ |
| `kind` CLI | `Dockerfile` `ARG KIND_VERSION` (Default `v0.31.0` — matched zu kindest/node:v1.34.0; ältere kind-Versionen können containerd-2.x-TOML im node-Container nicht parsen) | Routine ohne ADR |
| `kubectl` CLI | `Dockerfile` `ARG KUBECTL_VERSION` (Default `v1.34.0`) | Routine ohne ADR |
| `kindest/node`-Image | Default `kindest/node:v1.34.0` im Smoke-Skript (synchron mit [`ADR 0009 §2.2`](0009-k8s-versions-support-und-profile-mindestversionen.md) Min-Wert); per Env `K8S_VERSION=…` override-bar | Routine ohne ADR |

Eine Version-Matrix-Erweiterung (z. B. parallele Läufe gegen
`kindest/node:v1.32.0` + `v1.33.0` + `v1.34.0` zur Validierung der
gesamten Operator-Support-Matrix aus
[`ADR 0009 §2.1`](0009-k8s-versions-support-und-profile-mindestversionen.md))
bleibt eine **Folge-Entscheidung** für eine spätere Slice
(typischerweise M6 envtest-Suite oder M7 Release-Härtung).

### 2.5 CI-Verankerung

- Eigener GitHub-Actions-Workflow `.github/workflows/cluster-smoke.yml`.
- Trigger: `push` auf `main` (Post-Merge-Verifikation) und
  `workflow_dispatch` (manueller Trigger für Debugging).
- **Nicht** in `make gates` oder dem PR-Trigger — Cluster-Smoke kostet
  ~60–90 s pro Lauf und wäre PR-Pfad-Belastung ohne neuen
  Lastenheft-Beweis. Die Pflicht-Gates aus
  [`ADR 0012 §2.11`](0012-quality-gates.md) bleiben unverändert.
- Der Workflow installiert **keine externen Actions** für kind/kubectl
  (der `smoke`-Dockerfile-Stage bringt beides mit). Steps:
  `actions/checkout` (SHA-gepinnt analog `ci.yml`), `make build`,
  `make cluster-smoke`. Optional schreibt ein abschließender Step ein
  Attest-Artefakt (Status-YAML der Sample-CR).

### 2.6 Docker-only-Konsistenz: `smoke`-Stage im Dockerfile

`make cluster-smoke` hält die Docker-only-Konvention aus
[slice-M1 §2.1](../planning/done/slice-M1-repo-skeleton.md) ein:

- Eine neue Dockerfile-Stage `FROM … AS smoke` liefert kind CLI und
  kubectl pinned (analog tools-Stage / controller-gen-Pattern).
- `make cluster-smoke` ruft `docker build --target smoke` und
  anschließend `docker run` mit:
  - `-v /var/run/docker.sock:/var/run/docker.sock` (Docker-out-of-
    Docker: der `kind`-Prozess im Container spawnt
    `kindest/node`-Container über die **Host-Docker-Engine**;
    geladene lokale Images wie `k-deskflight:go` bleiben sichtbar).
  - `--network host` (kind exposed den apiserver auf `127.0.0.1:<port>`;
    aus dem smoke-Container muss derselbe Loopback erreichbar bleiben).
  - `-v "$(CURDIR):/src"` (Workspace-Mount; Skripte + Manifeste
    werden gelesen, keine Schreibe).
- Host-Tool-Voraussetzung bleibt damit auf `docker` reduziert —
  exakt wie die Pflicht-Targets in `make gates`.

`--network host` funktioniert stabil unter Linux (CI-Runner sind
ubuntu-latest, lokale Dev-Maschinen meist Linux). macOS-Docker-Desktop
hat hier Einschränkungen; falls künftig macOS-Dev-Support nötig wird,
kommt ein Folge-Pfad als `make cluster-smoke-macos` mit
Port-Forwarding statt host-Networking. Diese ADR bindet das nicht.

---

## 3. Konsequenzen

- **Roadmap:** §10.5-Attest-Einträge der Slice-M2/M3-Closure-Notizen
  zeigen jetzt explizit auf `make cluster-smoke` / kind als
  Verifikations-Mechanik. Der Folge-Attest-Pfad ist damit nicht mehr
  unspezifiziert.
- **Makefile:** zwei neue Targets — `cluster-smoke` (E2E-Lauf) und
  `cluster-smoke-cleanup` (kind-Cluster löschen).
- **CI:** neuer Workflow-File `.github/workflows/cluster-smoke.yml`,
  separat vom Pflicht-`ci.yml`-Bundle.
- **Host-Tool-Anforderung bleibt minimal:** nur Docker (siehe §2.6);
  kein zusätzlicher Eintrag in CONTRIBUTING.md nötig. Die Docker-only-
  Konvention aus [slice-M1 §2.1](../planning/done/slice-M1-repo-skeleton.md)
  bleibt unverändert wirksam.
- **Pflichtenheft (`LH-VM-002`):** Test-Infrastruktur-Beschreibung
  enthält künftig die kind-Variante als Default-Plattform.
- **Lastenheft `LH-AK-001`/`LH-AK-002`:** die operative Verifikation
  ist mit dieser ADR + `make cluster-smoke` definiert. Roadmap-§10.5-
  Notizen werden beim ersten erfolgreichen Lauf nachgepflegt
  (Pattern analog M1 §10.5 CI-Attest).
- **`LH-PROF-001` k3s-Profil:** wird nicht durch diese ADR
  vorweggenommen; der Trigger bleibt offen (Folge-Slice / Folge-ADR).
- **`LH-PROD-001` Naming:** der kind-Cluster heißt
  `k-deskflight-smoke` (Konvention `<projekt>-<rolle>`).

---

## 4. Nicht Gegenstand dieser ADR

- **Konkrete Smoke-Skript-Implementierung** (`scripts/cluster-smoke.sh`,
  Step-Reihenfolge, Assertion-Pattern) — entsteht mit der Folge-Slice
  bzw. der M2/M3-§10.5-Attest-Aktivierung. Diese ADR bindet nur
  Plattform und CI-Verortung.
- **Multi-Version-Matrix** (parallele Läufe gegen 1.32/1.33/1.34) —
  bewusst aufgeschoben (M6 oder M7).
- **k3d-Integration** — wird mit dem k3s-Profil-Slice erschlossen.
- **envtest-Integration** (in-Process-API-Server für Unit-/
  Integrationstests) — separate Mechanik, kommt mit M6 Tests-Slice.
- **Trivy Image-Scan** auf dem in kind geladenen Image — bleibt im
  CI über [`ADR 0012 §2.9`](0012-quality-gates.md) im `image-scan`-
  Target, das in M7 aktiviert wird.
- **Test-Daten / Fixture-Variation** (mehrere CR-Beispiele mit
  unterschiedlichen Specs) — kann im Smoke-Skript wachsen, ist aber
  nicht ADR-pflichtig.
