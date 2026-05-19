# Anwender- und Betreiberdokumentation

Dieses Verzeichnis enthält anwender- und betreibernahe Dokumentation
für den **k-deskflight**-Operator. Adressaten sind Cluster-Betreiber
und OpenDesk-Installateure, die den Operator deployen, eine
`OpenDeskPreflightCheck`-Ressource konfigurieren und die Ergebnisse
in einem Cluster lesen wollen.

Diese Dokumente sind keine Architekturartefakte und keine
Spezifikationen. Normative Inhalte gehören in [`spec/`](../../spec/),
Entscheidungen in [`docs/plan/adr/`](../plan/adr/).

---

## Inhaltsverzeichnis

| Dokument | Inhalt |
| -------- | ------ |
| [installation.md](installation.md) | Installation via raw manifests, Namespace-Override, Image-Pin, Prometheus-Scrape-Binding-Pattern. |
| [cr-examples.md](cr-examples.md) | Zwei vollständige CR-Beispiele (`evaluation` und `production`) mit Profile-Default-Auswertung und Wiederholintervall. |

Folgende Dokumente entstehen ebenfalls in der M6-Slice und werden
hier eingetragen, sobald sie committed sind:

- `conditions.md` — Conditions-Katalog (Reason/Severity pro
  ConditionType, geplant für M6 §4 Step 5).
- `troubleshooting.md` — typische Fehlerbilder + Diagnose-Kommandos
  (geplant für M6 §4 Step 6).

---

## Hinweise zur Metrik-Namens-Konvention

Der Operator exponiert in v0.1 ausschließlich die
controller-runtime-Framework-Defaults
(`controller_runtime_*`, `rest_client_*`, `workqueue_*`, plus
Prometheus-Client-Go-Standard wie `go_*` und `process_*`) auf
`/metrics`. Eigene Domänen-Metriken sind **nicht Bestandteil des
MVP** ([ADR 0007 §2.1](../plan/adr/0007-prometheus-metrik-scope-im-mvp.md)).

Sie kommen mit v0.2 unter dem reservierten Prefix `kdeskflight_*`
([ADR 0007 §2.3](../plan/adr/0007-prometheus-metrik-scope-im-mvp.md)).
Wer eigene Alert-Regeln gegen den Operator schreibt, kann
`kdeskflight_*` heute schon als zukünftiges Filter-Prefix reservieren
— in v0.1 matched es schlicht nichts.
