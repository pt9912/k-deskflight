# Trigger — CHANGELOG.md anlegen

**Trigger für:** Erste getaggte Release-Version (`LH-REL-001`)
**Eröffnet:** 2026-05-16
**Bezug:** [Lastenheft `LH-REL-001`..`LH-REL-004`](../../../../spec/lastenheft.md),
[ADR 0001](../../adr/0001-dokumentations-und-planungsstruktur.md)

---

## 1. Kontext

Das Repository hat aktuell keine `CHANGELOG.md`. Solange ausschließlich
Doku- und Plan-Artefakte entstehen, bilden Commit-Log und ADR-Reihe
den vollständigen Audit-Trail. Mit dem ersten getaggten Release
(`LH-REL-001`, MVP) brauchen Anwender allerdings eine versionsweise
Übersicht über Änderungen, getrennt von einzelnen Commits.

Das Schwesterprojekt `/Development/m-trace` führt eine `CHANGELOG.md`
nach einem etablierten Pattern und ist eine direkte Adaptionsvorlage.

---

## 2. Zu entscheiden

- **Format**: Keep-a-Changelog (m-trace nutzt es) vs.
  Conventional-Changelog-Generator vs. eigene Konvention.
- **Sprache**: Deutsch (analog Lastenheft-Zielgruppe) oder Englisch
  (`LH-NF-021` für Issues/PRs/Commits)?
- **Pflege**: pro Release manuell vs. Tooling (`release-please`,
  `git-cliff`, …)?
- **Granularität**: pro Release-Tag oder pro Slice?

---

## 3. Nächste Schritte

1. Mit Aktivierung des ersten Slice in `in-progress/` (Roadmap zum
   MVP-Scope `LH-MVP-002`) entscheiden, ob die `CHANGELOG.md` vor
   dem ersten `v0.1.0`-Tag aufgebaut wird oder erst mit dem Tag-Commit
   entsteht.
2. ADR für Format-/Pflegekonvention schreiben, sobald die Entscheidung
   steht — oder die Konvention inline in der Roadmap festhalten, falls
   die Wahl als Routine-Setup ohne langfristige Bindung tragfähig ist.
3. Erste `CHANGELOG.md` mit dem `v0.1.0`-Tag committen.

---

## 4. Status

Offen. Wandert nach `next/`, sobald die erste Roadmap unter
`in-progress/` steht. Bis dahin sind Commit-Log und ADRs ausreichende
Auditspuren.
