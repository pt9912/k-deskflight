# k-deskflight

**OpenDesk Preflight Operator** — Kubernetes-Operator zur Vorabprüfung von
Cluster-Voraussetzungen für [OpenDesk](https://docs.opendesk.eu/)-Installationen.

## Status

Das Projekt befindet sich in der **Spezifikationsphase** des V-Modells
(`LH-VM-001`). Es gibt noch keinen Code — die Anforderungen sind
dokumentiert, alle Architekturentscheidungen für den MVP-Scope sind getroffen,
die Implementierung beginnt mit `LH-VM-004`.

| Phase | Status | Quelle |
| ----- | ------ | ------ |
| Lastenheft (`LH-VM-001`) | Entwurf 0.1.0 | [`spec/lastenheft.md`](spec/lastenheft.md) |
| Architekturentscheidungen | 11 ADRs | [`docs/plan/adr/`](docs/plan/adr/) |
| Pflichtenheft (`LH-VM-002`) | offen | folgt |
| Implementierung (`LH-VM-004`) | offen | folgt |

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
| `README.md` (diese Datei) | Projektüberblick | Deutsch |
| [`CONTRIBUTING.md`](CONTRIBUTING.md), [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md), [`SECURITY.md`](SECURITY.md) | Open-Source-Konventionen | Englisch |
| Code, Commit-Messages, Issues, Pull Requests (ab `LH-VM-004`) | Implementierung und Community-Workflow | Englisch |

Begründung der Sprachstaffelung: `LH-NF-021` — die fachliche Spezifikation
richtet sich an behördennahe deutschsprachige Betreiber (`LH-PK-004`), der
Code-Workflow an internationale Mitwirkende.

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
