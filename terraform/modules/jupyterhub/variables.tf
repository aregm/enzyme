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

variable "jupyterhub_singleuser_default_image" {
  description = "Default Docker image for JupyterHub default profile"
  type = string
}

variable "jupyterhub_oneapi_profile_enabled" {
  description = "Enable JupyterHub oneAPI profile"
  type = bool
  default = false
}

variable "jupyterhub_oneapi_profile_image" {
  description = "Docker image for JupyterHub oneAPI profile"
  type = string
  default = ""
}

variable "prefect_api_url" {
  description = "Prefect API URL"
  type = string
  default = "http://prefect-server.prefect:4200/api"
}

variable "prefect_image" {
  description = "Prefect Docker image"
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
