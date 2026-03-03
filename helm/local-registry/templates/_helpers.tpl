{{- define "local-registry.name" -}}
local-registry
{{- end }}

{{- define "local-registry.fullname" -}}
{{ include "local-registry.name" . }}
{{- end }}

{{- define "registry.validateStorageClass" -}}
{{- if .Values.persistence.enabled }}
  {{- $sc := .Values.persistence.storageClass }}
  {{- if not $sc }}
    {{- fail "persistence.storageClass must be set" }}
  {{- end }}
  {{- if not (lookup "storage.k8s.io/v1" "StorageClass" "" $sc) }}
    {{- fail (printf "StorageClass '%s' not found" $sc) }}
  {{- end }}
{{- end }}
{{- end }}
