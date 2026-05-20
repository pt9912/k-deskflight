#!/usr/bin/env bash
# Failure-path tests for scripts/release-guard.sh (slice-M7 §2.5).
#
# Each case sets up a synthetic temp-dir repository with a copy of
# release-guard.sh and the minimum fixtures the guard expects
# (CHANGELOG.md with a `## [9.9.9]` section, an `origin` remote
# pointing at an unreachable URL), then exercises the guard with
# the env-var overrides that the production release path may never
# combine. Not run in CI by convention (m-trace pattern).

set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

fail() {
    echo "release-guard-test: $*" >&2
    exit 1
}

setup_repo() {
    local name="$1"
    local dir="$tmp_dir/$name"
    mkdir -p "$dir/scripts"
    cp "$root_dir/scripts/release-guard.sh" "$dir/scripts/release-guard.sh"
    cat >"$dir/CHANGELOG.md" <<'EOF'
# Changelog

## [9.9.9] - 2099-01-01

### Added

- Synthetic fixture entry for release-guard test.
EOF
    (
        cd "$dir"
        git init -q
        git branch -M main
        git add .
        git -c user.name="Release Guard Test" \
            -c user.email="release-guard-test@example.invalid" \
            commit -q -m init
        # Unreachable origin so `git ls-remote` exits with a network
        # error; ALLOW_OFFLINE override skips the failure case.
        git remote add origin https://example.invalid/k-deskflight.git
    )
    printf '%s\n' "$dir"
}

expect_success() {
    local name="$1"
    shift
    local output
    if ! output="$("$@" 2>&1)"; then
        printf '%s\n' "$output" >&2
        fail "$name: expected success"
    fi
}

expect_failure() {
    local name="$1"
    local expected="$2"
    shift 2
    local output
    if output="$("$@" 2>&1)"; then
        printf '%s\n' "$output" >&2
        fail "$name: expected failure"
    fi
    case "$output" in
        *"$expected"*) ;;
        *)
            printf '%s\n' "$output" >&2
            fail "$name: expected output to contain '$expected'"
            ;;
    esac
}

run_guard() {
    local repo="$1"
    shift
    (
        cd "$repo"
        "$@"
    )
}

# 1. Happy path — dry-run with all required env vars.
repo="$(setup_repo success)"
expect_success "approved dry-run" \
    run_guard "$repo" \
    env K_DESKFLIGHT_RELEASE_APPROVED=1 \
        K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE=1 \
        K_DESKFLIGHT_RELEASE_DRY_RUN=1 \
    bash scripts/release-guard.sh 9.9.9

# 2. Approval missing.
repo="$(setup_repo missing-approval)"
expect_failure "missing approval" "manual approval missing" \
    run_guard "$repo" \
    bash scripts/release-guard.sh 9.9.9

# 3. v-prefix rejected.
repo="$(setup_repo v-prefix)"
expect_failure "v-prefix" "without v-prefix" \
    run_guard "$repo" \
    env K_DESKFLIGHT_RELEASE_APPROVED=1 \
        K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE=1 \
    bash scripts/release-guard.sh v9.9.9

# 4. Malformed version.
repo="$(setup_repo malformed-version)"
expect_failure "malformed version" "version must look like" \
    run_guard "$repo" \
    env K_DESKFLIGHT_RELEASE_APPROVED=1 \
        K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE=1 \
    bash scripts/release-guard.sh 1.0

# 5. Non-main branch.
repo="$(setup_repo non-main)"
(
    cd "$repo"
    git switch -q -c feature/release-guard-test
)
expect_failure "non-main" "release must run on main" \
    run_guard "$repo" \
    env K_DESKFLIGHT_RELEASE_APPROVED=1 \
        K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE=1 \
    bash scripts/release-guard.sh 9.9.9

# 6. Non-main with explicit override.
expect_success "non-main override" \
    run_guard "$repo" \
    env K_DESKFLIGHT_RELEASE_APPROVED=1 \
        K_DESKFLIGHT_RELEASE_ALLOW_NON_MAIN=1 \
        K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE=1 \
    bash scripts/release-guard.sh 9.9.9

# 7. Dirty working tree.
repo="$(setup_repo dirty)"
printf '\n' >>"$repo/CHANGELOG.md"
expect_failure "dirty tree" "working tree must be clean" \
    run_guard "$repo" \
    env K_DESKFLIGHT_RELEASE_APPROVED=1 \
        K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE=1 \
    bash scripts/release-guard.sh 9.9.9

# 8. Local tag already exists.
repo="$(setup_repo local-tag)"
(
    cd "$repo"
    git tag v9.9.9
)
expect_failure "local tag" "tag already exists locally" \
    run_guard "$repo" \
    env K_DESKFLIGHT_RELEASE_APPROVED=1 \
        K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE=1 \
    bash scripts/release-guard.sh 9.9.9

# 9. Offline (unreachable origin) without override.
repo="$(setup_repo offline-no-override)"
expect_failure "offline no override" "could not verify tag on origin" \
    run_guard "$repo" \
    env K_DESKFLIGHT_RELEASE_APPROVED=1 \
    bash scripts/release-guard.sh 9.9.9

# 10. Missing CHANGELOG entry.
repo="$(setup_repo missing-changelog-section)"
(
    cd "$repo"
    rm CHANGELOG.md
    cat >CHANGELOG.md <<'EOF'
# Changelog

## [1.2.3] - 2099-01-01
EOF
    git add CHANGELOG.md
    git -c user.name="Release Guard Test" \
        -c user.email="release-guard-test@example.invalid" \
        commit -q -m "rewrite changelog without the test version"
)
expect_failure "missing changelog section" "CHANGELOG.md has no section for [9.9.9]" \
    run_guard "$repo" \
    env K_DESKFLIGHT_RELEASE_APPROVED=1 \
        K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE=1 \
    bash scripts/release-guard.sh 9.9.9

# 11. Missing CHANGELOG.md file altogether.
repo="$(setup_repo missing-changelog-file)"
(
    cd "$repo"
    git rm -q CHANGELOG.md
    git -c user.name="Release Guard Test" \
        -c user.email="release-guard-test@example.invalid" \
        commit -q -m "drop CHANGELOG entirely"
)
expect_failure "missing changelog file" "CHANGELOG.md is missing" \
    run_guard "$repo" \
    env K_DESKFLIGHT_RELEASE_APPROVED=1 \
        K_DESKFLIGHT_RELEASE_ALLOW_OFFLINE=1 \
    bash scripts/release-guard.sh 9.9.9

echo "release-guard-test: ok"
