{{- define "workloadType" -}}
{{- if .Values.vmConfig -}}
vm
{{- else -}}
pod
{{- end -}}
{{- end -}}