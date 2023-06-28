variable "release" {
  description = "Version of Ceph operator"
  default = "v1.11.8"
  type = string
}

variable "ingress_domain" {
  description = "Ingress domain name"
  default = "localtest.me"
  type = string
}

variable "ceph_device_filter" {
  description = "Regex for devices that are available for Ceph, for example '^sd.'"
  type = string
  default = ".*"
}
