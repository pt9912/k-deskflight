# k-deskflight

**OpenDesk Preflight Operator** — Kubernetes-Operator zur Vorabprüfung von
Cluster-Voraussetzungen für [OpenDesk](https://docs.opendesk.eu/)-Installationen.

> **Language version:** The English (default) variant of this README is at
> [`README.md`](README.md).

## Status

**MVP als [`v0.1.0`](https://github.com/pt9912/k-deskflight/releases/tag/v0.1.0)
am 2026-05-20 veröffentlicht** (`LH-VM-004`). Alle sieben
MVP-Slices sind geschlossen; das Container-Image
`ghcr.io/pt9912/k-deskflight:v0.1.0` ist das Release-Artefakt.
**v0.2 ist in Arbeit** — Slice M8 (Helm-Chart als zweiter
Distributionspfad) steht vor der Closure, M9–M15 sind in der
[v0.2-Roadmap](docs/plan/planning/in-progress/roadmap-0.2.md)
sequenziert, M16 schließt mit dem `v0.2.0`-Release-Tag.

| Phase | Status | Quelle |
| ----- | ------ | ------ |
| Lastenheft (`LH-VM-001`) | Entwurf 0.1.1 | [`spec/lastenheft.md`](spec/lastenheft.md) |
| Architekturentscheidungen | 15 ADRs | [`docs/plan/adr/`](docs/plan/adr/) |
| Architektur-Spec (`AR-*`) | Done | [`spec/architecture.md`](spec/architecture.md) |
| Implementierung (`LH-VM-004`) | M1–M7 done (`v0.1.0` ausgeliefert); M8 vor Closure; M9–M16 offen | [`docs/plan/planning/`](docs/plan/planning/) |
| Pflichtenheft (`LH-VM-002`) | wächst mit den Slices | Slice-Pläne unter [`docs/plan/planning/`](docs/plan/planning/) |

Release-Notes pro Version: [`CHANGELOG.md`](CHANGELOG.md).

### Was `v0.1.0` liefert

Alle fünf MVP-Pflicht-Prüfungen (`LH-AK-005..-009`) sind produktiv:
Kubernetes-Version, StorageClass, IngressClass, cert-manager-
Existenz und Cluster-Ressourcen (CPU/Memory). Der Operator prüft
vor jedem Lauf seine eigenen RBAC-Rechte per
`SelfSubjectAccessReview` (`LH-AK-016`), ist gegen Per-Check-
Panics und -Timeouts gehärtet (`LH-AK-010`) und schreibt nie
unsanitierte Messages in Status oder Logs (`LH-AK-012`). Der
controller-runtime-Manager läuft mit Leader-Election gegen ein
`coordination.k8s.io/lease` (`AR-026`). Der `/metrics`-Endpoint
wird über ein dediziertes `Service`-Objekt exponiert und
End-zu-End im kind-basierten Cluster-Smoke attestiert (siehe
[`ADR 0013`](docs/plan/adr/0013-cluster-smoke-platform.md)) —
sowohl gegen das passed-Sample als auch gegen vier failed-CR-
Szenarien. Der Operator unterstützt ein konfigurierbares
`spec.interval` (Default `5m`, Bounds `[30s, 24h]`,
AR-010-konforme Normalisierung). Anwender-Doku unter
[`docs/user/`](docs/user/) deckt Installation, evaluation/
production-CR-Beispiele, den Conditions-Katalog und ein
Troubleshooting-Runbook ab; zwei sofort einsetzbare CR-Templates
liegen unter [`deploy/samples/`](deploy/samples/), die rohen
Install-Manifeste unter
[`deploy/manifests/`](deploy/manifests/). Die Release-Pipeline
umfasst einen Trivy-Image-Scan (`CRITICAL`/`HIGH` blockierend)
und einen `make release-guard`-Schritt, der Approval-, Branch-,
Tag- und CHANGELOG-Section-Vorbedingungen vor dem Tag-Setzen
erzwingt.

### Installation

```bash
docker pull ghcr.io/pt9912/k-deskflight:v0.1.0
kubectl apply -f deploy/manifests/
```

Vollständiger Apply-Flow, Namespace-Override und Metrics-Scrape-
Binding: [`docs/user/installation.md`](docs/user/installation.md).

### v0.2-Stand — Helm-Chart verfügbar (Repository-Checkout bis M16)

Ein Helm-Chart liegt unter
[`deploy/charts/k-deskflight/`](deploy/charts/k-deskflight/);
die Templates sind 1:1 aus `deploy/manifests/` abgeleitet (per
`make helm-manifests-sync` verifiziert), und `helm install` ist
über dieselbe Cluster-Smoke-Matrix attestiert wie der Manifest-Pfad.
OCI-Distribution unter `oci://ghcr.io/pt9912/charts/k-deskflight`
ist mit
[`ADR 0015`](docs/plan/adr/0015-helm-chart-distributions-form.md)
festgelegt; der erste OCI-Push erfolgt mit dem `v0.2.0`-Tag in M16.
Bis dahin installieren aus dem Repository-Checkout:

```bash
helm install k-deskflight deploy/charts/k-deskflight/ \
    --namespace k-deskflight-system \
    --create-namespace \
    --set namespace.create=false
```

Vollständige Helm-Install-Operations-Doku:
[`docs/user/installation.md` §8](docs/user/installation.md).

## Was der Operator tun soll

Der Operator stellt die Custom Resource Definition `OpenDeskPreflightCheck`
(API-Gruppe `k-deskflight.geo-terrain.net/v1alpha1`, namespaced) bereit.
Anwender legen damit deklarativ fest, welche Cluster-Voraussetzungen geprüft
werden sollen — z. B. Kubernetes-Mindestversion, IngressClass,
StorageClasses, cert-manager-Verfügbarkeit, Ressourcen-Mindestgrenzen. Das
Ergebnis erscheint strukturiert im Status der CR (`LH-F-007`/`LH-F-032`) und
optional ab v0.2 zusätzlich als ConfigMap-Report mit YAML- und Markdown-Key
(`LH-F-028`, `ADR 0008`).

Der Operator beschränkt sich auf **lesende** Cluster-Inspektion (`LH-F-035`):
er installiert OpenDesk nicht, verändert keine OpenDesk-Komponenten und führt
keine destruktiven Aktionen aus (`LH-SYS-002..006`).

### Beispiel — MVP-Profil

```yaml
apiVersion: k-deskflight.geo-terrain.net/v1alpha1
kind: OpenDeskPreflightCheck
metadata:
  name: cluster-readiness
spec:
  profile: production
  checks:
    kubernetesVersion:
      min: "1.34"
    ingress:
      required: true
      className: nginx
    certManager:
      required: true
    storage:
      requiredClasses:
        - default
        - backup
    resources:
      minCpu: "16"
      minMemory: "64Gi"
```

Weitere Beispiele und Zielbilder unter `LH-PROD-003a` / `LH-PROD-003b` im
Lastenheft; sofort einsetzbare CR-Templates unter
[`deploy/samples/`](deploy/samples/).

## Phasen-Roadmap (Stand der ADRs)

| Version | Inhalt | Quelle |
| ------- | ------ | ------ |
| v0.1 (MVP) — **ausgeliefert 2026-05-20** | CRD, Controller, K8s-Version-/StorageClass-/IngressClass-/cert-manager-/Ressourcen-/RBAC-Prüfung, Container-Image, Beispielmanifeste (`deploy/manifests/`), Prometheus-`/metrics`-Endpoint mit Framework-Defaults | `LH-MVP-002`, `ADR 0005`, `ADR 0007` |
| v0.2 — **in Arbeit** | Helm-Chart (M8, vor Closure); DNS-/TLS-/Netzwerk-Reachability-Prüfung (M14/M15); Events (M9); ConfigMap-Report (M10); eigene Domänen-Metriken (M11) + OTel-Tracing-Spans (M12, `AR-OP-006`); Node- + ClusterIssuer-Prüfung (M13); Release-Tag `v0.2.0` (M16) | `LH-PRI-002`, `ADR 0005`, `ADR 0007`, `ADR 0008`, `ADR 0010`, `ADR 0014`, `ADR 0015` |
| v0.3+ | PostgreSQL-/S3-Erreichbarkeit (mit-Auth), HTML-Report, weitere Profile, kubectl-Plugin | `LH-PRI-003`, `ADR 0010` (mit-Auth-Block), Folge-ADR offen |

## Unterstützte Kubernetes-Versionen

Rolling-Window über die drei jeweils aktuellen Kubernetes-Minor-Versionen mit
aktivem Patch-Support (`ADR 0009`). Stand der ADR (2026-05-16): 1.34, 1.35,
1.36. Die jeweils aktuelle Matrix wird pro Operator-Release im Release-Note
dokumentiert.

## Projektartefakte und Sprachen

| Pfad | Inhalt | Sprache |
| ---- | ------ | ------- |
| `spec/lastenheft.md` | normatives Lastenheft mit `LH-*`-Kennungen | Deutsch |
| `docs/plan/adr/` | Architekturentscheidungen (ADRs) | Deutsch |
| `docs/plan/planning/` | Roadmap, offene Trigger, abgeschlossene Slices | Deutsch |
| `docs/archive/` | überholte oder verworfene Ideenskizzen | Deutsch |
| `README.md` | Default-Entry-Point | Englisch |
| `README.de.md` (diese Datei) | deutsche Übersetzung des `README.md` | Deutsch |
| [`CONTRIBUTING.md`](CONTRIBUTING.md), [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md), [`SECURITY.md`](SECURITY.md) | Open-Source-Konventionen | Englisch |
| Code, Commit-Messages, Issues, Pull Requests (ab `LH-VM-004`) | Implementierung und Community-Workflow | Englisch |

Sprachpolitik gemäß `LH-NF-021`. Die deutsche Spezifikation richtet sich
an behördennahe deutschsprachige Betreiber (`LH-PK-004`); der englische
`README.md`, der Code und der Community-Workflow öffnen das Projekt für
internationale Beitragende.

## Lizenz

[MIT](LICENSE).

## Beitragen

Beiträge sind willkommen. Konventionen, DCO-Sign-off (`git commit -s`),
Conventional-Commits-Format und Sprachregel stehen in
[`CONTRIBUTING.md`](CONTRIBUTING.md).

Sicherheitslücken und Verstöße gegen den
[Code of Conduct](CODE_OF_CONDUCT.md) bitte über [`SECURITY.md`](SECURITY.md)
melden.

## Verwandte Quellen

- OpenDesk-Projekt: https://docs.opendesk.eu/
- Kubernetes-Releases: https://kubernetes.io/releases/
- Contributor Covenant: https://www.contributor-covenant.org/version/2/1/
