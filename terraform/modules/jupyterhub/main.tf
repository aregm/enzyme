locals {
  jupyterhub_default_profile = {
    display_name = "X1"
    description = "Prefect, Modin, Ray"
    kubespawner_override = {
      image = var.jupyterhub_singleuser_default_image
      # required for sudo
      allow_privilege_escalation = true
    }
    default = true
  }

  jupyterhub_oneapi_profile = {
    display_name = "X1 oneAPI"
    description = "Prefect, Modin, Ray with oneAPI Base, HPC, AI Toolkits"
    kubespawner_override = {
      image = var.jupyterhub_oneapi_profile_image
    }
  }

  # https://z2jh.jupyter.org/en/latest/resources/reference.html#singleuser-profilelist
  jupyterhub_profiles = concat(
    [
      local.jupyterhub_default_profile,
    ],
    var.jupyterhub_oneapi_profile_enabled ? [local.jupyterhub_oneapi_profile] : [],
  )

  jupyterhub_storage = {
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

  # https://z2jh.jupyter.org/en/latest/resources/reference.html#singleuser-extrafiles
  extra_files = {
    x1settings = {
      mountPath = "/etc/x1/settings.yaml"
      data = merge(
        {
          default_address = var.ingress_domain
        },
        !var.shared_volume_enabled ? {} : {
          (var.ingress_domain) = {
            prefect_shared_volume_mount = "/data"
          }
        },
      )
    }
  }
}

resource "kubernetes_namespace" "jupyterhub" {
  metadata {
    name = "jupyterhub"
    labels = var.namespace_labels
  }
}

resource "helm_release" "jupyterhub" {
  name = "jupyterhub"
  namespace = kubernetes_namespace.jupyterhub.id
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
    yamlencode({
      singleuser = {
        extraFiles = local.extra_files
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
        # https://zero-to-jupyterhub.readthedocs.io/en/latest/jupyterhub/customizing/user-environment.html#use-jupyterlab-by-default
        defaultUrl: /lab
        # https://jupyterhub-kubespawner.readthedocs.io/en/latest/spawner.html#kubespawner.KubeSpawner.start_timeout
        startTimeout: 600
        extraEnv:
          JUPYTERHUB_SINGLEUSER_APP: jupyter_server.serverapp.ServerApp
          PREFECT_API_URL: "${var.prefect_api_url}"
        networkPolicy:
          enabled: false
      proxy:
        service:
          type: NodePort
          nodePorts:
            http: 30004
        chp:
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
          - jupyter.${var.ingress_domain}
    EOT
  ]
}

module "shared-volume" {
  count = var.shared_volume_enabled ? 1 : 0
  source = "../shared-volume-use"
  namespace = kubernetes_namespace.jupyterhub.id
}
