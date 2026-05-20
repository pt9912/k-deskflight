# syntax=docker/dockerfile:1.7

# ---------------------------------------------------------------------------
# k-deskflight — OpenDesk Preflight Operator
#
# Docker-only workflow per docs/plan/planning/done/
# slice-M1-repo-skeleton.md §2.1. Das Repository ist absichtlich
# Toolchain-frei: build/lint/test/coverage laufen über
#   docker build --target <stage> -t k-deskflight:go-<stage> .
#
# Stages:
#   deps      — Go-Modulauflösung (Cache-Layer).
#   compile   — Schnelles Compile-Feedback ohne Tests/Linter.
#   lint      — golangci-lint mit SOLID-nahem Pflicht-Profil
#               (ADR 0012 §2.2; .golangci.yml).
#   test      — `go test ./...`.
#   coverage  — `go test -coverprofile` + go tool cover -func +
#               scripts/coverage-gate.sh (ADR 0012 §2.5; M1 Smoketest-
#               Schwelle 0, M6 hebt auf 90).
#   build     — Statisch gelinkte Binary.
#   runtime   — distroless/static:nonroot.
#
# Pin-Politik: golang/golangci-lint-Versionen sind Routine-Pins
# (ADR 0012 §2.8 Abs. 3) — Hebung ohne ADR möglich, Begründung im
# Commit-Body.
# ---------------------------------------------------------------------------

# ---- deps ------------------------------------------------------------------
# AR-019 Step 1 (spec/architecture.md) fordert `ARG GO_VERSION` als
# expliziten Build-Pfad, damit Container- und Modul-Toolchain synchron
# pinnbar sind. Default folgt dem M1-Slice-Plan-Pin (§2.4); Hebung ist
# Routine ohne ADR (ADR 0012 §2.8 Abs. 3), neue Werte via
# `docker build --build-arg GO_VERSION=…` oder direktes Edit.
ARG GO_VERSION=1.26.3

FROM golang:${GO_VERSION} AS deps

WORKDIR /src
ENV GOFLAGS="-mod=readonly" \
    GOMODCACHE=/go/pkg/mod \
    GOCACHE=/root/.cache/go-build

COPY go.mod ./
# go.sum is copied if present. Der Glob `go.su[m]` ist eine Ein-
# Zeichen-Zeichenklasse, die `go.sum` matched, aber leer bleibt, wenn
# die Datei (noch) nicht existiert. Damit funktioniert dieselbe COPY-
# Zeile vor dem ersten `go mod tidy` (Bootstrap) und nach jedem späteren
# Stand — ohne Conditional und ohne `COPY --link`-Tricks. `go mod
# download` zieht die Module dann nur aus `go.mod`.
COPY go.su[m] ./

RUN mkdir -p "$GOMODCACHE" && go mod download

# ---- tools -----------------------------------------------------------------
# Stage für Generator-Tools (slice-M2 §2.3, architecture.md AR-007).
# Hier lebt `controller-gen` als statischer Tool-Pin. Hebung ist
# Routine ohne ADR (ADR 0012 §2.8 Abs. 3); Override via
# `make manifests CONTROLLER_GEN_VERSION=…`.
FROM deps AS tools

ARG CONTROLLER_GEN_VERSION=v0.21.0

RUN go install sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION}

# ---- smoke -----------------------------------------------------------------
# Cluster-Smoke-Stage (ADR 0013): pinnt `kind` und `kubectl` in einem
# Container, der über den gemounteten Host-Docker-Socket
# (Docker-out-of-Docker) `kindest/node`-Container auf der Host-Engine
# erzeugt. Damit hält make cluster-smoke die Docker-only-Konvention
# aus slice-M1 §2.1 ein — keine Host-Tool-Installation jenseits von
# Docker selbst.
#
# Pinning-Politik wie übrige Tool-Stages: ARG mit Default,
# `docker build --build-arg KIND_VERSION=…` zum Override.
# `alpine:3.20` ist klein (~5 MB) und bringt apk-paket-basiert
# `docker-cli` mit — kind shellt intern zu `docker` raus, deshalb
# ist die CLI Pflicht im smoke-Container.
FROM alpine:3.20 AS smoke

ARG KIND_VERSION=v0.31.0
ARG KUBECTL_VERSION=v1.34.0

RUN apk add --no-cache \
        bash \
        ca-certificates \
        curl \
        docker-cli \
        jq \
    && curl -fsSL "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64" -o /usr/local/bin/kind \
    && chmod +x /usr/local/bin/kind \
    && curl -fsSL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" -o /usr/local/bin/kubectl \
    && chmod +x /usr/local/bin/kubectl \
    && curl -fsSL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl.sha256" -o /tmp/kubectl.sha256 \
    && echo "$(cat /tmp/kubectl.sha256)  /usr/local/bin/kubectl" | sha256sum -c - \
    && rm /tmp/kubectl.sha256

WORKDIR /src

# ---- compile ---------------------------------------------------------------
FROM deps AS compile

COPY . .
RUN CGO_ENABLED=0 go build -o /tmp/operator ./cmd/operator

# ---- lint ------------------------------------------------------------------
# golangci-lint mit dem SOLID-nahen Pflicht-Profil aus ADR 0012 §2.2.
# Konfiguration in .golangci.yml; `//nolint`-Pragmas sind verboten
# (LH-QG-010), Carveouts werden zentral in der Config dokumentiert.
FROM golangci/golangci-lint:v2.12.1-alpine AS lint

WORKDIR /src
COPY --from=deps /go/pkg/mod /go/pkg/mod
COPY . .
RUN golangci-lint run ./...

# ---- test ------------------------------------------------------------------
FROM deps AS test

COPY . .
RUN CGO_ENABLED=0 go test ./...

# ---- coverage --------------------------------------------------------------
# Misst Coverage über produktive Pakete (Range-Selector grenzt cmd/
# bewusst aus; architecture.md AR-004). In M1 ist internal/ noch leer —
# der Stage setzt COVERAGE_BOOTSTRAP=1, scripts/coverage-gate.sh
# autorisiert genau dann eine leere Eingabe. Ab M2 (internal/ hat
# Code) ist COVERAGE_BOOTSTRAP=0, und ein fehlschlagender `go test`
# kann nicht mehr stillschweigend maskiert werden — `pipefail` lässt
# den `&&`-Chain abbrechen, fehlende `total:`-Zeile erzeugt Exit 2 im
# Gate. M6 hebt die Schwelle auf 90 % und macht das Gate PR-blockierend
# (Roadmap §3 M6).
FROM deps AS coverage

# SHELL mit -eo pipefail erzwingt, dass `go test … | tee …` den
# Exit-Code von `go test` durchreicht. Ohne pipefail wäre `tee`-OK das
# einzige, was zählt, und ein gecrashtes `go test` würde geräuschlos
# leere Coverage-Artefakte produzieren.
SHELL ["/bin/bash", "-eo", "pipefail", "-c"]

ARG COVERAGE_THRESHOLD=0
ENV COVERAGE_THRESHOLD=${COVERAGE_THRESHOLD}

COPY . .
RUN mkdir -p /out && \
    COVERPKG=$(go list ./internal/... 2>/dev/null | tr '\n' ',' | sed 's/,$//') && \
    if [ -z "$COVERPKG" ]; then \
        echo "coverage: no production packages in ./internal/... yet — bootstrap mode (M1)"; \
        : > /out/coverage.out; \
        : > /out/coverage-func.txt; \
        export COVERAGE_BOOTSTRAP=1; \
    else \
        CGO_ENABLED=0 go test \
            -coverpkg="$COVERPKG" \
            -coverprofile=/out/coverage.out \
            -covermode=atomic \
            ./... && \
        go tool cover -func=/out/coverage.out | tee /out/coverage-func.txt; \
        export COVERAGE_BOOTSTRAP=0; \
    fi && \
    bash scripts/coverage-gate.sh /out/coverage-func.txt "$COVERAGE_THRESHOLD"

# ---- build -----------------------------------------------------------------
FROM deps AS build

COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags="-s -w" \
    -o /out/operator \
    ./cmd/operator

# ---- runtime ---------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# OCI image labels. The `source` label is what GitHub uses to auto-
# link the GHCR package to this repository (Packages sidebar on the
# repo home page). The first v0.1.0 image was pushed before this
# label was added — for that release the repo↔package link must be
# set manually via the Package-Settings web UI; v0.1.1+ inherits the
# link automatically.
LABEL org.opencontainers.image.source="https://github.com/pt9912/k-deskflight" \
      org.opencontainers.image.description="OpenDesk Preflight Operator — Kubernetes-native pre-installation checks for OpenDesk deployments." \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.title="k-deskflight" \
      org.opencontainers.image.vendor="pt9912" \
      org.opencontainers.image.documentation="https://github.com/pt9912/k-deskflight/blob/main/docs/user/installation.md"

COPY --from=build /out/operator /usr/local/bin/operator

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/operator"]
