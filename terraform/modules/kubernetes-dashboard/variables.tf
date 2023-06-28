variable "release" {
  description = "Version of kubernetes-dashboard Helm chart"
  default = "5.7.0"
  type = string
}

variable "ingress_domain" {
  description = "Ingress domain name"
  default = "localtest.me"
  type = string
}
