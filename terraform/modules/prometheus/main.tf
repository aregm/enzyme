locals {
  prometheus = {
    prometheusSpec = {
      storageSpec = {
        volumeClaimTemplate = {
          spec = {
            storageClassName = var.storage_class
            resources = {
              requests = {
                storage = var.storage_size
              }
            }
          }
        }
      }
    }
  }

  grafana = {
    service = {
      type = "NodePort"
      nodePort = 30000
    }
    ingress = {
      enabled: true
      hosts = [
        "grafana.${var.ingress_domain}"
      ]
    }
  }
}

# Metrics Server collects resource metrics from kubelets and exposes them in Kubernetes apiserver through Metrics
# API. Metrics API can also be accessed by `kubectl top pods` and `kubectl top nodes`.
# https://github.com/kubernetes-sigs/metrics-server/
resource "helm_release" "metrics_server" {
  count = var.metrics_server_enabled ? 1 : 0
  name = "metrics-server"
  namespace = "kube-system"
  repository = "https://kubernetes-sigs.github.io/metrics-server/"
  chart = "metrics-server"
  version = "3.8.4"
  values = [
    # Disable certificate validation, kubelet certificate needs to be signed by cluster CA
    "args: [--kubelet-insecure-tls]",
  ]
}

# Kubernetes Prometheus stack includes Prometheus, Alertmanager, Grafana.
# https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack
resource "helm_release" "prometheus" {
  name = "prometheus"
  namespace = "prometheus"
  create_namespace = true
  repository = "https://prometheus-community.github.io/helm-charts/"
  chart = "kube-prometheus-stack"
  version = "45.9.1"
  # Takes longer on Kind cluster
  timeout = 1200
  # https://github.com/prometheus-community/helm-charts/blob/main/charts/kube-prometheus-stack/values.yaml
  values = [
    yamlencode({
      prometheus = local.prometheus
      grafana = local.grafana
    }),
  ]
}

# https://github.com/prometheus-community/helm-charts/tree/main/charts/prometheus-adapter
resource "helm_release" "prometheus-adapter" {
  name = "prometheus-adapter"
  namespace = "prometheus"
  create_namespace = true
  repository = "https://prometheus-community.github.io/helm-charts/"
  chart = "prometheus-adapter"
  version = "4.1.1"
  values = [
    <<-EOT
      prometheus:
        url: http://prometheus-kube-prometheus-prometheus.prometheus.svc
    EOT
  ]
}
