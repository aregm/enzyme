locals {
  jupyterhub_code_server_profile = {
    display_name = "Code Server"
    description = "Visual Studio Code server"
    kubespawner_override = {
      image = "codercom/code-server:4.11.0"
      cmd = [
        "--auth", "none",
        "--bind-addr", "0.0.0.0:8888",
        "-vvv",
      ]
      # required for sudo
      allow_privilege_escalation = true
    }
    default = true
  }

  # https://z2jh.jupyter.org/en/latest/resources/reference.html#singleuser-profilelist
  jupyterhub_profiles = [
    local.jupyterhub_code_server_profile,
  ]

  jupyterhub_storage = {
    homeMountPath = "/home/coder"
    capacity = var.jupyterhub_singleuser_volume_size

    dynamic = {
      storageClass = var.default_storage_class
    }

    extraVolumes = !var.shared_volume_enabled ? [] : [
      {
        name = "data"
        persistentVolumeClaim = {
          claimName = module.shared-volume.0.claim_name
        }
      }
    ]

    extraVolumeMounts = !var.shared_volume_enabled ? [] : [
      {
        mountPath = "/data"
        name = "data"
      }
    ]
  }
}

resource "kubernetes_namespace" "vscode" {
  metadata {
    name = "vscode"
    labels = var.namespace_labels
  }
}

resource "helm_release" "vscode" {
  name = "vscode"
  namespace = kubernetes_namespace.vscode.id
  chart = "jupyterhub"
  repository = "https://jupyterhub.github.io/helm-chart"
  version = "2.0.0"
  timeout = 1200
  # See https://github.com/jupyterhub/zero-to-jupyterhub-k8s/blob/HEAD/jupyterhub/values.yaml
  # See https://zero-to-jupyterhub.readthedocs.io/en/latest/resources/reference.html
  values = [
    yamlencode({
      singleuser = {
        profileList = local.jupyterhub_profiles
      }
    }),
    yamlencode({
      singleuser = {
        storage = local.jupyterhub_storage
      }
    }),
    <<-EOT
      hub:
        db:
          pvc:
            storage: 1Gi
            storageClassName: "${var.default_storage_class}"
        networkPolicy:
          enabled: false
      singleuser:
        # https://jupyterhub-kubespawner.readthedocs.io/en/latest/spawner.html#kubespawner.KubeSpawner.start_timeout
        startTimeout: 600
        networkPolicy:
          enabled: false
      proxy:
        service:
          type: ClusterIP
        chp:
          extraCommandLineFlags:
            - --no-include-prefix
          networkPolicy:
            enabled: false
      prePuller:
        hook:
          enabled: ${var.jupyterhub_pre_puller_enabled}
        continuous:
          enabled: ${var.jupyterhub_pre_puller_enabled}
      cull:
        enabled: true
        timeout: 2592000 # 1 month
      ingress:
        enabled: true
        hosts:
          - vscode.${var.ingress_domain}
    EOT
  ]
}

module "shared-volume" {
  count = var.shared_volume_enabled ? 1 : 0
  source = "../shared-volume-use"
  namespace = kubernetes_namespace.vscode.id
}
