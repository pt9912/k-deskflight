{{/*
Slice-M8 chart helpers.

Design-Prinzip: Resource-Namen sind **statisch** und matchen die
kanonische Quelle in deploy/manifests/ 1:1 (slice-M8 §2.2). Kein
release-präfix-basiertes name/fullname-Helper-Schema, weil:

  - der Operator-Chart pro Cluster nur einmal installiert wird,
  - eine zweite Helm-Release derselben Chart im selben Cluster keine
    sinnvolle Semantik hat (RBAC-Objekte, CRD, Cluster-Resources sind
    cluster-singletons),
  - das `helm-manifests-sync`-Gate (slice-M8 §2.5) Drift-Detektion
    zwischen Chart-Templates und `deploy/manifests/` ohne Spezialregeln
    durchführen kann.

  k-deskflight.chart           — `<chart>-<version>` for the helm.sh/chart
                                 label.
  k-deskflight.selectorLabels  — minimal selector labels (stabil über
                                 Upgrades; matched deploy/manifests/
                                 selector-Sets 1:1).
  k-deskflight.labels          — Kubernetes recommended labels mit
                                 Helm-Meta (chart, version, managed-by)
                                 zusätzlich.
  k-deskflight.serviceAccountName — Default-Override via
                                 .Values.serviceAccount.name; fällt auf
                                 statischen Namen `k-deskflight-operator`
                                 zurück.
  k-deskflight.imageRef        — full image reference; leerer
                                 .Values.image.tag fällt auf
                                 .Chart.AppVersion zurück.
*/}}

{{- define "k-deskflight.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "k-deskflight.selectorLabels" -}}
app.kubernetes.io/name: k-deskflight
app.kubernetes.io/component: operator
{{- end -}}

{{- define "k-deskflight.labels" -}}
{{ include "k-deskflight.selectorLabels" . }}
app.kubernetes.io/part-of: k-deskflight
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ include "k-deskflight.chart" . }}
{{- end -}}

{{- define "k-deskflight.serviceAccountName" -}}
{{- default "k-deskflight-operator" .Values.serviceAccount.name -}}
{{- end -}}

{{/*
imageRef: Helm-Konvention für appVersion ist "0.1.0" (ohne v-Präfix);
GHCR-/Git-Tag-Konvention im Projekt ist "v0.1.0" (mit v). Diese
Helper-Funktion brückt die beiden, indem sie das v-Präfix für den
Default-Fall hinzufügt. Anwender, die einen non-default Tag setzen
(z. B. `image.tag=sha-…`), kontrollieren das Präfix selbst.
*/}}
{{- define "k-deskflight.imageRef" -}}
{{- $tag := default (printf "v%s" .Chart.AppVersion) .Values.image.tag -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}
