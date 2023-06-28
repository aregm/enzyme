variable "namespace_labels" {
  description = "Labels for namespace"
  type = map(string)
  default = {}
}

variable "shared_volume_size" {
  description = "Size of shared volume"
  type = string
  default = "64Gi"
}

variable "shared_volume_storage_class" {
  description = "Storage class for shared volume"
  type = string
  default = "ceph-filesystem"
}

variable "ceph_secret_namespace" {
  description = "Ceph secret namespace"
  type = string
  default = "rook-ceph"
}

variable "ceph_secret_name" {
  description = "Ceph secret name"
  type = string
  default = "rook-csi-cephfs-node"
}
