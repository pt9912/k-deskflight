# ADR 0009 — Kubernetes-Versions-Support und Profile-Mindestversionen

**Status:** Accepted
**Datum:** 2026-05-16
**Bezug:** [Lastenheft](../../../spec/lastenheft.md),
[ADR 0001](0001-dokumentations-und-planungsstruktur.md),
[ADR 0002](0002-adr-lifecycle.md),
[ADR 0006](0006-api-gruppe-und-crd-scope.md)

---

## 1. Kontext

Zwei offene Lastenheftpunkte verlangen eine versionsbezogene
Entscheidung:

- `LH-OP-004` — „Unterstützte Kubernetes-Versionen definieren":
  Welche K8s-Versionen unterstützt der Operator beim Betrieb?
- `LH-OP-001` — „Exakte Mindestversionen für OpenDesk-Profile
  festlegen": Welche K8s-Mindestversion verlangt jedes Profile
  (`LH-PROF-002` `evaluation`, `LH-PROF-003` `production`)?

Drei Versions-Dimensionen sind zu unterscheiden:

| Dimension | Bedeutung | Hoheit |
| --------- | --------- | ------ |
| Operator-Build-Version | Version der `client-go`/`controller-runtime`-Bibliotheken | Pflichtenheft (`LH-VM-002`) |
| Operator-Run-Support-Matrix | K8s-Versionen, auf denen der Operator läuft und getestet ist | `LH-OP-004` — §2.1 dieser ADR |
| Profile-Mindestversion | K8s-Mindestversion, die ein OpenDesk-Profile vom geprüften Cluster verlangt (`LH-F-008`) | `LH-OP-001` — §2.2 dieser ADR |

Externe Anker zum Entscheidungszeitpunkt (2026-05-16):

- **Kubernetes-Upstream-Support-Politik**
  ([kubernetes.io/releases](https://kubernetes.io/releases)): drei
  Minor-Versionen parallel mit Patch-Support, ca. 1 Jahr
  Patch-Support pro Minor, neuer Minor alle 3–4 Monate. Aktiv
  supportet sind heute 1.34, 1.35, 1.36 (1.34 EOL 27.10.2026,
  1.35 EOL 28.02.2027, 1.36 EOL 28.06.2027).
- **OpenDesk-Cluster-Anforderung**
  ([docs.opendesk.eu/operations/requirements](https://docs.opendesk.eu/operations/requirements)):
  Kubernetes-Cluster `≥ v1.24`, CNCF-Certified-Distribution.
  Getestet wird OpenDesk-Deployment gegen `kubespray`-basierte
  Cluster; OpenShift ist nicht getestet. Quelle bestätigt im
  GitLab-Repository `docs/requirements.md`.

Spannung: Der OpenDesk-Doku-Floor (1.24) ist niedriger als der
Operator-Run-Support-Floor (heute 1.34). Diese ADR löst die
Spannung über das Operator-Floor-Argument (siehe §2.3).

---

## 2. Entscheidung

### 2.1 Operator-Run-Support-Matrix (LH-OP-004)

Der Operator unterstützt jeweils die **drei aktuellsten
Kubernetes-Minor-Versionen mit aktivem Patch-Support** (rolling
window), analog der Kubernetes-Upstream-Politik.

**Momentaufnahme zum ADR-Datum 2026-05-16:**

| K8s-Minor | Status | Patch-Support bis |
| --------- | ------ | ----------------- |
| 1.34 | Active (Floor) | 27.10.2026 |
| 1.35 | Active | 28.02.2027 |
| 1.36 | Latest (Ceiling) | 28.06.2027 |

Diese Liste ist eine **illustrierende Momentaufnahme**. Verbindlich
ist die *Politik* („drei aktuelle Minor mit Patch-Support"), nicht
die konkrete Zahlenliste. Der jeweils aktuelle Support-Stand wird
pro Operator-Release dokumentiert (Release-Notes-Pflicht).

### 2.2 Profile-Mindestversionen (LH-OP-001)

`evaluation` und `production` verwenden denselben K8s-Mindestversion-
Default: **die aktuelle Operator-Floor-Version** (heute 1.34).

```yaml
# Profile-Defaults für spec.checks.kubernetesVersion.min
evaluation: "1.34"   # = Operator-Floor zum ADR-Datum
production: "1.34"   # = Operator-Floor zum ADR-Datum
```

Profile-Differenzierung erfolgt **nicht** über die K8s-Version,
sondern über Ressourcen, Storage, TLS, Ingress und externe
Dienste (siehe `LH-PROF-003`). Die K8s-Version ist eine
Plattform-Anforderung, keine Profilcharakteristik.

Der Anwender kann pro CR via `spec.checks.kubernetesVersion.min`
einen anderen Wert setzen (`LH-F-008` — „konfigurierte
Mindestversion"). Der Profile-Default ist nur die Vorbelegung,
kein hartes Limit.

### 2.3 Verhältnis zum OpenDesk-Doku-Floor (1.24)

Die OpenDesk-Dokumentation nennt `K8s ≥ v1.24` als Minimum für
CNCF-Certified-Distributions. Diese Aussage bleibt der
**fachliche Hard-Floor**: unterhalb 1.24 ist OpenDesk laut Doku
nicht supportet.

Der Operator setzt seinen Profile-Default trotzdem auf den
**Operator-Floor** (heute 1.34), nicht auf den OpenDesk-Floor
(1.24). Begründung:

- Der Operator selbst läuft nur auf der eigenen Support-Matrix
  (§2.1). Ein Cluster auf 1.24 kann den Operator gar nicht
  betreiben — ein Profile-Default auf 1.24 wäre operativ wirkungslos
  (`max(Operator-Floor, OpenDesk-Floor) = Operator-Floor`).
- 1.24 ist seit Jahren ohne Upstream-Patch-Support (EOL ca.
  Mitte 2023). Ein produktives OpenDesk-Deployment auf 1.24 wäre
  trotz formaler Doku-Konformität sicherheitsbedenklich.
- Die OpenDesk-Doku-Aussage „v1.24" entstand historisch zu einem
  Zeitpunkt mit anderer K8s-Patch-Lage; eine Aktualisierung der
  Doku ist nicht in unserer Hoheit.

Anwender, die strikt auf den OpenDesk-Doku-Floor prüfen wollen,
können per CR `spec.checks.kubernetesVersion.min: "1.24"` setzen.
Die Prüfung gibt auf einem 1.34-Cluster trivialerweise „passed"
zurück.

### 2.4 Aktualisierungsmodell

Bei jedem neuen K8s-Minor-Release (Upstream) verschiebt sich die
Operator-Floor-Version. Mit der `LH-OP-004`-Politik führt das zu
einer planmäßigen Support-Matrix-Hebung im nächsten
Operator-Release. Der Floor-Wert in den Profile-Defaults
(`evaluation`, `production`) bewegt sich im gleichen Rhythmus
mit.

Die ADR wird dadurch **nicht abgelöst** — sie bindet das *Prinzip*,
nicht die konkrete Zahl. Die jeweils aktuelle Matrix lebt im
Operator-Release-Note bzw. in der pro-Release-aktualisierten
Versionsmatrix-Datei (Konkretisierung im Pflichtenheft).

---

## 3. Konsequenzen

- `LH-PROD-003a` (MVP-Beispiel) wird im selben Commit von
  `kubernetesVersion.min: "1.27"` auf `"1.34"` aktualisiert.
- `LH-PROF-002` und `LH-PROF-003` erhalten je eine Notiz zur
  K8s-Mindestversion-Default-Vorbelegung mit ADR-0009-Verweis.
- `LH-OP-001` und `LH-OP-004` werden im Lastenheft als geschlossen
  mit dieser ADR markiert (Formelhilfe aus `ADR 0002`).
- Die Operator-Floor-Version wird zu einem **laufend gepflegten
  Wert**. Bei jedem K8s-Minor-Release ist die Floor-Version pro
  Operator-Release zu aktualisieren — Folgearbeit, kein neuer
  ADR-Bedarf, weil das Prinzip in dieser ADR steht.
- `LH-F-008` bleibt unverändert (konfigurierte Mindestversion); die
  Profile-Defaults aus §2.2 sind nur Vorbelegungen, nicht die
  einzige zulässige Mindestversion.
- `LH-RISK-006` (Providerabhängigkeit) bleibt unberührt: die
  Support-Matrix gilt für **alle** CNCF-Certified-Distributions
  ohne Anbieterbindung — der OpenDesk-Doku-Hinweis auf
  „getestet gegen kubespray, nicht gegen OpenShift" ist ein
  OpenDesk-internes Test-Detail, kein Operator-Constraint.

---

## 4. Nicht Gegenstand dieser ADR

- **Konkretes CR-Schema** für `spec.checks.kubernetesVersion` —
  Pflichtenheft (`LH-VM-002`).
- **Operator-Build-Version** (`client-go`/`controller-runtime`-
  Bibliotheksversion) — Pflichtenheft.
- **Spätere Profile** aus `LH-PROF-001` (`k3s`, `scs`, `airgapped`,
  `custom` als Modus aus `LH-PROF-004`) — entstehen mit ihrer
  jeweiligen Einführungs-ADR; ggf. mit eigenen Versions-Defaults.
- **CRD-Conversion-Webhooks** und **Versions-Upgrade-Pfad** der
  CRD selbst (`v1alpha1` → `v1alpha2` etc.) — spätere ADR (siehe
  `ADR 0006 §4`).
- **OpenDesk-Doku-Verifikation**: diese ADR zitiert den OpenDesk-
  Stand vom Entscheidungsdatum. Eine spätere OpenDesk-Doku-Hebung
  (z. B. auf 1.30+) wird beim nächsten ADR-Lifecycle-Anlass
  berücksichtigt.
- **OpenShift-spezifische Anforderungen**: OpenDesk testet nicht
  gegen OpenShift; der Operator macht keine eigene OpenShift-
  Unterstützungs-Aussage. `LH-NF-018` (Plattformneutralität)
  bleibt der allgemeine Anker; OpenShift-Profile sind spätere
  optionale Folgearbeit (`LH-PROF-001`).
