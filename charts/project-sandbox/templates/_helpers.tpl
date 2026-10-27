{{- define "sandbox.project" -}}
{{- required "project is required" .Values.project -}}
{{- end -}}
{{- define "sandbox.namespace" -}}
{{- default (printf "sbx-%s" (include "sandbox.project" .)) .Values.namespace | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- define "sandbox.labels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/name: project-sandbox
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
warbleoss.org/sandbox: "true"
warbleoss.org/project: {{ include "sandbox.project" . | quote }}
{{- end -}}
