{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "kanister-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "kanister-operator.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*Create a default repository server controller name
*/}}
{{- define "repository-server-controller.name" -}}
{{- if .Values.repositoryServerController.container.name -}}
{{- .Values.repositoryServerController.container.name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- default "repository-server-controller"}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "kanister-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/* Helm required labels */}}
{{- define "kanister-operator.helmLabels" -}}
heritage: {{ .Release.Service }}
release: {{ .Release.Name }}
chart: {{ template "kanister-operator.chart" . }}
app: {{ template "kanister-operator.name" . }}
{{- end -}}

{{/*
Create the name of the service account to use
*/}}
{{- define "kanister-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default (include "kanister-operator.fullname" .) .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end -}}

{{/*
Figure out the target port of service, this depends
on the value of bpValidatingWebhook.enabled
*/}}
{{- define "kanister-operator.targetPort" -}}
{{- if .Values.bpValidatingWebhook.enabled -}}
    {{ 9443 }}
{{- else -}}
    {{ 8000 }}
{{- end -}}
{{- end -}}

{{/*
Figure out the port of service, this depends
on the value of bpValidatingWebhook.enabled
*/}}
{{- define "kanister-operator.servicePort" -}}
{{- if .Values.bpValidatingWebhook.enabled -}}
    {{ .Values.controller.service.port }}
{{- else -}}
    {{ .Values.controller.service.insecuredPort }}
{{- end -}}
{{- end -}}


{{/*
Figure out the target port of service, this depends
on the value of validatingWebhook.repositoryserver.enabled
*/}}
{{- define "reposerver-controller.targetPort" -}}
{{- if .Values.validatingWebhook.repositoryserver.enabled -}}
    {{ 8443 }}
{{- end -}}
{{- end -}}

{{/*
Figure out the port of service, this depends
on the value of validatingWebhook.repositoryserver.enabled
*/}}
{{- define "reposerver-controller.servicePort" -}}
{{- if .Values.validatingWebhook.repositoryserver.enabled -}}
    {{ .Values.repositoryServerController.service.port }}
{{- end -}}
{{- end -}}

{{/*
Define a custom kanister-tools image
*/}}
{{- define "kanister-tools.image" -}}
    {{- printf "%s:%s" (.Values.kanisterToolsImage.image) (.Values.kanisterToolsImage.tag) -}}
{{- end -}}

{{/*
Define a security Context at Pod level
*/}}
{{- define "podSecurityContext" -}}
    {{- if .Values.podSecurityContext }}
    securityContext: 
      {{ toYaml .Values.podSecurityContext  | nindent 6  }}
    {{- end }}
{{- end -}}

{{/*
Define a security Context at Container level
*/}}
{{- define "containerSecurityContext" -}}
    {{- if .Values.containerSecurityContext }}
    securityContext:
      {{ toYaml .Values.containerSecurityContext | nindent 6 }}
    {{- end }}
{{- end -}}


{{/*
Define a env variable for livenessProbe and readinessProbe
*/}}
{{- define "envVariableForProbes" -}}
    {{- if .Values.livenessProbe.enabled }}
    - name: LIVENESS_PATH
      value: {{ .Values.livenessProbe.httpGet.path| quote }}
    {{- end }}
    {{- if .Values.readinessProbe.enabled }}
    - name: READINESS_PATH
      value: {{ .Values.readinessProbe.httpGet.path| quote }}
    {{- end }}
    {{- if or .Values.readinessProbe.enabled .Values.livenessProbe.enabled }}
    - name: HEALTH_CHECK_PORT
      value: {{ .Values.healthCheckPort| quote }}
    {{- end }}
{{- end -}}

{{/*
Define a livenessprobe
*/}}
{{- define "livenessProbe" -}}
    {{- if .Values.livenessProbe.enabled }}
    livenessProbe:
      httpGet:
        path: {{ .Values.livenessProbe.httpGet.path }}
        port: {{ .Values.healthCheckPort }}
      initialDelaySeconds: {{ .Values.livenessProbe.initialDelaySeconds }}
      periodSeconds: {{ .Values.livenessProbe.periodSeconds }}
      timeoutSeconds: {{ .Values.livenessProbe.timeoutSeconds }}
      failureThreshold: {{ .Values.livenessProbe.failureThreshold }}
      successThreshold: {{ .Values.livenessProbe.successThreshold }}
    {{- end }}
{{- end -}}

{{/*
Define a readinessprobe
*/}}
{{- define "readinessProbe" -}}
    {{- if .Values.readinessProbe.enabled }}
    readinessProbe:
      httpGet:
        path: {{ .Values.readinessProbe.httpGet.path }}
        port: {{ .Values.healthCheckPort }}
      initialDelaySeconds: {{ .Values.readinessProbe.initialDelaySeconds }}
      periodSeconds: {{ .Values.readinessProbe.periodSeconds }}
      timeoutSeconds: {{ .Values.readinessProbe.timeoutSeconds }}
      failureThreshold: {{ .Values.readinessProbe.failureThreshold }}
      successThreshold: {{ .Values.readinessProbe.successThreshold }}
    {{- end }}
{{- end -}}

{{/*
Define a env variable for secureDefaultsForJobPods.
*/}}
{{- define "envVariableForSecureDefaults" -}}
    {{- if .Values.secureDefaultsForJobPods }}
    - name: SECURE_DEFAULTS_FOR_JOB_PODS
      value: {{ .Values.secureDefaultsForJobPods | quote }}
    {{- end }}
{{- end -}}