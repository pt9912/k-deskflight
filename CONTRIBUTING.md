# Contributing to k-deskflight

Thanks for considering a contribution. This document describes the conventions
and the workflow.

## Project Status

`k-deskflight` is in the **specification phase** (V-model `LH-VM-001`). The
authoritative requirements live in [`spec/lastenheft.md`](spec/lastenheft.md);
architecture decisions are recorded under [`docs/plan/adr/`](docs/plan/adr/).
The MVP (`v0.1`, `LH-REL-001`) is not yet implemented. Until the first code
commit (`LH-VM-004`), substantive contributions are best directed at reviewing
ADRs and the requirements specification.

## Code of Conduct

This project adopts the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md),
version 2.1. By participating, you are expected to uphold it.

## License and Developer Certificate of Origin

The project is licensed under the **MIT License** (see [`LICENSE`](LICENSE)).

Contributions follow the **Developer Certificate of Origin** (DCO, see
https://developercertificate.org/). Every commit must carry a Sign-off line:

```
Signed-off-by: Your Name <your.email@example.org>
```

Use `git commit -s` to add it automatically — the values come from your
`git config user.name` and `git config user.email`. By signing off, you certify
that you have the right to submit the contribution under the project license.

Pull requests with unsigned commits will be asked to rebase with sign-off.
Commits made before `ADR 0011` (commit `d3aab77`) are grandfathered.

## Languages

Per `LH-NF-021`:

- **English** — source code, identifiers, code comments, issues, pull
  requests, commit messages, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`,
  `SECURITY.md`, `README.md` (default entry point), user-visible operator
  output (conditions, reasons, messages, events, logs).
- **German** — `README.de.md` (German translation of `README.md`, kept
  1:1 symmetric in structure and detail), `Lastenheft`, `Pflichtenheft`,
  technical specification documents in `spec/`, ADR contents in
  `docs/plan/adr/`, planning artefacts in `docs/plan/`.

If you change a German specification document, the commit message stays
English. If you change `README.md`, mirror the change in `README.de.md`
in the same pull request.

## Commit Convention

Commits follow **Conventional Commits**: `type(scope): description`.

Allowed `type` values: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`,
`build`, `ci`, `perf`, `style`. The `scope` follows the repository layout and
is optional. Examples:

```
docs(plan): ADR 0012 closes ...
feat(operator): add storage class probe
fix(rbac): tighten RoleBinding to namespace scope
```

Bodies are wrapped at ~72 characters; the subject line stays under 70.

## Branch and Pull Request Workflow

- `main` is the only long-running branch. No `develop`, no Git-Flow.
- Feature work lives on short-lived branches (`feat/...`, `fix/...`,
  `docs/...`) and is merged into `main` via Pull Request.
- One logical change per PR. Squash-merge is preferred for small PRs;
  merge commits are acceptable for larger feature branches that have
  meaningful intermediate commits.
- All PRs require at least one maintainer review. Until additional maintainers
  are appointed, that is the project owner per `LH-PROD-001`.

## Architecture Decisions

Architecture Decision Records (ADRs) live in
[`docs/plan/adr/`](docs/plan/adr/). Read the following before contributing a
new ADR:

- `ADR 0001` — documentation and planning structure
- `ADR 0002` — ADR lifecycle, header schema, status transitions
- `ADR 0003` — cross-reference conventions (identifiers over section numbers)

File naming `NNNN-short-title.md` is compatible with
[adr-tools](https://github.com/npryce/adr-tools). ADR content is German
following `LH-NF-021`.

## Issue Reporting

For general issues, use the issue tracker.

For security vulnerabilities or Code of Conduct violations, follow
[`SECURITY.md`](SECURITY.md).

## Releases

Releases are tagged annotated Git tags `vX.Y.Z` (SemVer 2.0.0). The release
approval workflow (Makefile target, release guard script) will be added
during the build/release implementation phase per `LH-VM-002`.
