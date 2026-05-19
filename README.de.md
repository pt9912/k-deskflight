# k-deskflight

**OpenDesk Preflight Operator** — Kubernetes-Operator zur Vorabprüfung von
Cluster-Voraussetzungen für [OpenDesk](https://docs.opendesk.eu/)-Installationen.

> **Language version:** The English (default) variant of this README is at
> [`README.md`](README.md).

## Status

Die Implementierung ist **im Gang** (`LH-VM-004`). Sechs von sieben
MVP-Slices sind geschlossen; nur M7 (Release v0.1.0 mit
Beispielmanifesten, Trivy-Image-Scan und DCO) steht vor dem ersten
getaggten Release noch aus:

| Phase | Status | Quelle |
| ----- | ------ | ------ |
| Lastenheft (`LH-VM-001`) | Entwurf 0.1.0 | [`spec/lastenheft.md`](spec/lastenheft.md) |
| Architekturentscheidungen | 13 ADRs | [`docs/plan/adr/`](docs/plan/adr/) |
| Architektur-Spec (`AR-*`) | Done | [`spec/architecture.md`](spec/architecture.md) |
| Implementierung (`LH-VM-004`) | M1–M6 done — M7 pending | [`docs/plan/planning/`](docs/plan/planning/) |
| Pflichtenheft (`LH-VM-002`) | wächst mit den Slices | Slice-Pläne unter [`docs/plan/planning/done/`](docs/plan/planning/done/) |

Alle fünf MVP-Pflicht-Prüfungen (`LH-AK-005..-009`) sind produktiv:
Kubernetes-Version, StorageClass, IngressClass, cert-manager-
Existenz und Cluster-Ressourcen (CPU/Memory). Der Operator prüft
vor jedem Lauf seine eigenen RBAC-Rechte per
`SelfSubjectAccessReview` (`LH-AK-016`), ist gegen Per-Check-
Panics und -Timeouts gehärtet (`LH-AK-010`) und schreibt nie
unsanitierte Messages in Status oder Logs (`LH-AK-012`). CRD-
Installation, Operator-Rollout, Status-Reconcile und HTTP-Healthz/
Readyz/Metrics-Endpoints werden bei jedem Push gegen einen
realen kind-Cluster attestiert (siehe
[`ADR 0013`](docs/plan/adr/0013-cluster-smoke-platform.md)).

**Neu in M6** (`LH-AK-013`): Der `/metrics`-Endpoint wird jetzt über
ein dediziertes `Service`-Objekt exponiert und End-zu-End im
Cluster-Smoke attestiert — ein Probe-Pod scraped via Service-DNS-FQDN,
und die Response wird auf Prometheus-Format, controller-runtime-
Baseline-Metriken und Sanity-Zeilenzahl geprüft. Cluster-Smoke führt
zusätzlich vier failed-CR-Szenarien parallel zum passed-Sample aus
und re-attestiert `LH-AK-005`/`-006`/`-007`/`-009` sowohl auf
passed- als auch auf failed-Pfad. Der Operator unterstützt jetzt
ein konfigurierbares `spec.interval` (Default `5m`, Bounds
`[30s, 24h]`, AR-010-konforme Normalisierung). Anwender-Doku unter
[`docs/user/`](docs/user/) deckt Installation, evaluation/production-
CR-Beispiele, den Conditions-Katalog und ein Troubleshooting-Runbook
ab. Architektur-Punkt `AR-OP-005` (Namespace-Override-Mechanik)
ist in [`spec/architecture.md`](spec/architecture.md) geschlossen.

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
Lastenheft.

## Phasen-Roadmap (Stand der ADRs)

| Version | Inhalt | Quelle |
| ------- | ------ | ------ |
| v0.1 (MVP) | CRD, Controller, K8s-Version-/StorageClass-/IngressClass-/cert-manager-/Ressourcen-/RBAC-Prüfung, Container-Image, Beispielmanifeste (`deploy/manifests/`), Prometheus-`/metrics`-Endpoint mit Framework-Defaults | `LH-MVP-002`, `ADR 0005`, `ADR 0007` |
| v0.2 | DNS-, TLS-, Netzwerk-Reachability-Prüfung; Events; ConfigMap-Report; eigene Domänen-Metriken; Helm Chart | `LH-PRI-002`, `ADR 0005`, `ADR 0007`, `ADR 0008`, `ADR 0010` |
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
