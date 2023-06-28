variable "namespace_labels" {
  description = "Labels for namespace"
  type = map(string)
  default = {}
}

variable "release" {
  description = "Version of KubeRay"
  default = "0.4.0"
  type = string
}

variable "ray_image" {
  description = "Full tag for Ray Docker image"
  type = string
}

variable "ray_object_store" {
  description = "Size of Ray object store, bytes"
  type = number
}

variable "ray_worker_nodes" {
  description = "Number of Ray worker nodes"
  type = number
}

variable "ray_load_balancer_enabled" {
  description = "Enable LoadBalancer service on port 80 for Ray client endpoint"
  type = bool
  default = false
}

variable "ingress_domain" {
  description = "Ingress domain name"
  default = "localtest.me"
  type = string
}

variable "shared_volume_enabled" {
  description = "Enable shared volume"
  type = bool
  default = false
}
