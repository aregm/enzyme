resource "helm_release" "clearml" {
  name = "clearml"
  namespace = "clearml"
  create_namespace = true
  repository = "https://allegroai.github.io/clearml-helm-charts"
  chart = "clearml"
  version = "7.0.1"
  timeout = 1200
  values = [
    <<-EOT
      apiserver:
        service:
          type: ClusterIP
        ingress:
          enabled: true
          hostName: "api.clearml.${var.ingress_domain}"
      webserver:
        service:
          type: ClusterIP
        ingress:
          enabled: true
          hostName: "app.clearml.${var.ingress_domain}"
      fileserver:
        service:
          type: ClusterIP
        ingress:
          enabled: true
          hostName: "files.clearml.${var.ingress_domain}"
    EOT
  ]
}