{{/*
Namespace
*/}}
{{- define "atlhyper.namespace" -}}
{{- .Values.global.namespace | default "atlhyper" }}
{{- end }}

{{/*
Image tag
*/}}
{{- define "atlhyper.imageTag" -}}
{{- .Values.global.imageTag | default "latest" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "atlhyper.labels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}

{{/*
Controller selector labels
*/}}
{{- define "atlhyper.controller.selectorLabels" -}}
app: atlhyper-controller
{{- end }}

{{/*
Agent selector labels
*/}}
{{- define "atlhyper.agent.selectorLabels" -}}
app: atlhyper-agent
{{- end }}

{{/*
Web selector labels
*/}}
{{- define "atlhyper.web.selectorLabels" -}}
app: atlhyper-web
{{- end }}

{{/*
Metrics selector labels
*/}}
{{- define "atlhyper.metrics.selectorLabels" -}}
app: atlhyper-metrics
{{- end }}

{{/*
Service URLs (自动生成)
*/}}
{{- define "atlhyper.controllerGatewayUrl" -}}
http://atlhyper-controller.{{ include "atlhyper.namespace" . }}.svc.cluster.local:8080
{{- end }}

{{- define "atlhyper.controllerAgentSdkUrl" -}}
http://atlhyper-controller.{{ include "atlhyper.namespace" . }}.svc.cluster.local:8081
{{- end }}

{{- define "atlhyper.agentServiceUrl" -}}
http://atlhyper-agent-service.{{ include "atlhyper.namespace" . }}.svc.cluster.local:8082
{{- end }}
