# https://kubernetes.github.io/ingress-nginx/
resource "helm_release" "ingress_nginx" {
  name = "ingress-nginx"
  namespace = "ingress-nginx"
  create_namespace = true
  repository = "https://kubernetes.github.io/ingress-nginx"
  chart = "ingress-nginx"
  version = "4.3.0"
  # https://github.com/kubernetes/ingress-nginx/blob/main/charts/ingress-nginx/values.yaml
  values = [
    <<-EOT
      controller:
        ingressClassResource:
          default: true
        watchIngressWithoutClass: true
        service:
          enabled: ${var.ingress_nginx_service_enabled}
        kind: DaemonSet
        hostPort:
          enabled: true
          ports:
            http: 80
            https: 443
        admissionWebhooks:
          enabled: false
        config:
          use-forwarded-headers: true
    EOT
  ]
}
