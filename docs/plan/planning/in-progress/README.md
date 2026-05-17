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
| [`roadmap.md`](roadmap.md) | MVP v0.1 (`LH-MVP-002`, `LH-PRI-001`) in sieben Slices M1–M7 | In Progress — M1 aktiviert am 2026-05-17; M2–M7 weiterhin Pending. |
| [`slice-M1-repo-skeleton.md`](slice-M1-repo-skeleton.md) | Go-Skeleton, Dockerfile, Makefile, `.golangci.yml`, Doc-Refs-/Coverage-Gate-Scripts, CI-Workflow-Skelett mit `gates` + `security-gates` parallel | In Progress — eröffnet 2026-05-17 |

Slice-spezifische Pläne (`slice-MX-…md`) entstehen pro Slice beim
Aktivieren und tragen Detail-Lieferziele, Abnahmekriterien und
Test-Schritte. Die Roadmap selbst bleibt der Sammel-Schnitt.
