variable "namespace_labels" {
  description = "Labels for namespace"
  type = map(string)
  default = {}
}

variable "default_storage_class" {
  description = "Kubernetes storage class"
  type = string
}

variable "jupyterhub_pre_puller_enabled" {
  description = "Enable JupyterHub image pre-puller"
  type = bool
}

variable "jupyterhub_singleuser_volume_size" {
  description = "Size of a persistent volume for a single user session"
  type = string
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
