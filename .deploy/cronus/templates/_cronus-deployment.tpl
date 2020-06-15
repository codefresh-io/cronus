{{/*
We create Deployment resource as template to be able to use many deployments but with 
different name and version. This is for Istio POC.
*/}}
{{- define "cronus.renderDeployment" -}}
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ template "cronus.fullname" $ }}-{{ .version | default "base" }}
  labels:
    app: {{ template "cronus.fullname" . }}
    role: {{ template "cronus.role" . }}
    chart: "{{ .Chart.Name }}-{{ .Chart.Version | replace "+" "_" }}"
    release: {{ .Release.Name  | quote }}
    heritage: {{ .Release.Service  | quote }}
    version: {{ .version | default "base" | quote  }}
spec:
  strategy:
    type: Recreate
    rollingUpdate: null
  replicas: 1
  selector:
    matchLabels:
      app: {{ template "cronus.name" . }}
  template:
    metadata:
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/configmap.yaml") . | sha256sum }}
        sidecar.istio.io/inject: {{ $.Values.global.istio.enabled | default "false" | quote }}
      labels:
        app: {{ template "cronus.name" . }}
        role: {{ template "cronus.role" . }}
        release: {{ .Release.Name }}
        type: {{ .Values.event.type }}
        kind: {{ .Values.event.kind }}
        action: {{ .Values.event.action }}
        version: {{ .version | default "base" | quote  }}
    spec:
      {{- if not .Values.global.devEnvironment }}
      {{- $podSecurityContext := (kindIs "invalid" .Values.global.podSecurityContextOverride) | ternary .Values.podSecurityContext .Values.global.podSecurityContextOverride }}
      {{- with $podSecurityContext }}
      securityContext:
{{ toYaml . | indent 8}}
      {{- end }}
      {{- end }}
      volumes:
      - name: boltdb-store
        persistentVolumeClaim:
          claimName: {{ include "cronus.pvcName" . }}
      imagePullSecrets:
        - name: "{{ .Release.Name }}-{{ .Values.global.codefresh }}-registry"
      containers:
        - name: {{ .Chart.Name }}
          {{- if .Values.global.privateRegistry }}
          image: "{{ .Values.global.dockerRegistry }}{{ .Values.image.name }}:{{ .imageTag }}"
          {{- else }}
          image: "{{ .Values.image.dockerRegistry }}{{ .Values.image.name }}:{{ .imageTag }}"
          {{- end }}
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - containerPort: {{ .Values.service.internalPort }}
          env:
            {{- if .Values.global.env }}
              {{- range $key, $value := .Values.global.env }}
            - name: {{ $key }}
              value: {{ $value | quote }}
            {{- end}}
            {{- end}}
            - name: LOG_LEVEL
              value: {{ .Values.logLevel | quote }}
            {{- if ne .Values.logLevel "debug" }}
            - name: GIN_MODE
              value: release
            {{- end }}
            - name: HERMES_SERVICE
              value: {{ .Values.hermesService | default (printf "%s-hermes" .Release.Name) }}
            - name: PORT
              value: {{ .Values.service.internalPort | quote }}
            - name: STORE_FILE
              value: "/var/boltdb/events.db"
          volumeMounts:
            - mountPath: "/var/boltdb"
              name: boltdb-store
          {{- if not .Values.global.devEnvironment }}
          securityContext:
            allowPrivilegeEscalation: false
          {{- end }}
          livenessProbe:
            httpGet:
              path: /health
              port: {{ .Values.service.internalPort }}
            initialDelaySeconds: 5
            timeoutSeconds: 3
            periodSeconds: 10
            failureThreshold: 5
          readinessProbe:
            httpGet:
              path: /ping
              port: {{ .Values.service.internalPort }}
            initialDelaySeconds: 5
            timeoutSeconds: 3
            periodSeconds: 10
            failureThreshold: 5
          resources:
{{ toYaml .Values.resources | indent 12 }}
        {{- $nodeSelector := coalesce .Values.nodeSelector .Values.global.storagePodNodeSelector }}
        {{- if $nodeSelector }}
      nodeSelector:
{{ toYaml $nodeSelector | indent 8 }}
        {{- end }}
      {{- with (default .Values.global.appServiceTolerations .Values.tolerations ) }}
      tolerations:
{{ toYaml . | indent 8}}
      {{- end }}
      affinity:
{{ toYaml (default .Values.global.appServiceAffinity .Values.affinity) | indent 8 }}
{{- end }}
