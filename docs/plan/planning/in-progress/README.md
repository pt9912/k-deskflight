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
| [`roadmap.md`](roadmap.md) | MVP v0.1 (`LH-MVP-002`, `LH-PRI-001`) in sieben Slices M1–M7 | In Progress — M1 + M2 geschlossen 2026-05-17; M3 aktiviert 2026-05-17; M4–M7 weiterhin Pending. |
| [`slice-M3-kubernetes-version-check.md`](slice-M3-kubernetes-version-check.md) | Erster echter Check: KubernetesVersion-Vergleich, Reconciler-Phasen 1+3+4+5+6, Check-Interface (AR-012), Registry (AR-013), Aggregator (AR-014) | In Progress — eröffnet 2026-05-17 |

Slice-spezifische Pläne (`slice-MX-…md`) entstehen pro Slice beim
Aktivieren und tragen Detail-Lieferziele, Abnahmekriterien und
Test-Schritte. Die Roadmap selbst bleibt der Sammel-Schnitt.
