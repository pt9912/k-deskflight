#!/usr/bin/env bash
# Pre-release consistency guard (slice-M7 §2.5, ADR 0011 §2.5).
# Invoked from `make release-guard VER=X.Y.Z` or directly from
# scripts/release-guard.sh — all checks must pass before the
# annotated git tag `vX.Y.Z` may be created in a follow-up step
# (slice-M7 §4 step 15).
#
# The script ONLY validates. It does not create any tag, push any
# ref, or modify the working tree. Tag creation is a separate
# explicit `git tag -a` step.

set -euo pipefail

usage() {
    cat <<'USAGE'
Usage: scripts/release-guard.sh X.Y.Z

Required approval:
  K_DESKFLIGHT_RELEASE_APPROVED=1

Optional test overrides (LOCAL GUARD TESTS ONLY — do not use these
in real release runs; they exist so scripts/test-release-guard.sh
can exercise the failure paths against synthetic fixtures):
  K_DESKFLIGHT_RELEASE_ALLOW_NON_MAIN=1   skip the `branch == main` check
  K_DESKFLIGHT_RELEASE_ALLOW_DIRTY=1      skip the working-tree-clean check
  K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE=1    skip the `git ls-remote` network probe
  K_DESKFLIGHT_RELEASE_DRY_RUN=1          report success as "dry-run ok" instead of "ok"
USAGE
}

fail() {
    echo "release-guard: $*" >&2
    exit 1
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
fi

version="${1:-}"
if [ -z "$version" ]; then
    usage >&2
    exit 2
fi
shift
# Trailing arguments are almost always a typo (e.g.
# `make release-guard VER="0.1.0 --dry-run"` interpolates the
# whole string into $1). Reject them explicitly so the user gets a
# direct hint instead of a misleading version-format error.
if [ "$#" -gt 0 ]; then
    fail "unexpected extra arguments: $*"
fi

# Reject a v-prefix explicitly so callers cannot ambiguously pass
# both v0.1.0 and 0.1.0 — the script always prepends `v` itself.
case "$version" in
    v*) fail "pass the version without v-prefix, got $version" ;;
esac

if ! [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]; then
    fail "version must look like X.Y.Z or X.Y.Z-PRE, got $version"
fi

tag="v$version"

# Surface every active override on stderr — these env vars exist
# only for the local guard tests; a real release run must never see
# them. The warnings make accidental shell-history re-use visible
# in the build log instead of silently passing the check.
warn_override() {
    echo "release-guard: WARNING: $1 active — local-test override, not safe for real releases" >&2
}
[ "${K_DESKFLIGHT_RELEASE_ALLOW_NON_MAIN:-}" = "1" ] && warn_override K_DESKFLIGHT_RELEASE_ALLOW_NON_MAIN
[ "${K_DESKFLIGHT_RELEASE_ALLOW_DIRTY:-}" = "1" ]    && warn_override K_DESKFLIGHT_RELEASE_ALLOW_DIRTY
[ "${K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE:-}" = "1" ]  && warn_override K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE

# 1. Manual approval — the operator must declare intent.
[ "${K_DESKFLIGHT_RELEASE_APPROVED:-}" = "1" ] || \
    fail "manual approval missing: set K_DESKFLIGHT_RELEASE_APPROVED=1"

# 2. Branch — releases come from main only (ADR 0011 §2.5).
branch="$(git rev-parse --abbrev-ref HEAD)"
if [ "$branch" != "main" ] && [ "${K_DESKFLIGHT_RELEASE_ALLOW_NON_MAIN:-}" != "1" ]; then
    fail "release must run on main (current: $branch)"
fi

# 3. Working tree — no uncommitted changes.
if [ -n "$(git status --porcelain)" ] && [ "${K_DESKFLIGHT_RELEASE_ALLOW_DIRTY:-}" != "1" ]; then
    fail "working tree must be clean"
fi

# 4a. Local tag — must not already exist.
if git rev-parse -q --verify "refs/tags/$tag" >/dev/null; then
    fail "tag already exists locally: $tag"
fi

# 4b. Origin tag — must not already exist on the remote either.
# `git ls-remote --exit-code` returns 0 when the ref exists, 2 when
# it does not match, and other non-zero codes for network/auth
# errors. The else-branch distinguishes the "no such tag" success
# case from a real network failure that requires explicit override.
if git ls-remote --exit-code --tags origin "refs/tags/$tag" >/dev/null 2>&1; then
    fail "tag already exists on origin: $tag"
else
    ls_remote_status=$?
    if [ "$ls_remote_status" -ne 2 ] && [ "${K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE:-}" != "1" ]; then
        fail "could not verify tag on origin (set K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE=1 only for local guard tests)"
    fi
fi

# 5. CHANGELOG.md — section for this version must exist. The
# CHANGELOG is the canonical release-notes source per slice-M7 §2.6.
if [ ! -f CHANGELOG.md ]; then
    fail "CHANGELOG.md is missing — required for release notes (slice-M7 §2.6)"
fi
if ! grep -q "## \\[$version\\]" CHANGELOG.md; then
    fail "CHANGELOG.md has no section for [$version]"
fi

if [ "${K_DESKFLIGHT_RELEASE_DRY_RUN:-}" = "1" ]; then
    echo "release-guard: dry-run ok for $tag"
else
    echo "release-guard: ok for $tag"
fi
