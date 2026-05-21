# k-deskflight

**OpenDesk Preflight Operator** — Kubernetes operator that performs preflight
checks on a cluster before an [OpenDesk](https://docs.opendesk.eu/) installation.

> **Sprachversion:** Die deutsche Variante dieses README liegt unter
> [`README.de.md`](README.de.md).

## Status

**MVP released as [`v0.1.0`](https://github.com/pt9912/k-deskflight/releases/tag/v0.1.0)
on 2026-05-20** (`LH-VM-004`). All seven MVP slices are closed; the
container image `ghcr.io/pt9912/k-deskflight:v0.1.0` is the release
artifact. **v0.2 is in progress** — slice M8 (Helm-Chart as a second
distribution path) is ready for closure, M9–M15 are sequenced in the
[v0.2-roadmap](docs/plan/planning/in-progress/roadmap-0.2.md), M16
closes with the `v0.2.0` release tag.

| Phase | Status | Source |
| ----- | ------ | ------ |
| Lastenheft (`LH-VM-001`) | Draft 0.1.1 | [`spec/lastenheft.md`](spec/lastenheft.md) |
| Architecture decisions | 15 ADRs | [`docs/plan/adr/`](docs/plan/adr/) |
| Architecture spec (`AR-*`) | Done | [`spec/architecture.md`](spec/architecture.md) |
| Implementation (`LH-VM-004`) | M1–M7 done (`v0.1.0` shipped); M8 ready for closure; M9–M16 open | [`docs/plan/planning/`](docs/plan/planning/) |
| Pflichtenheft (`LH-VM-002`) | grows alongside slices | per-slice plans in [`docs/plan/planning/`](docs/plan/planning/) |

Release notes per version: [`CHANGELOG.md`](CHANGELOG.md).

### What `v0.1.0` ships

All five MVP-mandatory checks are live (`LH-AK-005..-009`):
Kubernetes version, StorageClass, IngressClass, cert-manager
presence, and cluster resources (CPU/memory). The operator
self-checks its own RBAC permissions before each run via
`SelfSubjectAccessReview` (`LH-AK-016`), is hardened against
per-check panics and timeouts (`LH-AK-010`), and never writes
unsanitised messages into status or logs (`LH-AK-012`). The
controller-runtime manager runs with leader election against a
`coordination.k8s.io/lease` (`AR-026`). The `/metrics` endpoint
is exposed via a dedicated `Service` and end-to-end attested
through the kind-based cluster-smoke pipeline (see
[`ADR 0013`](docs/plan/adr/0013-cluster-smoke-platform.md)),
covering the passed sample and four failed-CR scenarios. The
operator supports a configurable `spec.interval` (default `5m`,
bounds `[30s, 24h]`, AR-010-conformant normalisation). End-user
documentation under [`docs/user/`](docs/user/) covers
installation, evaluation/production CR examples, the conditions
catalogue and a troubleshooting runbook; two ready-to-apply CR
templates ship under [`deploy/samples/`](deploy/samples/), raw
install manifests under [`deploy/manifests/`](deploy/manifests/).
The release pipeline includes a Trivy image scan
(`CRITICAL`/`HIGH` blocking) and a `make release-guard` step
that enforces approval, branch, tag and CHANGELOG-section
preconditions before tagging.

### Install

```bash
docker pull ghcr.io/pt9912/k-deskflight:v0.1.0
kubectl apply -f deploy/manifests/
```

Full apply flow, namespace override and metrics-scrape binding:
[`docs/user/installation.md`](docs/user/installation.md).

### v0.2 progress — Helm-Chart available (repository-checkout only
until M16)

A Helm-Chart now lives under
[`deploy/charts/k-deskflight/`](deploy/charts/k-deskflight/);
templates are 1:1 derived from `deploy/manifests/` (verified by
`make helm-manifests-sync`) and `helm install` is attested through
the same cluster-smoke matrix as the raw-manifest path. OCI
distribution at `oci://ghcr.io/pt9912/charts/k-deskflight` is
fixed by
[`ADR 0015`](docs/plan/adr/0015-helm-chart-distributions-form.md);
the first OCI push happens with the `v0.2.0` tag at M16. Until
then, install from the repository checkout:

```bash
helm install k-deskflight deploy/charts/k-deskflight/ \
    --namespace k-deskflight-system \
    --create-namespace \
    --set namespace.create=false
```

Full Helm-install operations doc:
[`docs/user/installation.md` §8](docs/user/installation.md).

## What the operator does

The operator ships a Custom Resource Definition `OpenDeskPreflightCheck`
(API group `k-deskflight.geo-terrain.net/v1alpha1`, namespaced). Operators
declare which cluster preconditions to check — e.g. Kubernetes minimum
version, IngressClass, StorageClasses, cert-manager availability, resource
floors. Results land structured in the CR status (`LH-F-007` / `LH-F-032`)
and, from v0.2 onward, optionally as a ConfigMap report with a YAML and a
Markdown key (`LH-F-028`, `ADR 0008`).

The operator is **read-only** with respect to the cluster (`LH-F-035`):
it does not install OpenDesk, does not change OpenDesk components, and
does not perform destructive actions (`LH-SYS-002..006`).

### Example — MVP profile

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

More examples and the target picture are in `LH-PROD-003a` / `LH-PROD-003b`
of the Lastenheft; ready-to-apply CR templates under
[`deploy/samples/`](deploy/samples/).

## Phase roadmap (state of the ADRs)

| Version | Content | Source |
| ------- | ------- | ------ |
| v0.1 (MVP) — **shipped 2026-05-20** | CRD, controller, K8s-version / StorageClass / IngressClass / cert-manager / resource / RBAC checks, container image, example manifests (`deploy/manifests/`), Prometheus `/metrics` endpoint with framework defaults | `LH-MVP-002`, `ADR 0005`, `ADR 0007` |
| v0.2 — **in progress** | Helm-Chart (M8, ready for closure); DNS / TLS / network reachability checks (M14/M15); events (M9); ConfigMap report (M10); own domain metrics (M11) + OTel tracing spans (M12, `AR-OP-006`); Node + ClusterIssuer checks (M13); release tag `v0.2.0` (M16) | `LH-PRI-002`, `ADR 0005`, `ADR 0007`, `ADR 0008`, `ADR 0010`, `ADR 0014`, `ADR 0015` |
| v0.3+ | PostgreSQL / S3 reachability (with auth), HTML report, additional profiles, kubectl plugin | `LH-PRI-003`, `ADR 0010` (with-auth block), follow-up ADR open |

## Supported Kubernetes versions

Rolling window over the three current Kubernetes minor versions with active
patch support (`ADR 0009`). State as of the ADR (2026-05-16): 1.34, 1.35,
1.36. The current matrix per operator release is documented in the release
note.

## Project artefacts and languages

| Path | Content | Language |
| ---- | ------- | -------- |
| `spec/lastenheft.md` | normative Lastenheft with `LH-*` identifiers | German |
| `docs/plan/adr/` | architecture decision records (ADRs) | German |
| `docs/plan/planning/` | roadmap, open triggers, completed slices | German |
| `docs/archive/` | superseded or rejected sketches | German |
| `README.md` (this file) | default project overview | English |
| `README.de.md` | German translation of `README.md` | German |
| [`CONTRIBUTING.md`](CONTRIBUTING.md), [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md), [`SECURITY.md`](SECURITY.md) | open-source conventions | English |
| Code, commit messages, issues, pull requests (from `LH-VM-004`) | implementation and community workflow | English |

Language policy lives in `LH-NF-021`. The German specification serves the
audience of public-sector and German-speaking operators (`LH-PK-004`); the
English README, code, and community workflow open the project to international
contributors.

## License

[MIT](LICENSE).

## Contributing

Contributions are welcome. Conventions, DCO sign-off (`git commit -s`),
Conventional Commits format and the language policy are in
[`CONTRIBUTING.md`](CONTRIBUTING.md).

Security vulnerabilities and Code of Conduct violations: please use
[`SECURITY.md`](SECURITY.md).

## Related sources

- OpenDesk project: https://docs.opendesk.eu/
- Kubernetes releases: https://kubernetes.io/releases/
- Contributor Covenant: https://www.contributor-covenant.org/version/2/1/
