data "http" "crds" {
  url = "https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/master/deploy/upstream/quickstart/crds.yaml"
}

resource "local_file" "crds" {
  content  = data.http.crds.response_body
  filename = "/tmp/olm-crds.yaml"
}

data "http" "olm" {
  url = "https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/master/deploy/upstream/quickstart/olm.yaml"
}

resource "local_file" "olm" {
  content  = data.http.olm.response_body
  filename = "/tmp/olm-olm.yaml"
}

resource "null_resource" "crds" {
  provisioner "local-exec" {
    command = <<-EOT
      kubectl apply -f ${local_file.crds.filename}
      kubectl wait --timeout=60s --for=condition=established -f ${local_file.crds.filename}
    EOT
  }

  triggers = {
    checksum = local_file.crds.id
  }
}

resource "null_resource" "olm" {
  depends_on = [null_resource.crds]
  provisioner "local-exec" {
    command = <<-EOT
      kubectl apply -f ${local_file.olm.filename}
    EOT
  }

  triggers = {
    checksum = local_file.olm.id
  }
}
