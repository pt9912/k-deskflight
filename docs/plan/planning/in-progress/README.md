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
| [`roadmap.md`](roadmap.md) | MVP v0.1 (`LH-MVP-002`, `LH-PRI-001`) in sieben Slices M1–M7 | In Progress — M1, M2, M3 geschlossen 2026-05-17; M4 geschlossen 2026-05-18; M5 aktiv seit 2026-05-18; M6 und M7 weiterhin Pending. |
| [`slice-M5-rbac-self-check-robustness.md`](slice-M5-rbac-self-check-robustness.md) | SelfSubjectAccessReview pro Check + Panic-Härtung + Per-Check-Timeout + Secret-Filter-Konvention; `LH-AK-010/-012/-015/-016` (`LH-F-024`, `LH-NF-005`, `LH-SEC-001/002/005`) | In Progress — eröffnet 2026-05-18. |

Slice-spezifische Pläne (`slice-MX-…md`) entstehen pro Slice beim
Aktivieren und tragen Detail-Lieferziele, Abnahmekriterien und
Test-Schritte. Die Roadmap selbst bleibt der Sammel-Schnitt.
