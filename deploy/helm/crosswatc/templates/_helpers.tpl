{{- define "crosswatc.labels" -}}
app.kubernetes.io/part-of: crosswatc
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end }}

{{- define "crosswatc.image" -}}
{{ .root.Values.image.registry }}/crosswatc-{{ .name }}:{{ .root.Values.image.tag | default .root.Chart.AppVersion }}
{{- end }}

{{- define "crosswatc.commonEnv" -}}
- name: CROSSWATC_ENV
  value: {{ .Values.env | quote }}
- name: CROSSWATC_DATABASE_URL
  valueFrom:
    secretKeyRef:
      name: {{ .Values.database.existingSecret }}
      key: DATABASE_URL
{{- end }}

{{- define "crosswatc.securityContext" -}}
securityContext:
  runAsNonRoot: true
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop: ["ALL"]
{{- end }}

{{- define "crosswatc.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
    {{ default "crosswatc" .Values.serviceAccount.name }}
{{- else -}}
    {{ default "default" .Values.serviceAccount.name }}
{{- end -}}
{{- end }}

{{- define "crosswatc.otelEnv" -}}
{{- with .Values.observability.otlpEndpoint }}
- name: OTEL_EXPORTER_OTLP_ENDPOINT
  value: {{ . | quote }}
{{- end }}
{{- end }}
