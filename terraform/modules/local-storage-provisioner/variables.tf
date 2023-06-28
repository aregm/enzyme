variable "release" {
  description = "Version of local-storage-provisioner"
  default = "v2.5.0"
  type = string
}

variable "local_storage_disks" {
  description = "List of disks managed by local-storage-provisioner, for example ['sda']"
  type = list(string)
  default = []
}
