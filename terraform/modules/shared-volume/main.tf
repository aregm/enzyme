# Shared volume backed by Ceph filesystem volume that is created in this namespace and can be used
# in other namespaces by creating a static PV and PVC that point to the same volume.

resource "kubernetes_namespace" "shared-volume" {
  metadata {
    name = "shared-volume"
    labels = var.namespace_labels
  }
}

resource "kubernetes_persistent_volume_claim" "shared-volume" {
  metadata {
    name = "shared-volume"
    namespace = kubernetes_namespace.shared-volume.id
  }
  spec {
    access_modes = ["ReadWriteMany"]
    resources {
      requests = {
        storage = var.shared_volume_size
      }
    }
    storage_class_name = var.shared_volume_storage_class
  }
  wait_until_bound = true
}

# Creates a new secret from rook-ceph/rook-csi-cephfs-node replacing "adminID" and "adminKey" with
# "userID" and "userKey". This secret is required for static PV created by "shared-volume-use".
resource "terraform_data" "secret" {
  triggers_replace = [
    var.ceph_secret_namespace,
    var.ceph_secret_name,
  ]

  provisioner "local-exec" {
    command = <<-EOT
      kubectl --namespace ${var.ceph_secret_namespace} \
        get secret ${var.ceph_secret_name} -o go-template='{{.data.adminID | base64decode}}' > /tmp/${var.ceph_secret_name}-id
      kubectl --namespace ${var.ceph_secret_namespace} \
        get secret ${var.ceph_secret_name} -o go-template='{{.data.adminKey | base64decode}}' > /tmp/${var.ceph_secret_name}-key
      kubectl --namespace ${kubernetes_namespace.shared-volume.id} \
        create secret generic --type kubernetes.io/rook ${var.ceph_secret_name} \
        --from-file=userID=/tmp/${var.ceph_secret_name}-id \
        --from-file=userKey=/tmp/${var.ceph_secret_name}-key
    EOT
  }
}

data "kubernetes_resource" "shared-volume" {
  depends_on = [
    kubernetes_persistent_volume_claim.shared-volume,
    terraform_data.secret,
  ]

  api_version = "v1"
  kind = "PersistentVolume"

  metadata {
    name = kubernetes_persistent_volume_claim.shared-volume.spec.0.volume_name
    namespace = kubernetes_namespace.shared-volume.id
  }
}

resource "kubernetes_config_map" "shared-volume" {
  metadata {
    name = "shared-volume"
    namespace = kubernetes_namespace.shared-volume.id
  }
  data = {
    capacity = try(data.kubernetes_resource.shared-volume.object.spec.capacity.storage, "")
    driver = try(data.kubernetes_resource.shared-volume.object.spec.csi.driver, "")
    storage_class_name = var.shared_volume_storage_class
    secret_name = var.ceph_secret_name
    secret_namespace = kubernetes_namespace.shared-volume.id
    cluster_id = try(data.kubernetes_resource.shared-volume.object.spec.csi.volumeAttributes.clusterID, "")
    fs_name = try(data.kubernetes_resource.shared-volume.object.spec.csi.volumeAttributes.fsName, "")
    root_path = try(data.kubernetes_resource.shared-volume.object.spec.csi.volumeAttributes.subvolumePath, "")
  }
}
