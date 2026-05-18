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
| [`slice-M2-crd-controller-skeleton.md`](slice-M2-crd-controller-skeleton.md) | 2026-05-17 | CRD `OpenDeskPreflightCheck` v1alpha1 (kubebuilder-Marker, controller-gen-Pipeline mit `tools`-Stage), Reconciler-Skelett (Pending→Running→Passed mit M3-Fix), controller-runtime v0.24.1, depguard nach AR-005 scharf, Generated-Drift-Gate aktiv, deploy/manifests + sample CR. **Alle §7-Items inkl. LH-AK-001/-002/-003/-004/-011 attestiert** (lokal via Tests, real via Cluster-Smoke nach ADR 0013). |
| [`slice-M3-kubernetes-version-check.md`](slice-M3-kubernetes-version-check.md) | 2026-05-17 | Erste echte Check-Implementierung: `KubernetesVersionCheck` vergleicht `discovery.ServerVersion()` mit `spec.checks.kubernetesVersion.min`. AR-012 Check-Interface, AR-013 Registry (Port + Map-Adapter), AR-014 Aggregator (Severity→Phase + Conditions-Dedupe/Sort). Reconciler durchläuft minimal-sequenziellen AR-009-Pfad (Phasen 1+2+3+4+5+6 ohne Worker-Pool). Coverage 86.1 %. **LH-AK-005 inkl. realer Cluster-Verifikation** über `cluster-smoke`-Workflow (ADR 0013). |
| [`slice-M4-cluster-state-checks.md`](slice-M4-cluster-state-checks.md) | 2026-05-18 | Vier weitere Checks (`StorageClass`, `IngressClass`, `CertManager`, `ClusterResources`) schließen das MVP-Pflichtset (`LH-PRI-001`). Interface-Segregation pro Discovery-Domain (vier neue `port.*Discovery`-Interfaces); Adapter-Bündel über `NewClusterClients`. cert-manager-Severity bewusst `warning` (slice §9, Plichtmessage-Substring „external TLS termination" fixiert). Coverage strikt 90 % (von 76.9 % auf 95.8 % gehoben, `Makefile.THRESHOLD` von 0 auf 90 — slice-M4 zieht `ADR 0012 §2.5`-Wert vor; M6 erbt). Cluster-Smoke nutzt Minimal-Stubs unter `hack/cluster-smoke/` statt voller Upstream-Manifeste. **`LH-AK-006..009` real attestiert** im `cluster-smoke`-Workflow. |
