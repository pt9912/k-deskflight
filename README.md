# k-deskflight

**OpenDesk Preflight Operator** — Kubernetes operator that performs preflight
checks on a cluster before an [OpenDesk](https://docs.opendesk.eu/) installation.

> **Sprachversion:** Die deutsche Variante dieses README liegt unter
> [`README.de.md`](README.de.md).

## Status

The project is in the **specification phase** of the V-model (`LH-VM-001`).
There is no code yet — the requirements are documented, every architecture
decision for the MVP scope is taken, and implementation begins with
`LH-VM-004`.

| Phase | Status | Source |
| ----- | ------ | ------ |
| Lastenheft (`LH-VM-001`) | Draft 0.1.0 | [`spec/lastenheft.md`](spec/lastenheft.md) |
| Architecture decisions | 11 ADRs | [`docs/plan/adr/`](docs/plan/adr/) |
| Pflichtenheft (`LH-VM-002`) | open | follows |
| Implementation (`LH-VM-004`) | open | follows |

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
of the Lastenheft.

## Phase roadmap (state of the ADRs)

| Version | Content | Source |
| ------- | ------- | ------ |
| v0.1 (MVP) | CRD, controller, K8s-version / StorageClass / IngressClass / cert-manager / resource / RBAC checks, container image, example manifests (`deploy/manifests/`), Prometheus `/metrics` endpoint with framework defaults | `LH-MVP-002`, `ADR 0005`, `ADR 0007` |
| v0.2 | DNS / TLS / network reachability checks; events; ConfigMap report; own domain metrics; Helm chart | `LH-PRI-002`, `ADR 0005`, `ADR 0007`, `ADR 0008`, `ADR 0010` |
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
