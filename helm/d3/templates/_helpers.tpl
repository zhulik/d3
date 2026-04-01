{{/*
Expand the name of the chart.
*/}}
{{- define "d3.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "d3.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Redis component name
*/}}
{{- define "d3.redis.fullname" -}}
{{- printf "%s-redis" (include "d3.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Admin secret name (chart-managed)
*/}}
{{- define "d3.adminSecretName" -}}
{{- if .Values.admin.existingSecret }}
{{- .Values.admin.existingSecret }}
{{- else }}
{{- printf "%s-admin" (include "d3.fullname" .) }}
{{- end }}
{{- end }}

{{/*
Chart labels
*/}}
{{- define "d3.labels" -}}
helm.sh/chart: {{ include "d3.chart" . }}
{{ include "d3.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{- define "d3.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "d3.selectorLabels" -}}
app.kubernetes.io/name: {{ include "d3.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{- define "d3.redis.labels" -}}
{{ include "d3.labels" . }}
app.kubernetes.io/component: redis
{{- end }}

{{- define "d3.redis.selectorLabels" -}}
{{ include "d3.selectorLabels" . }}
app.kubernetes.io/component: redis
{{- end }}

{{- define "d3.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "d3.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}
