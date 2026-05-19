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
| [conditions.md](conditions.md) | Conditions-Katalog: Reasons + Severity pro ConditionType, plus Per-Check-Runner-Reasons aus Slice M5. |
| [troubleshooting.md](troubleshooting.md) | Acht typische Fehlerbilder + ein v0.2-Vorgriff: Symptom → Diagnose-Kommando → Lösungs-Schritt. |

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
