variable "minio_ha_enabled" {
  description = "MinIO HA mode (true), or a single server mode (false)"
  type = bool
}

variable "default_storage_class" {
  description = "Kubernetes storage class to use for MinIO tools "
  type = string
}

variable "ingress_domain" {
  description = "Ingress domain name"
  default = "localtest.me"
  type = string
}

variable "minio_servers" {
  description = "Number of MinIO servers in HA configuration"
  type = number
  default = 1
}
