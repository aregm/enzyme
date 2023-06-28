resource "kubernetes_namespace" "ray" {
  metadata {
    name = "ray"
    labels = var.namespace_labels
  }
}

resource "helm_release" "ray-operator" {
  name = "ray-operator"
  namespace = kubernetes_namespace.ray.id
  repository = "https://ray-project.github.io/kuberay-helm/"
  chart = "kuberay-operator"
  version = var.release
  # https://github.com/ray-project/kuberay/blob/master/helm-chart/kuberay-operator/values.yaml
}

locals {
  volumes = concat(
    [
      {
        name = "log-volume"
        emptyDir = {}
      }
    ],
    !var.shared_volume_enabled ? [] : [
      {
        name = "data"
        persistentVolumeClaim = {
          claimName = module.shared-volume.0.claim_name
        }
      }
    ],
  )

  volumeMounts = concat(
    [
      {
        mountPath = "/tmp/ray"
        name = "log-volume"
      }
    ],
    !var.shared_volume_enabled ? [] : [
      {
        mountPath = "/data"
        name = "data"
      }
    ]
  )
}

resource "helm_release" "ray" {
  depends_on = [helm_release.ray-operator]
  name = "ray"
  namespace = kubernetes_namespace.ray.id
  repository = "https://ray-project.github.io/kuberay-helm/"
  chart = "ray-cluster"
  version = var.release
  # https://github.com/ray-project/kuberay/blob/master/helm-chart/ray-cluster/values.yaml
  values = [
    yamlencode({
      head = {
        volumes = local.volumes
        volumeMounts = local.volumeMounts
      }
      worker = {
        volumes = local.volumes
        volumeMounts = local.volumeMounts
      }
    }),
    <<-EOT
      image:
        repository: '${split(":", var.ray_image)[0]}'
        tag: '${split(":", var.ray_image)[1]}'
      head:
        replicas: 1
        # TODO: use ray_node_ram, ray_node_cpu
        resources:
          requests:
            cpu: null
            memory: null
          limits:
            cpu: null
            memory: null
        rayStartParams:
          dashboard-host: '0.0.0.0'
          block: 'true'
          object-store-memory: "${var.ray_object_store}"
          disable-usage-stats: 'true'
        affinity:
          podAntiAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
              - weight: 1
                podAffinityTerm:
                  labelSelector:
                    matchExpressions:
                      - key: ray.io/is-ray-node
                        operator: In
                        values:
                          - "yes"
                  topologyKey: kubernetes.io/hostname
      worker:
        replicas: "${var.ray_worker_nodes}"
        miniReplicas: "${var.ray_worker_nodes}"
        maxiReplicas: "${var.ray_worker_nodes}"
        # TODO: use ray_node_ram, ray_node_cpu
        resources:
          requests:
            cpu: null
            memory: null
          limits:
            cpu: null
            memory: null
        rayStartParams:
          object-store-memory: "${var.ray_object_store}"
        affinity:
          podAntiAffinity:
            preferredDuringSchedulingIgnoredDuringExecution:
              - weight: 1
                podAffinityTerm:
                  labelSelector:
                    matchExpressions:
                      - key: ray.io/is-ray-node
                        operator: In
                        values:
                          - "yes"
                  topologyKey: kubernetes.io/hostname
    EOT
  ]
}

# Service that is accessible via name "ray.ray" in the cluster
resource "kubernetes_service" "ray-service" {
  depends_on = [helm_release.ray]
  metadata {
    name = "ray"
    namespace = kubernetes_namespace.ray.id
  }
  spec {
    type = "ExternalName"
    external_name = "ray-kuberay-head-svc.${kubernetes_namespace.ray.id}.svc.cluster.local"
  }
}

# NodePort service for Ray
resource "kubernetes_service" "ray-node-port" {
  depends_on = [helm_release.ray]
  metadata {
    name = "ray-node-port"
    namespace = kubernetes_namespace.ray.id
  }
  spec {
    type = "NodePort"
    port {
      name = "dashboard"
      port = 8265
      target_port = 8265
      protocol = "TCP"
      node_port = 30002
    }
    port {
      name = "client"
      port = 10001
      target_port = 10001
      protocol = "TCP"
      node_port = 30009
    }
    selector = {
      "ray.io/node-type" = "head"
    }
  }
}

# LoadBalancer service on port 80 for Ray client port 10001
resource "kubernetes_service" "ray-client" {
  count = var.ray_load_balancer_enabled ? 1 : 0
  depends_on = [helm_release.ray]
  metadata {
    name = "ray-client"
    namespace = kubernetes_namespace.ray.id
  }
  spec {
    type = "LoadBalancer"
    port {
      name = "client"
      port = 80
      target_port = 10001
      protocol = "TCP"
    }
    selector = {
      "ray.io/node-type" = "head"
    }
  }
}

resource "kubernetes_ingress_v1" "dashboard" {
  depends_on = [helm_release.ray]
  metadata {
    name = "dashboard"
    namespace = kubernetes_namespace.ray.id
  }
  spec {
    rule {
      host = "ray.${var.ingress_domain}"
      http {
        path {
          path = "/"
          backend {
            service {
              name = "ray-node-port"
              port {
                name = "dashboard"
              }
            }
          }
        }
      }
    }
  }
}

module "shared-volume" {
  count = var.shared_volume_enabled ? 1 : 0
  source = "../shared-volume-use"
  namespace = kubernetes_namespace.ray.id
}
