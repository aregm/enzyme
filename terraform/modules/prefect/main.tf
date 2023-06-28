resource "kubernetes_namespace" "prefect" {
  metadata {
    name = "prefect"
    labels = var.namespace_labels
  }
}

locals {
  agent_namespace_manifest = templatefile(
    "${path.module}/agent-namespace.yaml",
    {
      namespace_labels = var.namespace_labels
    }
  )
}

# Use "null_resource" with "kubectl apply" instead of "kubernetes_namespace" because the default
# namespace for Prefect agent is "default" and resource "kubernetes_namespace" fails when namespace
# already exists. At the same time we have to apply "namespace_labels" to the namespace.
resource "null_resource" "agent-namespace" {
  provisioner "local-exec" {
    command = <<-EOT
      cat <<'EOF' | kubectl apply -f -
      ${local.agent_namespace_manifest}
      EOF
    EOT
  }

  triggers = {
    checksum = sha256(local.agent_namespace_manifest)
  }
}

resource "helm_release" "prefect-server" {
  name = "prefect-server"
  namespace = kubernetes_namespace.prefect.id
  repository = "https://prefecthq.github.io/prefect-helm"
  chart = "prefect-server"
  version = var.chart_version
  values = [
    <<-EOT
      namespaceOverride: prefect
      server:
        image:
          prefectTag: "${var.image_tag}"
        publicApiUrl: "${var.api_url == "" ? "http://prefect.${var.ingress_domain}/api" : var.api_url}"
      service:
        type: NodePort
        nodePort: 30001
      ingress:
        enabled: true
        host:
          hostname: prefect.${var.ingress_domain}
      # TODO: install postgresql separately, postgresql.useSubChart = true is not recommended for production
      postgresql:
        enabled: true
        useSubChart: true
        auth:
          password: e0e2eda98519739fa4656e4cc502841b
    EOT
  ]
}

resource "helm_release" "prefect-agent" {
  depends_on = [null_resource.agent-namespace]
  name = "prefect-agent"
  namespace = var.agent_namespace
  repository = "https://prefecthq.github.io/prefect-helm"
  chart = "prefect-agent"
  version = var.chart_version
  values = [
    <<-EOT
      namespaceOverride: default
      agent:
        image:
          prefectTag: "${var.image_tag}"
        config:
          workQueues:
            - prod
        apiConfig: server
        serverApiConfig:
          apiUrl: http://prefect-server.prefect:4200/api
        resources: {}
    EOT
  ]
}

module "shared-volume" {
  count = var.shared_volume_enabled ? 1 : 0
  depends_on = [null_resource.agent-namespace]
  source = "../shared-volume-use"
  namespace = "default"
}
