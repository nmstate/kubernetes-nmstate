{{/*
Expand the name of the chart.
*/}}
{{- define "kubernetes-nmstate.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kubernetes-nmstate.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "kubernetes-nmstate.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kubernetes-nmstate.labels" -}}
helm.sh/chart: {{ include "kubernetes-nmstate.chart" . }}
{{ include "kubernetes-nmstate.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kubernetes-nmstate.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kubernetes-nmstate.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Operator namespace — from values.operator.namespace
*/}}
{{- define "kubernetes-nmstate.operatorNamespace" -}}
{{- .Values.operator.namespace | default "nmstate" }}
{{- end }}

{{/*
Handler namespace — falls back to operator namespace when empty
*/}}
{{- define "kubernetes-nmstate.handlerNamespace" -}}
{{- .Values.handler.namespace | default (include "kubernetes-nmstate.operatorNamespace" .) }}
{{- end }}

{{/*
Full operator image string: registry/repository:tag
*/}}
{{- define "kubernetes-nmstate.operatorImage" -}}
{{- printf "%s/%s:%s" .Values.operator.image.registry .Values.operator.image.repository (.Values.operator.image.tag | default .Chart.AppVersion) }}
{{- end }}

{{/*
Full handler image string: registry/repository:tag
*/}}
{{- define "kubernetes-nmstate.handlerImage" -}}
{{- printf "%s/%s:%s" .Values.handler.image.registry .Values.handler.image.repository (.Values.handler.image.tag | default .Chart.AppVersion) }}
{{- end }}
