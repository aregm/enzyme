variable "dask_workers" {
  description = "Number of Dask workers"
  type = number
}

variable "ingress_domain" {
  description = "Ingress domain name"
  default = "localtest.me"
  type = string
}
