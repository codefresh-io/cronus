{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "cronus.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app role.
*/}}
{{- define "cronus.role" -}}
{{- default "trigger-cron" .Values.roleOverride -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "cronus.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Calculates Existing pvc name
*/}}
{{- define "cronus.existingPvc" -}}
{{- $existingPvc := coalesce .Values.existingPvc .Values.existingClaim .Values.pvcName | default "" -}}
{{- printf "%s" $existingPvc -}}
{{- end -}}

{{/*
Calculates pvcName
*/}}
{{- define "cronus.pvcName" -}}
{{- $pvcName := include "cronus.existingPvc" .  | default (include "cronus.fullname" . ) -}}
{{- printf "%s" $pvcName -}}
{{- end -}}

{{/*
Calculates storage class name
*/}}
{{- define "cronus.storageClass" -}}
{{- $storageClass := coalesce .Values.storageClass .Values.StorageClass .Values.global.storageClass | default "" -}}
{{- printf "%s" $storageClass -}}
{{- end -}}

{{/*
Calculates storage size
*/}}
{{- define "cronus.storageSize" -}}
{{- $storageSize := coalesce .Values.storageSize .Values.store.size -}}
{{- printf "%s" $storageSize -}}
{{- end -}}

{{/*
   Create a default fully qualified app name.
   We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
   If release name contains chart name it will be used as a full name.
  */}}
{{- define "cronus.fqdn" -}}
  {{- $name := "" -}}
  {{- if $.Values.fullnameOverride -}}
    {{- $name =$.Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
  {{- else -}}
    {{- $name = default .Chart.Name .Values.nameOverride -}}
    {{- if contains $name .Release.Name -}}
    {{- $name = .Release.Name | trunc 63 | trimSuffix "-" -}}
    {{- else -}}
    {{- $name = printf "%s-%s" .Release.Name $name -}}
  {{- end -}}
  {{- end -}}
{{- printf "%s.%s.svc.cluster.local" $name .Release.Namespace  | trunc 63 | trimSuffix "-" -}}
{{- end -}}