resource "helm_release" "ceph_operator" {
  name = "rook-ceph"
  namespace = "rook-ceph"
  create_namespace = true
  repository = "https://charts.rook.io/release"
  chart = "rook-ceph"
  version = var.release
}

resource "helm_release" "ceph_cluster" {
  depends_on = [helm_release.ceph_operator]
  name = "rook-ceph-cluster"
  namespace = "rook-ceph"
  create_namespace = true
  repository = "https://charts.rook.io/release"
  chart = "rook-ceph-cluster"
  version = var.release
  # https://github.com/rook/rook/blob/release-1.9/Documentation/Helm-Charts/ceph-cluster-chart.md
  # https://github.com/rook/rook/blob/release-1.9/deploy/charts/rook-ceph-cluster/values.yaml
  values = [
    <<-EOT
      toolbox:
        enabled: true
      cephClusterSpec:
        dashboard:
          enabled: true
          ssl: false
        storage:
          deviceFilter: "${var.ceph_device_filter}"
        # cephObjectStores:
        # TODO: instances == number of nodes
        # TODO: increase CPU and RAM limits for the gateway
    EOT
  ]
}

# TODO: create Ceph users
# https://github.com/rook/rook/blob/release-1.9/deploy/examples/object-user.yaml

resource "kubernetes_service" "dashboard" {
  depends_on = [helm_release.ceph_cluster]
  metadata {
    name = "dashboard-node-port"
    namespace = "rook-ceph"
  }
  spec {
    type = "NodePort"
    port {
      name = "http-dashboard"
      port = 7000
      target_port = 7000
      node_port = 30007
      protocol = "TCP"
    }
    selector = {
      app = "rook-ceph-mgr"
      ceph_daemon_id = "a"
      rook_cluster = "rook-ceph"
    }
  }
}

resource "kubernetes_ingress_v1" "dashboard" {
  depends_on = [helm_release.ceph_cluster]
  metadata {
    name = "dashboard"
    namespace = "rook-ceph"
  }
  spec {
    rule {
      host = "ceph.${var.ingress_domain}"
      http {
        path {
          path = "/"
          backend {
            service {
              name = "rook-ceph-mgr-dashboard"
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

resource "kubernetes_service" "objectstore" {
  depends_on = [helm_release.ceph_cluster]
  metadata {
    name = "ceph-node-port"
    namespace = "rook-ceph"
  }
  spec {
    type = "NodePort"
    port {
      name = "http"
      port = 80
      target_port = 8080
      node_port = 30008
      protocol = "TCP"
    }
    selector = {
      app = "rook-ceph-rgw"
      ceph_daemon_id = "ceph-objectstore"
      rgw = "ceph-objectstore"
      rook_cluster = "rook-ceph"
      rook_object_store = "ceph-objectstore"
    }
  }
}
