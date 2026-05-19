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
| [`roadmap.md`](roadmap.md) | MVP v0.1 (`LH-MVP-002`, `LH-PRI-001`) in sieben Slices M1–M7 | In Progress — M1, M2, M3 geschlossen 2026-05-17; M4 und M5 geschlossen 2026-05-18; M6 geschlossen 2026-05-19; M7 weiterhin Pending. |

Slice-spezifische Pläne (`slice-MX-…md`) entstehen pro Slice beim
Aktivieren und tragen Detail-Lieferziele, Abnahmekriterien und
Test-Schritte. Die Roadmap selbst bleibt der Sammel-Schnitt.

---

## Slice-Konventionen (kumulativ aus M1–M6 Lessons)

Diese Konventionen sind **bindend für alle aktiven Slices** und
ergänzen die obigen Pflicht-Bestandteile. Sie destillieren wiederkehrende
Erkenntnisse aus den abgeschlossenen Slice-Closure-Notizen (siehe
jeweils `done/slice-MX-…md §10.4`).

### K-1 — Doku-Review obligatorisch nach jedem Doku-Step

**Konvention:** Jeder Slice-Schritt, der primär Doku liefert
(`docs(user)`-, `docs(plan)`-, `docs(arch)`-Commits), wird **vor**
dem nächsten Step vom `code-reviewer`-Subagent reviewt — nicht erst
am Slice-Ende.

**Hintergrund:** M6 §4 Step 4–6 (drei `docs(user)`-Commits) hatten
in einer einzigen Doku-Review-Runde **drei `[Hoch]`-Befunde**, die
Code-Reviews vorher nicht gefunden hatten — Falsch-Versprechungen,
die nur durch das tatsächliche Lesen des Doku-Texts auffielen
(`wget` im distroless-Image, Annotation-Bump-trigger-Versprechen,
falsche ADR-Quellenangabe). Siehe
[`done/slice-M6-metrics-tests-doku.md §10.4`](../done/slice-M6-metrics-tests-doku.md).

**Pragmatik:** Mehrere zusammenhängende Doku-Commits dürfen als
ein Doku-Review-Block gebündelt werden (M6 hat Step 4+5+6 in einer
Review-Runde adressiert); reine Doku-Querverweis-Fixes
(`make doc-refs`-Repath nach `git mv`) brauchen keinen separaten
Review.

### K-2 — `make`/Docker-Konvention im Subagent-Briefing wiederholen

**Konvention:** Bei jedem Aufruf eines Subagents (`code-reviewer`,
`Explore`, etc.) wird die `make`/Docker-Konvention dieses Repos
explizit im Briefing genannt — auch wenn sie schon in einer früheren
Iteration im selben Chat erwähnt wurde.

**Hintergrund:** Subagents starten ohne Repo-Memory. M6 hatte
einen Subagent-Run, der ursprünglich nicht über die `make`-Wrapper
verifiziert hat; spätere Runs mit explizitem Briefing („`make test`/
`make lint`/`make coverage-gate` — niemals `go test`/`golangci-lint`
direkt") haben die Konvention sauber eingehalten. Siehe
[`done/slice-M6-metrics-tests-doku.md §10.4`](../done/slice-M6-metrics-tests-doku.md)
letzte Lesson.

**Briefing-Muster:**

```
**Wichtige Konvention für diesen Repo:** Alle Verifikationen laufen
über `make` + Docker, nicht direkt über `go test`/`kubectl`/`bash`.
Relevante Targets: `make test`, `make lint`, `make coverage-gate`,
`make doc-refs`, `make generated-drift-check`, `make gates`,
`make cluster-smoke`. Falls du etwas verifizieren willst, nutze diese
Targets. **Niemals direkte Tool-Aufrufe.**
```
