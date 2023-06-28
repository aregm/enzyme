variable "ingress_nginx_service_enabled" {
  description = "Enable LoadBalancer service for ingress-nginx (required for AWS)"
  type = bool
  default = false
}
