// Package main is the operator entry point.
//
// In slice M1 (repo & build skeleton) this is a smoke binary: it prints
// a self-identification line on stdout and exits with status 0. The
// controller-runtime wiring, signal handling, OTel/metrics setup and CRD
// scheme registration come in slice M2 once the CRD types exist (see
// spec/architecture.md §AR-003 and docs/plan/planning/in-progress/
// slice-M1-repo-skeleton.md §8 out-of-scope).
package main

import (
	"log/slog"
	"os"
)

const productName = "k-deskflight"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("operator skeleton starting", slog.String("product", productName))
}
