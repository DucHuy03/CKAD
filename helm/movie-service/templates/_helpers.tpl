{{- define "movie-service.fullname" -}}
{{- .Values.serviceName | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- define "movie-service.labels" -}}
app.kubernetes.io/name: {{ include "movie-service.fullname" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
{{- define "movie-service.selectorLabels" -}}
app.kubernetes.io/name: {{ include "movie-service.fullname" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
