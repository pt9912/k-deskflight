# In Progress

Aktive Roadmap und laufende Slice-Pläne.

Jeder Plan in diesem Verzeichnis MUSS enthalten:

- Lieferziel (was wird umgesetzt),
- Lastenheft-Kennungen (`LH-*`),
- Architekturartefakte (sobald `architecture.md` existiert),
- Abnahmekriterium (Verifikationspfad),
- Status (Pending / In Progress / Done).

Abgeschlossene Pläne wandern als Closure-Notiz nach `../done/`.

---

## Bestand

| Datei | Lieferziel | Status |
| ----- | ---------- | ------ |
| [`roadmap.md`](roadmap.md) | MVP v0.1 (`LH-MVP-002`, `LH-PRI-001`) in sieben Slices M1–M7 | In Progress — M1, M2, M3 geschlossen 2026-05-17; M4 aktiv seit 2026-05-18; M5–M7 weiterhin Pending. |
| [`slice-M4-cluster-state-checks.md`](slice-M4-cluster-state-checks.md) | Vier Cluster-State-Checks (StorageClass, IngressClass, cert-manager, ClusterResources); `LH-AK-006..009` (`LH-F-010..015`) | In Progress — eröffnet 2026-05-18. |

Slice-spezifische Pläne (`slice-MX-…md`) entstehen pro Slice beim
Aktivieren und tragen Detail-Lieferziele, Abnahmekriterien und
Test-Schritte. Die Roadmap selbst bleibt der Sammel-Schnitt.
