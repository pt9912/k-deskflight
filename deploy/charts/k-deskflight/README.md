# k-deskflight Helm-Chart

[`k-deskflight`](https://github.com/pt9912/k-deskflight) ist der
**OpenDesk Preflight Operator** — ein Kubernetes-Operator, der vor
einer OpenDesk-Installation Cluster-Voraussetzungen prüft.

Dieser Chart liefert den Operator als Helm-Release.

## Voraussetzungen

- Kubernetes ≥ 1.34 (siehe
  [`ADR 0009`](https://github.com/pt9912/k-deskflight/blob/main/docs/plan/adr/0009-k8s-versions-support-und-profile-mindestversionen.md)).
- Helm 3.x auf dem Client.
- Cluster-Admin-Rechte für CRD-/ClusterRole-/ClusterRoleBinding-Apply.

## Installation

Aus dem Repository-Checkout:

```bash
helm install k-deskflight deploy/charts/k-deskflight/ \
    --namespace k-deskflight-system \
    --create-namespace \
    --set namespace.create=false
```

`--create-namespace` lässt Helm den Operator-Namespace anlegen,
`--set namespace.create=false` schaltet das chart-eigene Namespace-
Template ab, sonst kollidiert es mit der Helm-Anlage. Eine
ausführliche Begründung samt Pattern-B-Alternative (Anwender legt
den Namespace selbst an) steht in der Operations-Doku unter
[`docs/user/installation.md §8`](https://github.com/pt9912/k-deskflight/blob/main/docs/user/installation.md#8-alternative-installation-via-helm-chart).

## Konfiguration

Vollständige Werte-Definition mit Defaults und Kommentaren:
[`values.yaml`](values.yaml). Die JSON-Schema-Validierung
([`values.schema.json`](values.schema.json)) weist invalide Werte
client-seitig ab — vor jedem Server-Round-Trip.

Übersicht der wichtigsten Slots:

| Pfad | Default | Zweck |
| ---- | ------- | ----- |
| `namespace.name` | `k-deskflight-system` | Operator-Namespace; muss zum `--namespace`-Flag passen. |
| `namespace.create` | `true` | Chart rendert Namespace; auf `false` setzen wenn `--create-namespace` (Helm) oder externe Namespace-Anlage verwendet wird. |
| `image.repository` | `ghcr.io/pt9912/k-deskflight` | Operator-Image. |
| `image.tag` | `""` | Leer → fällt auf `Chart.appVersion` zurück (mit v-Präfix für GHCR-Konvention). |
| `operator.mode` | `cluster-wide` | `cluster-wide` (Default) oder `namespace-scope` ([`AR-016`/`AR-017`](https://github.com/pt9912/k-deskflight/blob/main/spec/architecture.md)). |
| `operator.replicas` | `1` | Multi-Replica-HA-Tuning ist v0.2-out-of-scope. |
| `operator.leaderElect` | `true` | Leader-Election ([`AR-026`](https://github.com/pt9912/k-deskflight/blob/main/spec/architecture.md)). |
| `metrics.enabled` | `true` | Prometheus-`/metrics`-Endpoint via Service. |
| `metrics.port` | `8080` | Service-Port; Container-Port ist statisch 8080. |
| `metrics.clusterRolePattern.create` | `true` | Pattern-Asset-ClusterRole für Prometheus-Operator-Scrape (v0.1 explizit unauthenticated, [`ADR 0007 §3`](https://github.com/pt9912/k-deskflight/blob/main/docs/plan/adr/0007-prometheus-metrik-scope-im-mvp.md)). |
| `rbac.create` | `true` | ClusterRole + Bindings via Chart erzeugen. Bei `false` muss RBAC extern verwaltet werden — siehe Operations-Doku. |
| `serviceAccount.create` | `true` | Operator-ServiceAccount via Chart erzeugen. |
| `crd.install` | `true` | CRD via Chart rendern. Bei `false` (z. B. zentrale GitOps-CRD-Verwaltung) muss die CRD separat appliziert werden. |

## Betriebsmodi

Cluster-Wide Mode (Default) reconciliert OpenDeskPreflightCheck-CRs
in allen Namespaces. Namespace-Reconcile-Scope Mode beschränkt den
Reconcile auf den Operator-Namespace:

```bash
helm install k-deskflight deploy/charts/k-deskflight/ \
    --namespace k-deskflight-system --create-namespace \
    --set namespace.create=false \
    --set operator.mode=namespace-scope
```

Beide Modi sind über die Cluster-Smoke-Pipeline attestiert
(`make cluster-smoke INSTALL_MODE=helm`).

## Anwendung

Nach dem Install liefert der Chart NOTES-Output mit den
Verifikations-Kommandos. Ein erstes Preflight-Check-CR:

```bash
kubectl apply -f https://raw.githubusercontent.com/pt9912/k-deskflight/main/deploy/samples/cluster-readiness-evaluation.yaml
kubectl get opendeskpreflightcheck cluster-readiness
```

Vollständige Anwender-Doku, Conditions-Katalog und Troubleshooting:
[`docs/user/`](https://github.com/pt9912/k-deskflight/tree/main/docs/user).

## Uninstall

```bash
helm uninstall k-deskflight --namespace k-deskflight-system
```

**Achtung:** `helm uninstall` löscht auch die CRD und damit alle
`OpenDeskPreflightCheck`-CRs cluster-weit. CRs vorher sichern.
Wer CRs überleben lassen will, installiert mit
`--set crd.install=false` und verwaltet die CRD separat.

## Source

- Repository: <https://github.com/pt9912/k-deskflight>
- Lizenz: MIT
- Maintainer: pt9912
