# Security Policy

## Reporting a Vulnerability

If you discover a vulnerability in `k-deskflight`, please report it through
the **GitHub Security Advisories** workflow on this repository:

1. Navigate to the **Security** tab of the repository.
2. Open **Advisories** → **Report a vulnerability**.
3. Provide a clear description, reproduction steps if available, and any
   relevant version or commit references.

This channel is private; only project maintainers can see your report. Please
do not file public issues or pull requests that disclose the vulnerability.

## Disclosure Timeline

We follow a 90-day **coordinated disclosure** policy:

- We acknowledge receipt of a report within **5 working days**.
- We aim to provide an initial assessment and a fix plan within **30 days**.
- A fix is released no later than **90 days** after the report, unless we
  agree with the reporter on an extension.

Once a fix is released, we publish a public security advisory crediting the
reporter (unless the reporter requests anonymity).

## Scope

This policy covers:

- The k-deskflight operator code (once published with the first code commit
  per `LH-VM-004`).
- The deployed Kubernetes manifests in the operator repository (CRD, RBAC,
  Deployment, examples).
- The Helm chart once released (v0.2+, see `ADR 0005`).

This policy does **not** cover:

- Vulnerabilities in upstream Kubernetes, `controller-runtime`, or third-party
  dependencies — please report those directly to the respective projects.
- Vulnerabilities in OpenDesk components or in external services (PostgreSQL,
  S3 providers) that `k-deskflight` only inspects.

## Maintainer

The initial maintainer is the project owner per `LH-PROD-001`. The GitHub
Security Advisory workflow notifies the maintainer team automatically.

## Code of Conduct Violations

Reports of Code of Conduct violations (see [`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md))
use the same private channel. Prefix the advisory title with `Code of Conduct:`
so the report is routed correctly.

## Hosting Note

This policy assumes GitHub hosting (see `ADR 0011 §2.6`). A future change of
the hosting platform would require a superseding ADR and a corresponding
update to this file.
