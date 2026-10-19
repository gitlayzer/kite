{{/*
Expand the name of the chart.
*/}}
{{- define "kite.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kite.fullname" -}}
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
{{- define "kite.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kite.labels" -}}
helm.sh/chart: {{ include "kite.chart" . }}
{{ include "kite.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kite.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kite.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "kite.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kite.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Name of the PersistentVolumeClaim backing the sqlite database file.
Honors db.sqlite.persistence.pvc.existingClaim when provided; otherwise
derives a claim name from the release fullname.
*/}}
{{- define "kite.sqliteClaimName" -}}
{{- $pvc := default (dict) .Values.db.sqlite.persistence.pvc -}}
{{- $existing := (default "" $pvc.existingClaim) | trim -}}
{{- if $existing -}}
{{- $existing -}}
{{- else -}}
{{- printf "%s-storage" (include "kite.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end }}

{{- define "kite.boolStringIsTrue" -}}
{{- $value := printf "%v" . | lower -}}
{{- if or (eq $value "true") (eq $value "yes") (eq $value "y") (eq $value "1") (eq $value "on") -}}true{{- else -}}false{{- end -}}
{{- end }}

{{- define "kite.disableHttps" -}}
{{- include "kite.boolStringIsTrue" (default false .Values.disableHttps) -}}
{{- end }}

{{- define "kite.externalScheme" -}}
{{- if eq (include "kite.disableHttps" .) "true" -}}http{{- else -}}https{{- end -}}
{{- end }}

{{- define "kite.externalPort" -}}
{{- if eq (include "kite.disableHttps" .) "true" -}}
{{- default "80" .Values.httpPort -}}
{{- else -}}
{{- default "443" .Values.cloudPort -}}
{{- end -}}
{{- end }}

{{- define "kite.externalPortSuffix" -}}
{{- $scheme := include "kite.externalScheme" . -}}
{{- $port := printf "%v" (include "kite.externalPort" .) -}}
{{- if and (eq $scheme "http") (eq $port "80") -}}
{{- else if and (eq $scheme "https") (eq $port "443") -}}
{{- else -}}
{{- printf ":%s" $port -}}
{{- end -}}
{{- end }}

{{- define "kite.externalHost" -}}
{{- $cloudDomain := required "cloudDomain is required when external access is enabled" .Values.cloudDomain -}}
{{- printf "kite.%s" $cloudDomain -}}
{{- end }}

{{- define "kite.externalURL" -}}
{{- printf "%s://%s%s" (include "kite.externalScheme" .) (include "kite.externalHost" .) (include "kite.externalPortSuffix" .) -}}
{{- end }}

{{- define "kite.ingressTlsEnabled" -}}
{{- if eq (include "kite.disableHttps" .) "true" -}}false{{- else -}}true{{- end -}}
{{- end }}


{{- define "kite.secret" -}}
{{- if .Values.secret.existingSecret }}
{{- .Values.secret.existingSecret }}
{{- else }}
{{- printf "%s-secret" (include "kite.fullname" .) }}
{{- end }}
{{- end }}
