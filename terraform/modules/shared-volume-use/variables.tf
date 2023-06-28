variable "namespace" {
  description = "Namespace"
  type = string
}

variable "name" {
  description = "Name of PV and PVC"
  type = string
  default = "shared-volume"
}

variable "shared_volume_namespace" {
  description = "Namespace where shared volume is created"
  type = string
  default = "shared-volume"
}

variable "shared_volume_config_map" {
  description = "Name of ConfigMap that contains configuration for shared volume"
  type = string
  default = "shared-volume"
}
