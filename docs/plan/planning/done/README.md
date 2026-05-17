# Abgeschlossene Pläne

Dieses Verzeichnis sammelt Closure-Notizen zu abgeschlossenen
Meilensteinen und Plänen.

Eine Closure-Notiz fasst zusammen:

- was wurde geliefert (Code, Specs, ADRs),
- welche Lastenheft-Kennungen sind damit umgesetzt,
- was wurde explizit nicht erledigt und wandert weiter (`../open/`,
  `../next/` oder Folge-Meilenstein),
- Verweis auf Tag/Release im CHANGELOG (sobald vorhanden).

---

## Bestand

| Datei | Geschlossen | Lieferziel |
| ----- | ----------- | ---------- |
| [`slice-M1-repo-skeleton.md`](slice-M1-repo-skeleton.md) | 2026-05-17 | Go-Modul-Skeleton, Verzeichnis-Layout, Multi-Stage `Dockerfile`, `Makefile` (Docker-only), `.golangci.yml` (5 Default + 24 SOLID Linter), Doc-Refs- und Coverage-Gate-Skripte, GitHub-Actions-CI mit `gates` + `security-gates` parallel. Alle lokal verifizierbaren Abnahmekriterien aus §7 grün. |
| [`slice-M2-crd-controller-skeleton.md`](slice-M2-crd-controller-skeleton.md) | 2026-05-17 | CRD `OpenDeskPreflightCheck` v1alpha1 (kubebuilder-Marker, controller-gen-Pipeline mit `tools`-Stage), Reconciler-Skelett (Pending→Running→Passed minimal, fake-client-Tests mit 83 % Coverage), controller-runtime v0.24.1, depguard nach AR-005 scharf, Generated-Drift-Gate aktiv in `make gates`, deploy/manifests + sample CR. Items §7 #1–#8 lokal grün; #9–#11 observational (kind-/minikube-Attest). |
