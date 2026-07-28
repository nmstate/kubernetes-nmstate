{{- define "kubernetes-nmstate.operatorImage" -}}
{{- .Values.operator.image | default (printf "quay.io/nmstate/kubernetes-nmstate-operator:%s" .Chart.AppVersion) -}}
{{- end -}}

{{- define "kubernetes-nmstate.handlerImage" -}}
{{- .Values.handler.image | default (printf "quay.io/nmstate/kubernetes-nmstate-handler:%s" .Chart.AppVersion) -}}
{{- end -}}
