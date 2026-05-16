# ADR 0011 — Governance und Beitragskonventionen

**Status:** Accepted
**Datum:** 2026-05-16
**Bezug:** [Lastenheft](../../../spec/lastenheft.md),
[ADR 0001](0001-dokumentations-und-planungsstruktur.md),
[ADR 0002](0002-adr-lifecycle.md),
[ADR 0004](0004-projektname.md)

---

## 1. Kontext

`LH-OP-010` verlangt eine Governance-Festlegung für
Open-Source-Beiträge. Direkt betroffene Lastenheftaussagen:

- `LH-PK-005`: Open-Source-Veröffentlichung; das Projekt soll „für
  externe Mitwirkende verständlich, testbar und nachvollziehbar"
  aufgebaut sein.
- `LH-STK-006`: Stakeholder-Gruppe „Open-Source-Mitwirkende".
- `LH-AK-014`: MVP-Pflicht — MIT-Lizenzdatei (✓ vorhanden), README
  (fehlt), „grundlegende Beitragsinformationen" (fehlt).
- `LH-NF-021`: Sprachpolitik — Issues/PRs/Commits/CONTRIBUTING/
  Code of Conduct Englisch; README/Lastenheft/Pflichtenheft Deutsch.

Diese ADR bindet die operative Politik (Code of Conduct, Commit-
Mechanik, Branch-Strategie, Security-Disclosure, Maintainer-Modell)
und nimmt damit `LH-AK-014`-Pflicht-Dokumente konzeptuell vorweg —
die konkreten Textdateien (README, CONTRIBUTING, CODE_OF_CONDUCT,
SECURITY) entstehen mit dem ersten Code-Commit (`LH-VM-004`).

---

## 2. Entscheidung

### 2.1 Code of Conduct

**Contributor Covenant v2.1** wird als Code of Conduct übernommen.
Quelle: `https://www.contributor-covenant.org/version/2/1/code_of_conduct/`.
Die Datei `CODE_OF_CONDUCT.md` enthält den unveränderten Text der
v2.1-Version (Englisch, `LH-NF-021`-konform) plus eine
Kontaktangabe für Verstoß-Meldungen (siehe §2.6 Security-Pfad als
Anker).

### 2.2 Commit-Konvention

Commits folgen **Conventional Commits** im Format:

```text
type(scope): subject

[optional body]

[optional footer(s)]
```

Erlaubte `type`-Werte (zunächst): `feat`, `fix`, `docs`, `chore`,
`refactor`, `test`, `build`, `ci`, `perf`, `style`. Der `scope`
folgt der Repository-Struktur und ist optional. Bisherige
Commit-Messages dieses Repositories verwenden diese Form bereits
(`docs(plan):`, `docs(spec):`); ADR 0011 macht sie verbindlich.

### 2.3 Commit-Sprache

Strikt Englisch ab Akzeptanz dieser ADR. `LH-NF-021` wird ohne
Pre-Code-Ausnahme angewendet — auch ADR-Closure-Commits in der
Spec-Phase sind ab jetzt Englisch.

Bisherige Deutsche Commits (vor ADR-0011-Acceptance) bleiben
unverändert; sie sind Teil der Historie und werden nicht
nachträglich umgeschrieben. Diese ADR-Acceptance ist die
Bruchkante.

### 2.4 Developer Certificate of Origin (DCO)

Beiträge unterliegen dem **Developer Certificate of Origin** (DCO,
`https://developercertificate.org/`). Jeder Commit trägt am Ende
der Message eine Sign-off-Zeile:

```text
Signed-off-by: Vorname Nachname <email@example.org>
```

Sign-off erfolgt via `git commit -s` (Email-Adresse aus `git config
user.email`, Klarname aus `git config user.name`). Kein separates
CLA, kein digitales Signing-Tool — DCO ist Standard-Git-Mechanik.

**Geltungsbereich:** ab Akzeptanz dieser ADR. Frühere Commits sind
*grandfathered* und werden nicht nachträglich signiert; alle
früheren Commits stammen vom Projektowner selbst und sind damit
implizit unter MIT lizenziert.

**Durchsetzung:** sobald CI eingerichtet ist (`LH-VM-005`), prüft
ein DCO-Check (z. B. GitHub-Apps wie `probot/dco`) jede neue PR
auf Sign-off-Zeilen pro Commit. Bis dahin manuell beim Merge
kontrollieren.

### 2.5 Branch- und Release-Strategie

- **`main`** ist die einzige langlaufende Branch. Kein `develop`,
  kein Git-Flow.
- Feature-Arbeit läuft auf kurzlebigen Branches (`feat/...`,
  `fix/...`, …) und wird über Pull Requests in `main` gemergt.
- Releases werden als annotierte Git-Tags `vX.Y.Z` (SemVer 2.0.0)
  gesetzt; pre-Release-Versionen folgen SemVer-Pre-Release-Form
  (`v0.1.0-rc.1` etc.).
- Long-term-support-Branches für ältere Versionen sind nicht
  vorgesehen.

### 2.6 Security-Disclosure-Pfad

Sicherheitslücken werden über den **GitHub Security Advisories**-Pfad
gemeldet (Repository → Security → Advisories → Report a vulnerability).
Keine separate Email-Adresse.

**Hosting-Annahme:** Diese Wahl setzt voraus, dass das Repository
auf GitHub gehostet wird. Ein Wechsel des Hosters (Codeberg,
GitLab, andere) würde diese Aussage hinfällig machen und benötigt
eine ablösende ADR mit Hoster-spezifischem Disclosure-Mechanismus.

**Disclosure-Frist:** 90 Tage Coordinated-Disclosure ab Eingang
einer Meldung. Vor Ablauf erfolgt entweder ein Fix-Release oder
eine ausdrückliche Verlängerungsvereinbarung mit dem Meldenden.

**Verstoß-Meldungen zum Code of Conduct** nutzen denselben
GHSA-Pfad (private Channel zum Maintainer). Begründung: solange
keine eigene Mailing-Adresse existiert, ist der GHSA-Channel der
einzige etablierte private Kommunikationsweg.

### 2.7 Maintainer-Modell

**Initial:** Single-Maintainer. Projektowner gemäß `LH-PROD-001`
(Dietmar Burkard) ist alleiniger Maintainer. Maintainer-Aufgaben
umfassen: PR-Reviews, Merge-Rechte, Release-Tags, Security-
Disclosure-Triage.

**Erweiterung:** Aufnahme weiterer Maintainer ist offen und
erfolgt entweder über eine spätere ADR (für strukturelle
Festlegungen, z. B. Mehrheits-Regeln) oder über ein
`GOVERNANCE.md`-Update (für Personalwechsel). Solange nur ein
Maintainer existiert, ist `GOVERNANCE.md` als eigenes Dokument
nicht erforderlich.

---

## 3. Konsequenzen

- `LH-AK-014` (Open-Source-Veröffentlichung möglich) wird zum
  MVP-Release (v0.1, `LH-REL-001`) durch folgende Dateien im
  Repository-Root erfüllt:
  - `README.md` — Englisch, Default-Entry-Point auf GitHub.
  - `README.de.md` — Deutsche Übersetzung des `README.md`,
    1:1-symmetrisch in Struktur und Detailgrad. Bedient die
    Zielgruppe `LH-PK-004`.
  - `CONTRIBUTING.md` — Englisch, beschreibt DCO, Conventional
    Commits, Branch-Workflow.
  - `CODE_OF_CONDUCT.md` — Contributor Covenant v2.1, Englisch.
  - `SECURITY.md` — Englisch, GHSA-Pfad, 90-Tage-Frist.
  - `LICENSE` — MIT (✓ bereits vorhanden).
- **README-Sprachpolitik:** `LH-NF-021` wird im selben Commit
  geschärft. `README.md` ist Englisch (Default-Entry-Point für
  GitHub-Discoverability und internationale Mitwirkende),
  `README.de.md` enthält die deutsche Übersetzung für die
  Zielgruppe `LH-PK-004`. Pflege beider Versionen ist
  Beitragenden-Pflicht: inhaltliche Änderungen landen in beiden
  Files im selben Pull Request.
- `LH-NF-021` wird ab Akzeptanz dieser ADR strikt angewendet — auch
  Spec-/Plan-Commits sind ab jetzt Englisch. Die Bruchkante ist
  diese ADR (genauer: der ADR-0011-Closure-Commit). Bisherige
  Deutsche Commits bleiben unverändert.
- `LH-OP-010` wird im Lastenheft als geschlossen mit dieser ADR
  markiert (Formelhilfe aus `ADR 0002`).
- **Hosting-Annahme:** Die GHSA-Wahl bindet das Repository an
  GitHub. Eine spätere Wahl-Änderung erfordert eine ablösende ADR
  (Stichwort: Sovereignity, Codeberg/GitLab-Alternativen für
  behördennahe Zielgruppen — `LH-PK-004`).
- **DCO-CI-Bot** ist Folgearbeit für `LH-VM-005` (Integrationstest-
  Phase) und kein blockierender MVP-Bestandteil.
- **`Co-Authored-By`-Footer** in bisherigen Commit-Messages bleiben
  zulässig; sie ersetzen kein DCO-Sign-off, sondern dokumentieren
  Mit-Urheberschaft. Ab ADR-0011-Acceptance kommt der DCO-Sign-off
  zusätzlich hinzu.

---

## 4. Nicht Gegenstand dieser ADR

- **Konkrete Textinhalte** der Pflicht-Dateien (`README.md`,
  `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`) —
  entstehen mit dem ersten Code-Commit (`LH-VM-004`) bzw.
  spätestens zum MVP-Release (`LH-AK-014`).
- **Konkreter Repository-Pfad** und **GitHub-Organisation**
  (`github.com/<owner>/k-deskflight`) — operative Folgearbeit,
  nicht ADR-pflichtig (siehe `ADR 0004 §4`).
- **PR- und Issue-Templates** — operative Folgearbeit, kann via
  `.github/ISSUE_TEMPLATE/` und `.github/pull_request_template.md`
  jederzeit ergänzt werden.
- **Communication-Channels** (Mailing-Liste, Matrix-Room, Slack)
  — solange kein Bedarf, kein Channel; Issue-Tracker reicht.
- **Maintainer-Aufnahme-Prozess** (Wahl, Konsens, Probezeit) —
  spätere ADR, sobald mehr als ein Maintainer Bedarf entsteht.
- **CHANGELOG-Format und Generierungs-Tooling** — eigener
  Trigger-Eintrag `docs/plan/planning/open/changelog.md`.
- **Release-Approval-Mechanismus** (Pendant zu m-trace's
  `MTRACE_RELEASE_APPROVED=1`) — Pflichtenheft (`LH-VM-002`).
- **Versionierungs-Politik für die CRD** (`v1alpha1` → `v1alpha2`
  etc.) — separat von Repository-Versionen, siehe `ADR 0006 §4`.
