resource "helm_release" "cert-manager" {
  name = "cert-manager"
  namespace = "cert-manager"
  create_namespace = true
  repository = "https://charts.jetstack.io"
  chart = "cert-manager"
  version = "v1.10.0"
  # https://github.com/cert-manager/cert-manager/blob/master/deploy/charts/cert-manager/values.yaml
  values = [
    <<-EOT
      installCRDs: true
    EOT
  ]
}

locals {
  cluster_issuer_manifest = templatefile(
    "${path.module}/selfsigned-cluster-issuer.yaml",
    {
      ingress_domain = var.ingress_domain
    }
  )
}

resource "null_resource" "selfsigned-cluster-issuer" {
  depends_on = [helm_release.cert-manager]
  provisioner "local-exec" {
    command = <<-EOT
      cat <<'EOF' | kubectl --namespace cert-manager apply -f -
      ${local.cluster_issuer_manifest}
      EOF
    EOT
  }

  provisioner "local-exec" {
    when = destroy
    command = <<-EOT
      kubectl --namespace cert-manager delete clusterissuer selfsigned-cluster-issuer
    EOT
    on_failure = continue
  }

  triggers = {
    checksum = sha256(local.cluster_issuer_manifest)
  }
}
