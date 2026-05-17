# syntax=docker/dockerfile:1.7

# ---------------------------------------------------------------------------
# k-deskflight — OpenDesk Preflight Operator
#
# Docker-only workflow per docs/plan/planning/in-progress/
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
FROM golang:1.26.3 AS deps

WORKDIR /src
ENV GOFLAGS="-mod=readonly" \
    GOMODCACHE=/go/pkg/mod \
    GOCACHE=/root/.cache/go-build

COPY go.mod ./
# go.sum is copied if present (initial bootstrap may not have it yet);
# `go mod download` populates the cache from go.mod alone.
COPY go.su[m] ./

RUN mkdir -p "$GOMODCACHE" && go mod download

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

COPY --from=build /out/operator /usr/local/bin/operator

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/operator"]
