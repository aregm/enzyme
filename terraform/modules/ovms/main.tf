# OpenVINO Model Server operator
# https://operatorhub.io/operator/ovms-operator
# https://github.com/openvinotoolkit/operator
# https://github.com/openvinotoolkit/model_server

locals {
  kubernetes_manifest = {
    apiVersion = "operators.coreos.com/v1alpha1"
    kind = "Subscription"
    metadata = {
      name = "ovms-operator"
      namespace = "operators"
    }
    spec = {
      channel = "alpha"
      name = "ovms-operator"
      source = "operatorhubio-catalog"
      sourceNamespace = "olm"
    }
  }

  openshift_manifest = {
    apiVersion = "operators.coreos.com/v1alpha1"
    kind = "Subscription"
    metadata = {
      name = "ovms-operator"
      namespace = "openshift-operators"
    }
    spec = {
      channel = "alpha"
      name = "ovms-operator"
      source = "certified-operators"
      sourceNamespace = "openshift-marketplace"
    }
  }
}

resource "local_file" "ovms" {
  content  = var.openshift_enabled ? yamlencode(local.openshift_manifest) : yamlencode(local.kubernetes_manifest)
  filename = "/tmp/ovms-operator.yaml"
}

resource "null_resource" "ovms" {
  provisioner "local-exec" {
    command = <<-EOT
      kubectl apply -f ${local_file.ovms.filename}
    EOT
  }

  provisioner "local-exec" {
    when = destroy
    command = "kubectl --namespace ${self.triggers.namespace} delete subscription ovms-operator"
    on_failure = continue
  }

  triggers = {
    checksum = local_file.ovms.id
    namespace = var.openshift_enabled ? local.openshift_manifest.metadata.namespace : local.kubernetes_manifest.metadata.namespace
  }
}
