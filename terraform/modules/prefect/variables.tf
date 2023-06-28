variable "namespace_labels" {
  description = "Labels for namespace"
  type = map(string)
  default = {}
}

variable "agent_namespace" {
  description = "Namespace for Prefect agent"
  type = string
  default = "default"
}

variable "chart_version" {
  description = "Version of Prefect Helm chart"
  default = "2023.03.09"
  type = string
}

variable "image_tag" {
  description = "Tag of Prefect Docker image"
  type = string
}

variable "api_url" {
  description = "External URL for Prefect API to be used in UI, so it can communicate with the API from a web browser"
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
