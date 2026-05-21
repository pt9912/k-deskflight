# Offene Pläne und Trigger-Watch

Dieses Verzeichnis sammelt **trigger-getriebene Folgearbeit** und
**Vorabklärungen**, die noch nicht in eine aktive Roadmap aufgenommen
wurden.

Einträge wandern entweder:

- nach `next/`, sobald ein Scope skizziert ist, aber noch kein Slice aktiv,
- nach `in-progress/`, wenn sie direkt aktiviert werden, oder
- nach `../../archive/`, wenn sie bewusst verworfen werden.

---

## Bestand

| Datei | Trigger für | Kurzbeschreibung |
| ----- | ----------- | ---------------- |
| [`external-services-v03-activation.md`](external-services-v03-activation.md) | `LH-F-020`, `LH-F-021` | Folge-ADR zur v0.3+-Aktivierung der mit-Auth-Prüfungen aus `ADR 0010`. |
| [`chart-testing-activation.md`](chart-testing-activation.md) | `helm/chart-testing` (`ct`) als Quality-Gate | Erweiterung von `make gates` um `ct lint`/`ct install`; verschoben aus slice-M8 §8 Out-of-Scope. |
| [`helm-docs-automation.md`](helm-docs-automation.md) | `helm-docs`-Generierung der Chart-`README.md` | Drift-Prevention zwischen `values.yaml`-Kommentaren und Anwender-Doku; verschoben aus slice-M8 §8. |
| [`controller-gen-annotation-whitelist.md`](controller-gen-annotation-whitelist.md) | `helm-manifests-sync`-Hardening | Refactor des Drift-Gate-Normalisierers von Blacklist auf Whitelist; deferred aus slice-M8 step-5-Review H-1. |

Der frühere `changelog.md`-Trigger ist mit der M7-Closure
(2026-05-20) als
[`../done/changelog-trigger.md`](../done/changelog-trigger.md)
geschlossen.

Die kanonische Liste offener fachlicher Punkte lebt im Lastenheft
unter §22 (`LH-OP-*`). Einträge in diesem Verzeichnis sind die
Plan-/Arbeitssicht auf solche oder zusätzliche Trigger.
