# Local path provisioner allows to create PVCs with storageClassName: local-path
# https://github.com/rancher/local-path-provisioner

resource "null_resource" "repository" {
  provisioner "local-exec" {
    command = <<-EOT
      kubectl apply -k github.com/rancher/local-path-provisioner/deploy?ref=${var.release}
    EOT
  }

  triggers = {
    release = var.release
  }
}
