# https://helm.dask.org/
# https://kubernetes.dask.org/en/latest/operator_installation.html
# https://github.com/dask/dask-kubernetes/blob/main/dask_kubernetes/operator/deployment/helm/dask-kubernetes-operator/
resource "helm_release" "dask-operator" {
  name = "dask"
  namespace = "dask"
  create_namespace = true
  chart = "dask-kubernetes-operator"
  repository = "https://helm.dask.org"
  version = "2022.7.0"
}

locals {
  dask_cluster_manifest = templatefile(
    "${path.module}/dask-cluster.yaml",
    {
      dask_workers = var.dask_workers
    }
  )
}

resource "null_resource" "dask-cluster" {
  depends_on = [helm_release.dask-operator]
  provisioner "local-exec" {
    command = <<-EOT
      cat <<'EOF' | kubectl --namespace dask apply -f -
      ${local.dask_cluster_manifest}
      EOF
    EOT
  }

  provisioner "local-exec" {
    when = destroy
    command = <<-EOT
      kubectl --namespace dask delete daskcluster dask
    EOT
    on_failure = continue
  }

  triggers = {
    checksum = sha256(local.dask_cluster_manifest)
  }
}

resource "kubernetes_ingress_v1" "dashboard" {
  depends_on = [helm_release.dask-operator]
  metadata {
    name = "dashboard"
    namespace = "dask"
  }
  spec {
    rule {
      host = "dask.${var.ingress_domain}"
      http {
        path {
          path = "/"
          backend {
            service {
              name = "dask-service"
              port {
                name = "http-dashboard"
              }
            }
          }
        }
      }
    }
  }
}
