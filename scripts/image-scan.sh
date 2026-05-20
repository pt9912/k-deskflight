#!/usr/bin/env bash
# Trivy CRITICAL/HIGH + MEDIUM scan against the GHCR-tagged operator
# image (slice-M7 §2.4, ADR 0012 §2.9).
#
# Two-pass scan policy:
#   - Pass 1 (CRITICAL,HIGH, --exit-code 1): release-blocking.
#   - Pass 2 (MEDIUM,        --exit-code 0): informational report.
#
# Inputs:
#   $1 (or VER env): semantic version without the `v` prefix
#                    (e.g. 0.1.0; the script prepends `v` for the tag).
#
# Optional env overrides:
#   IMAGE_REPO   — full image repo path (default ghcr.io/pt9912/k-deskflight).
#   TRIVY_IMAGE  — Trivy container image pin (default aquasec/trivy:0.59.1).
#   SKIP_BUILD=1 — skip the image-build prereq (assumes the tag exists).
#
# Maintainer-only: this script mounts the Docker daemon socket into the
# Trivy container so Trivy can pull locally-built images by tag. That
# gives the Trivy container full Docker daemon access; the pattern is
# acceptable for a maintainer-/CI-only tool but MUST NOT be wrapped
# into any user-facing entry point.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

VER="${1:-${VER:-}}"
if [[ -z "${VER}" ]]; then
    echo "image-scan: VER=X.Y.Z is required (e.g. bash scripts/image-scan.sh 0.1.0)" >&2
    exit 2
fi

IMAGE_REPO="${IMAGE_REPO:-ghcr.io/pt9912/k-deskflight}"
TRIVY_IMAGE="${TRIVY_IMAGE:-aquasec/trivy:0.59.1}"
IMAGE_TAG="${IMAGE_REPO}:v${VER}"

if [[ "${SKIP_BUILD:-0}" != "1" ]]; then
    make --no-print-directory image-build VER="${VER}"
fi

mkdir -p .security/.trivy-cache
bash scripts/render-trivyignore.sh

CACHE_MOUNT="${ROOT_DIR}/.security/.trivy-cache:/root/.cache/trivy"
IGNORE_MOUNT="${ROOT_DIR}/.security/.trivyignore:/work/.trivyignore:ro"
SOCK_MOUNT="/var/run/docker.sock:/var/run/docker.sock"

echo "[image-scan] Scanning ${IMAGE_TAG} for CRITICAL/HIGH (release-blocking)..."
docker run --rm \
    -v "${SOCK_MOUNT}" \
    -v "${CACHE_MOUNT}" \
    -v "${IGNORE_MOUNT}" \
    "${TRIVY_IMAGE}" image \
        --severity CRITICAL,HIGH \
        --exit-code 1 \
        --no-progress \
        --ignorefile /work/.trivyignore \
        "${IMAGE_TAG}"

echo "[image-scan] CRITICAL/HIGH: clean. Reporting MEDIUM findings (non-blocking)..."
docker run --rm \
    -v "${SOCK_MOUNT}" \
    -v "${CACHE_MOUNT}" \
    -v "${IGNORE_MOUNT}" \
    "${TRIVY_IMAGE}" image \
        --severity MEDIUM \
        --exit-code 0 \
        --no-progress \
        --ignorefile /work/.trivyignore \
        "${IMAGE_TAG}" || true
