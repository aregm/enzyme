resource "helm_release" "docker-registry" {
  name = "docker-registry"
  namespace = "docker-registry"
  create_namespace = true
  repository = "https://helm.twun.io"
  chart = "docker-registry"
  version = var.release
  values = [
    <<-EOT
      # https://github.com/twuni/docker-registry.helm
      persistence:
        enabled: true
        size: "${var.storage_size}"
        storageClass: "${var.storage_class}"
      service:
        type: NodePort
        nodePort: 30500
    EOT
  ]
}

# Docker registry proxy runs on each node and listens on port 5000. This is required to avoid "http: server gave HTTP
# response to HTTPS client" error when using Docker registry directly. Both docker and nerdctl allow using an HTTP
# endpoint to a local host, so the Docker registry can be used inside the cluster as "localhost:5000/ubuntu:latest".
# This is a workaround, the correct way is to use TLS termination for Docker registry and use a trusted certificate.
#
# Using the registry outside the cluster:
# With nerdctl: `nerdctl image push {node}:30500/ubuntu:20.04 --insecure-registry`
# With docker: add `{node}:30500` to "insecure-registries" in /etc/docker/daemon.json or run kube-registry-proxy locally.
resource "kubernetes_daemonset" "docker-registry-proxy" {
  depends_on = [helm_release.docker-registry]
  metadata {
    name = "docker-registry-proxy"
    namespace = "docker-registry"
  }
  spec {
    selector {
      match_labels = {
        name = "docker-registry-proxy"
      }
    }
    template {
      metadata {
        labels = {
          name = "docker-registry-proxy"
        }
      }
      spec {
        container {
          name = "docker-registry-proxy"
          image = "gcr.io/google_containers/kube-registry-proxy:0.4"
          env {
            name = "REGISTRY_HOST"
            value = "docker-registry.docker-registry.svc.cluster.local"
          }
          env {
            name = "REGISTRY_PORT"
            value = "5000"
          }
          port {
            name = "registry"
            container_port = 80
            host_port = 5000
          }
        }
      }
    }
  }
}
