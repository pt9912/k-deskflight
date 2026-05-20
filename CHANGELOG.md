# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

<!-- Sub-section placeholders for the next release follow the same
order as the [0.1.0] section below: Added, Changed, Deprecated,
Removed, Fixed, Security. -->

## [0.1.0] - 2026-05-XX <!-- DATE_PLACEHOLDER -->

> **MVP release** of the OpenDesk Preflight Operator
> (`k-deskflight`). The release closes the seven-slice roadmap in
> [`docs/plan/planning/done/roadmap.md`](docs/plan/planning/done/roadmap.md)
> and delivers the
> [`LH-PRI-001`](spec/lastenheft.md) must-have requirements plus
> the [`LH-AK-001..-016`](spec/lastenheft.md) acceptance criteria
> against [Kubernetes 1.34 and newer](docs/plan/adr/0009-k8s-versions-support-und-profile-mindestversionen.md).
>
> The container image
> `ghcr.io/pt9912/k-deskflight:v0.1.0` is the release artifact;
> raw manifests under [`deploy/manifests/`](deploy/manifests/)
> install the operator without a Helm chart
> ([ADR 0005](docs/plan/adr/0005-helm-chart-nicht-im-mvp.md)).
> See [`docs/user/installation.md`](docs/user/installation.md)
> for the apply flow and
> [`docs/user/cr-examples.md`](docs/user/cr-examples.md) for
> production / evaluation CR templates.
>
> Pull the image:
>
> ```bash
> docker pull ghcr.io/pt9912/k-deskflight:v0.1.0
> ```
>
> This is the very first tag of the project. The preceding spec,
> plan and ADR commits (2026-05-15 .. 2026-05-19) are part of the
> same logical artifact but are not enumerated here — the
> roadmap closure under
> [`docs/plan/planning/done/`](docs/plan/planning/done/)
> carries the full slice-by-slice trail. Out-of-MVP scope (Helm
> chart, ServiceMonitor stack, external service checks for
> PostgreSQL / S3 / DNS / TLS, custom domain metrics,
> multi-maintainer governance) is tracked in
> [`docs/plan/planning/open/`](docs/plan/planning/open/) and the
> v0.2+ section of the roadmap.

### Added

- **CRD `OpenDeskPreflightCheck`**
  (`k-deskflight.geo-terrain.net/v1alpha1`, namespaced) with
  `spec.profile`, `spec.interval`, `spec.checks.{kubernetesVersion,
  storageClass, ingressClass, certManager, clusterResources}`, plus
  the AR-006 status schema (`phase`, `conditions[]` with severity,
  `summary`, `observedGeneration`, `lastChecked`). See
  [`done/slice-M2-crd-controller-skeleton.md`](docs/plan/planning/done/slice-M2-crd-controller-skeleton.md).
- **Reconciler** that drives the CR through
  `Pending → Running → Passed | Warning | Failed | Unknown` with
  per-check aggregation, deterministic condition ordering, and a
  `spec.interval`-driven requeue (default `5m`, bounds
  `[30s, 24h]`, normaliser + `ConfigurationInvalid` warning for
  out-of-range or unparseable values).
  [`AR-009`](spec/architecture.md), [`AR-010.1`](spec/architecture.md).
- **Five MVP checks**
  ([`LH-AK-005..-009`](spec/lastenheft.md)):
  - `kubernetesVersion` against
    [`spec.checks.kubernetesVersion.min`](docs/user/cr-examples.md)
    (default `1.34`, operator floor per
    [`ADR 0009 §2.2`](docs/plan/adr/0009-k8s-versions-support-und-profile-mindestversionen.md)).
    [`done/slice-M3-kubernetes-version-check.md`](docs/plan/planning/done/slice-M3-kubernetes-version-check.md).
  - `storageClass` presence-of-names plus optional default-class
    detection (`requireDefault`), tolerating both the GA
    annotation and the legacy beta key.
    [`done/slice-M4-cluster-state-checks.md`](docs/plan/planning/done/slice-M4-cluster-state-checks.md).
  - `ingressClass` presence-of-names.
  - `certManager` API-group registration (presence-only;
    ClusterIssuer detail validation is v0.2 per
    [`ADR 0010`](docs/plan/adr/0010-externe-dienstpruefungen-und-secret-mechanik.md)).
  - `clusterResources` allocatable-CPU/memory summation across
    Ready nodes, with per-profile defaults (production 4 CPU /
    8 Gi, evaluation 2 CPU / 4 Gi).
- **RBAC self-check** (`LH-F-024`, `LH-AK-016`): each enabled
  check runs a `SelfSubjectAccessReview` before its main path
  and reports `RBACInsufficient` (denied) or `RBACCheckFailed`
  (subsystem error) with `Status=Unknown` rather than aborting
  the whole reconcile.
  [`done/slice-M5-rbac-self-check-robustness.md`](docs/plan/planning/done/slice-M5-rbac-self-check-robustness.md).
- **Per-check fault isolation** (`LH-NF-005`, `LH-AK-010`): each
  check runs under a `defer`/`recover` with per-check timeout;
  panics or hangs in one check do not bring down the operator or
  starve sibling checks. Outer reconciler also has a best-effort
  status-write recover so the CR always reflects the latest known
  state.
- **Secret-output filter** (`LH-SEC-002`, `LH-NF-007`,
  `LH-AK-012`): `SanitizeMessage` + `SanitizeAttrs` + `LogResult`
  wrap status writes, structured log lines and the
  `SelfSubjectAccessReview` error path so secrets cannot leak
  into `kubectl get -o yaml` or container logs.
- **Prometheus `/metrics` endpoint** with
  controller-runtime framework defaults, exposed via the
  `k-deskflight-operator-metrics:8080` `Service`
  ([`LH-SST-004`](spec/lastenheft.md), [`ADR 0007`](docs/plan/adr/0007-prometheus-metrik-scope-im-mvp.md)).
  The companion `ClusterRole`
  `k-deskflight-metrics-scrape` is shipped as a pattern asset for
  Prometheus-Operator stacks — explicitly **unauthenticated** in
  v0.1; auth-filter activation is v0.2 with a follow-up ADR.
  [`done/slice-M6-metrics-tests-doku.md`](docs/plan/planning/done/slice-M6-metrics-tests-doku.md).
- **Leader election** (`AR-026`): the controller-runtime manager
  runs with `LeaderElection=true` against a
  `coordination.k8s.io/lease` named `k-deskflight-operator` in
  the operator namespace (`POD_NAMESPACE` via Downward-API).
  `--leader-elect=false` is supported as a single-pod debug mode
  with an `--expected-replica-count` topology guard. Default
  Deployment ships with `replicas: 1`; multi-replica HA tuning is
  v0.2.
- **User documentation** under
  [`docs/user/`](docs/user/): installation
  (raw-manifest apply, namespace override, metrics scrape
  binding), CR examples with profile-default explanation,
  conditions catalog (reason codes plus severity) and a
  troubleshooting handbook. Two ready-to-apply CR templates ship
  under [`deploy/samples/`](deploy/samples/).
- **Quality-gate suite**
  ([ADR 0012](docs/plan/adr/0012-quality-gates.md),
  [`LH-QG-001..009`](spec/lastenheft.md)): `make gates` bundles
  lint (5 default + 24 SOLID-adjacent linters with the depguard
  hexagonal-boundary rules from `AR-005`), tests, 90 % line
  coverage over `internal/`, doc-refs, and the controller-gen
  drift check. `make security-gates` runs `govulncheck`
  (function-based) and, with `VER` set, the Trivy image scan
  with severity policy `CRITICAL`/`HIGH` blocking and `MEDIUM`
  informational. Vulnignore entries carry a mandatory `expires`
  date that breaks the build when stale
  ([`ADR 0012 §2.8 abs 3`](docs/plan/adr/0012-quality-gates.md)).
- **kind-based cluster smoke** (`ADR 0013`,
  `make cluster-smoke`): end-to-end attestation of CRD install,
  operator startup, all five checks against either the smoke
  cluster-state stubs or the kind defaults, HTTP probes
  (`/healthz`, `/readyz`, `/metrics`), Service-DNS metrics
  scrape, ClusterRole structure, and Leader-Election lease
  existence.
- **Release tooling**: `make image-build VER=`,
  `make image-publish-{dry-run,guard,}` and `make image-scan VER=`
  for the GHCR-tagged image flow
  ([`done/slice-M6-metrics-tests-doku.md`](docs/plan/planning/done/slice-M6-metrics-tests-doku.md)
  baseline; M7 wires the GHCR pattern). `make release-guard
  VER=` enforces approval, branch, working-tree, tag-existence,
  and CHANGELOG-section checks before the annotated
  `vX.Y.Z` tag is created. All release-side env vars carry the
  `K_DESKFLIGHT_*` prefix.

### Changed

- **`api/v1alpha1.OpenDeskPreflightCheckSpec.Interval` is no
  longer nullable**: the type narrowed from `*string` to `string`
  (commit `dc4a14d`, M6 Step-1 Review Round-2 §4 finding 6). The
  CRD-schema default `5m` continues to apply unchanged via
  `+kubebuilder:default="5m"`. User-visible effect only when
  constructing a CR programmatically without API-server defaulting
  (e.g. raw `Unstructured` clients) — declarative manifests and
  `kubectl apply` are unaffected.

<!--
GitHub compare/tag URLs below resolve only after the v0.1.0 tag
is pushed (slice-M7 §4 step 15). Until then they return 404 — the
same expectation pattern as the raw.githubusercontent reference in
docs/user/installation.md §6.
-->

[Unreleased]: https://github.com/pt9912/k-deskflight/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/pt9912/k-deskflight/releases/tag/v0.1.0
