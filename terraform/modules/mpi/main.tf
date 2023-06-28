resource "kubernetes_namespace" "mpi-operator" {
  metadata {
    name = "mpi-operator"
    labels = var.namespace_labels
  }
}

data "http" "mpi-operator" {
  url = "https://raw.githubusercontent.com/kubeflow/mpi-operator/v0.4.0/deploy/v2beta1/mpi-operator.yaml"
}

resource "local_file" "mpi-operator" {
  content  = data.http.mpi-operator.response_body
  filename = "/tmp/mpi-operator.yaml"
}

resource "null_resource" "mpi-operator" {
  depends_on = [kubernetes_namespace.mpi-operator]
  provisioner "local-exec" {
    command = <<-EOT
      kubectl apply -f ${local_file.mpi-operator.filename}
    EOT
  }

  triggers = {
    checksum = local_file.mpi-operator.id
  }
}
