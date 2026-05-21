{{/*
Slice-M8 chart helpers.

Naming and label helpers used by every template in this chart.

  k-deskflight.name             — chart-derived short name (label value).
  k-deskflight.fullname         — release-prefixed resource name.
  k-deskflight.chart            — `<chart>-<version>` for the helm.sh/chart
                                  label.
  k-deskflight.labels           — Kubernetes recommended labels.
  k-deskflight.selectorLabels   — minimal selector labels (stable across
                                  upgrades; MUST NOT include version-bearing
                                  labels).
  k-deskflight.serviceAccountName — release-derived SA name, override-able
                                  via .Values.serviceAccount.name.
  k-deskflight.imageRef         — full image reference; empty
                                  .Values.image.tag falls back to
                                  .Chart.AppVersion.
*/}}

{{- define "k-deskflight.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "k-deskflight.fullname" -}}
{{- $name := .Chart.Name -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{- define "k-deskflight.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "k-deskflight.labels" -}}
helm.sh/chart: {{ include "k-deskflight.chart" . }}
{{ include "k-deskflight.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: k-deskflight
{{- end -}}

{{- define "k-deskflight.selectorLabels" -}}
app.kubernetes.io/name: {{ include "k-deskflight.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "k-deskflight.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "k-deskflight.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "k-deskflight.imageRef" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag -}}
{{- printf "%s:%s" .Values.image.repository $tag -}}
{{- end -}}
