# Trigger — Whitelist statt Blacklist für controller-gen-Annotationen

**Trigger für:** Refactor des `helm-manifests-sync`-Normalisierungs-
Pfads von einer Blacklist (eine controller-gen-Annotation explizit
gedroppt) auf eine Whitelist (nur erwartete Annotationen behalten).
**Eröffnet:** 2026-05-21 (aus slice-M8 step-5-Review H-1 deferred)
**Bezug:** [Slice M8 — Helm-Chart](../in-progress/slice-M8-helm-chart.md),
[`scripts/helm-manifests-sync.sh`](../../../../scripts/helm-manifests-sync.sh),
[`spec/architecture.md` AR-007](../../../../spec/architecture.md)

---

## 1. Kontext

Slice M8 step 5 hat das Drift-Detektions-Gate
`make helm-manifests-sync` aktiviert. Die Normalisierungs-Pipeline
in `scripts/helm-manifests-sync.sh` droppt aktuell **eine** specifische
controller-gen-Annotation, damit Chart-Render und kustomize-Output
übereinstimmen:

```yq
(.. | select(tag == "!!map" and has("annotations")) | .annotations) |= del(."controller-gen.kubebuilder.io/version")
```

Das ist eine **Blacklist** — explizit benannte Annotationen werden
entfernt; alle anderen bleiben.

Step-5-Review-Befund H-1 (2026-05-21):

> controller-gen schreibt in seinen Outputs jedoch in bestimmten
> Konfigurationen zusätzlich `api-approved.kubernetes.io` (bei
> kubebuilder-Validation mit External-Approval-Bedarf) und kann je
> nach CRD-Markern weitere Annotationen durchreichen. Aktuell
> harmlos (unsere CRD hat keine), aber sobald `make manifests` mit
> erweiterten Markern läuft, kippt das Gate stumm in False-Positive.

Eine **Whitelist** — nur bekannte/erwartete Annotationen behalten,
alles andere droppen — wäre defensiv robuster. Step 5 hat den
Refactor explizit deferred mit der Begründung „eigener kleiner
Slice oder M16-Sub-Task".

---

## 2. Aktivierungs-Anlass

Aktivieren, wenn einer der folgenden Anlässe eintritt:

- **`make manifests` produziert eine neue Annotation,** die das
  Gate stumm umgeht (False-Positive: Drift existiert, wird aber
  nicht erkannt). Beobachtbar als „Anwender installiert via Helm,
  sieht eine Annotation auf der CRD, die im `kubectl apply -k`-Pfad
  nicht da ist" — Indiz: Operator-Logs zeigen Behavior-Drift
  zwischen den zwei Install-Modi.
- **controller-gen-Version-Hebung** (z. B. v0.21.0 → v0.22.x).
  Neue controller-gen-Major-Versionen können neue Default-
  Annotationen einführen.
- **CRD-Marker-Erweiterung** in der API: jedes neue
  `+kubebuilder:validation:`-Marker-Pattern kann Annotationen
  triggern.

---

## 3. Zu entscheiden bei Aktivierung

- **Welche Annotationen gehören in die Whitelist?** Mindestens:
  - `meta.helm.sh/release-name`, `meta.helm.sh/release-namespace` —
    werden bei `helm install` post-render gesetzt; im Smoke-Pfad
    aber durch helm direkt appliziert, nicht in der `helm
    template`-Output-Phase.
  - Anwender-spezifische Annotationen aus
    `values.yaml.serviceAccount.annotations` und ähnlichen Slots
    (Default leer).
  - Stable-Kubernetes-Annotationen, die NICHT controller-gen-
    generiert sind (z. B. `kubernetes.io/description`).
- **Implementierungs-Strategie:** ein yq-Pattern, das alle
  Annotationen mit Prefix `controller-gen.kubebuilder.io/` UND
  `api-approved.kubernetes.io` droppt — oder eine echte
  Allowlist über `keys`-Filter? Tradeoff: Prefix-Drop ist
  einfacher, Allowlist ist defensiver.
- **Test-Strategie:** ein synthetischer Probe-CRD-Output mit
  bekannten Annotationen, gegen die der Filter validiert wird.

---

## 4. Nächste Schritte

1. Bei Aktivierung wandert dieser Eintrag nach `next/` (mit
   Scope-Skizze) oder direkt nach `in-progress/` (mit Slice-Plan).
2. `scripts/helm-manifests-sync.sh` Normalisierungs-Block
   refaktoren: aus zwei expliziten `del(…)`-Calls eine Whitelist-
   Pattern-Pipeline machen.
3. Verifikation: heutiger Stand muss weiterhin drift-frei sein
   (Regression-Schutz); neue synthetische Annotation in einem
   Test-CRD muss das Gate triggern.
4. Slice-Closure aktualisiert `slice-M8-helm-chart.md §4 Step 5`
   („Deferred Future-Concern" streichen, Folge-Slice-Verweis
   hinzufügen).

---

## 5. Status

Offen. Niedrige Priorität — heute kein konkreter Drift, der das
Gate umgeht. Wahrscheinlichste Aktivierung mit der nächsten
`CONTROLLER_GEN_VERSION`-Hebung oder einer CRD-Erweiterung in
v0.3+ (`AR-OP-007`-Aktivierung, `v1alpha2`/`v1beta1`-Sprung). Bis
dahin als bekannte Gate-Härtungs-Lücke dokumentiert.
