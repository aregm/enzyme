variable "metrics_server_enabled" {
  description = "Enable metrics-server (https://github.com/kubernetes-sigs/metrics-server/)"
  type = bool
  default = true
}

variable "ingress_domain" {
  description = "Ingress domain name"
  default = "localtest.me"
  type = string
}

variable "storage_class" {
  description = "Storage class for Prometheus"
  type = string
  default = ""
}

variable "storage_size" {
  description = "Storage class for Prometheus"
  type = string
  default = "50Gi"
}
