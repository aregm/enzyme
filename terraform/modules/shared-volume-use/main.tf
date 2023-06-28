# Create PV and PVC per namespace that point to a volume created by "shared-volume".

data "kubernetes_config_map" "config" {
  metadata {
    name = var.shared_volume_config_map
    namespace = var.shared_volume_namespace
  }
}

resource "kubernetes_persistent_volume" "shared-volume" {
  metadata {
    name = "${var.namespace}-${var.name}"
  }
  spec {
    capacity = {
      storage = data.kubernetes_config_map.config.data.capacity
    }
    access_modes = ["ReadWriteMany"]
    persistent_volume_source {
      csi {
        driver = data.kubernetes_config_map.config.data.driver
        node_stage_secret_ref {
          name = data.kubernetes_config_map.config.data.secret_name
          namespace = data.kubernetes_config_map.config.data.secret_namespace
        }
        volume_attributes = {
          clusterID = data.kubernetes_config_map.config.data.cluster_id
          fsName = data.kubernetes_config_map.config.data.fs_name
          rootPath = data.kubernetes_config_map.config.data.root_path
          staticVolume = "true"
        }
        volume_handle = "${var.namespace}-${var.name}"
      }
    }
    persistent_volume_reclaim_policy = "Retain"
    storage_class_name = data.kubernetes_config_map.config.data.storage_class_name
    volume_mode = "Filesystem"
  }
}

resource "kubernetes_persistent_volume_claim" "shared-volume" {
  metadata {
    name = var.name
    namespace = var.namespace
  }
  spec {
    access_modes = ["ReadWriteMany"]
    resources {
      requests = {
        storage = data.kubernetes_config_map.config.data.capacity
      }
    }
    storage_class_name = data.kubernetes_config_map.config.data.storage_class_name
    volume_name = kubernetes_persistent_volume.shared-volume.metadata.0.name
  }
}
