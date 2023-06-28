# https://github.com/minio/operator/tree/master/helm/operator
resource "helm_release" "operator" {
  name = "operator"
  namespace = "minio"
  create_namespace = true
  chart = "https://github.com/minio/operator/raw/v4.5.1/helm-releases/operator-4.5.1.tgz"
  values = [
    <<-EOT
      console:
        ingress:
          enabled: true
          host: "minio.${var.ingress_domain}"
    EOT
  ]
}

# Helm chart for MinIO operator does not allow changing a Service type for its console, so creating a dedicated NodePort
# Service with the same selector.
resource "kubernetes_service" "console" {
  depends_on = [helm_release.operator]
  metadata {
    name = "console-node-port"
    namespace = "minio"
  }
  spec {
    type = "NodePort"
    port {
      name = "console"
      port = 9090
      target_port = 9090
      protocol = "TCP"
      node_port = 30003
    }
    selector = {
      "app.kubernetes.io/instance" = "operator-console"
      "app.kubernetes.io/name" = "operator"
    }
  }
}

# Helm chart for MinIO does not allow changing a Service type for its S3 endpoint, so creating a dedicated NodePort
# Service with the same selector.
resource "kubernetes_service" "endpoint" {
  depends_on = [helm_release.operator]
  metadata {
    name = "minio-node-port"
    namespace = "minio"
  }
  spec {
    type = "NodePort"
    port {
      name = "https-minio"
      port = 443
      target_port = 9000
      protocol = "TCP"
      node_port = 30006
    }
    selector = {
      "v1.min.io/tenant" = "minio"
    }
  }
}

resource "kubernetes_ingress_v1" "endpoint" {
  depends_on = [helm_release.operator]
  metadata {
    name = "minio"
    namespace = "minio"
    annotations = {
      "nginx.ingress.kubernetes.io/backend-protocol" = "HTTPS"
      "nginx.ingress.kubernetes.io/proxy-buffering" = "off"
      "nginx.ingress.kubernetes.io/proxy-body-size" = "0"
      "nginx.ingress.kubernetes.io/server-snippet" = <<-EOF
        # To allow special characters in headers
        ignore_invalid_headers off;
      EOF
    }
  }
  spec {
    rule {
      host = "s3.${var.ingress_domain}"
      http {
        path {
          path = "/"
          backend {
            service {
              name = "minio"
              port {
                name = "https-minio"
              }
            }
          }
        }
      }
    }
  }
}

resource "kubernetes_secret" "x1miniouser" {
  depends_on = [helm_release.operator]
  metadata {
    name = "x1miniouser"
    namespace = "minio"
  }
  data = {
    CONSOLE_ACCESS_KEY = "x1miniouser"
    CONSOLE_SECRET_KEY = "x1miniopass"
  }
}

# https://github.com/minio/operator/tree/master/helm/tenant
resource "helm_release" "minio_ha" {
  depends_on = [helm_release.operator, kubernetes_secret.x1miniouser]
  count = var.minio_ha_enabled ? 1 : 0
  name = "minio"
  namespace = "minio"
  create_namespace = true
  chart = "https://github.com/minio/operator/raw/v4.5.1/helm-releases/tenant-4.5.1.tgz"
  values = [
    <<-EOT
      secrets:
        name: minio-env-configuration
      tenant:
        name: minio
        pools:
          - name: pool-0
            servers: ${var.minio_servers}
            # TODO: move to variables
            volumesPerServer: 4
            size: 1.5T
            storageClassName: local-storage
        prometheus:
          storageClassName: "${var.default_storage_class}"
        log:
          db:
            volumeClaimTemplate:
              spec:
                storageClassName: "${var.default_storage_class}"
        buckets:
          - name: prefect
        users:
          - name: x1miniouser
    EOT
  ]
}

# https://github.com/minio/operator/tree/master/helm/tenant
resource "helm_release" "minio" {
  depends_on = [helm_release.operator, kubernetes_secret.x1miniouser]
  count = var.minio_ha_enabled ? 0 : 1
  name = "minio"
  namespace = "minio"
  create_namespace = true
  chart = "https://github.com/minio/operator/raw/v4.5.1/helm-releases/tenant-4.5.1.tgz"
  values = [
    <<-EOT
      secrets:
        name: minio-env-configuration
      tenant:
        name: minio
        pools:
          - name: pool-0
            servers: 1
            volumesPerServer: 1
            size: 10Gi
            storageClassName: "${var.default_storage_class}"
        prometheus: null
        log: null
        buckets:
          - name: prefect
        users:
          - name: x1miniouser
    EOT
  ]
}

