variable "release" {
  description = "Version of docker-registry Helm chart"
  default = "2.1.0"
  type = string
}

variable "storage_class" {
  description = "Storage class for docker-registry volume"
  type = string
}

variable "storage_size" {
  description = "Storage cize for docker-registry volume"
  type = string
}
