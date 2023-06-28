locals {
  local_chart = "/tmp/local-storage-provisioner-${var.release}"
}

resource "null_resource" "chart" {
  provisioner "local-exec" {
    command = <<-EOT
      git clone --config advice.detachedHead=false --quiet --depth 1 --branch ${var.release} -- \
        https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner.git \
        ${local.local_chart}
    EOT
  }

  triggers = {
    release = var.release
  }
}

# Local storage provisioner allows to create PVCs with storageClassName: local-storage
# https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner
resource "helm_release" "local_storage_provisioner" {
  depends_on = [null_resource.chart]
  name = "local-storage"
  namespace = "local-storage"
  create_namespace = true
  chart = "${local.local_chart}/helm/provisioner"
  values = [
    <<-EOT
      # https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/blob/master/helm/README.md
      # https://github.com/kubernetes-sigs/sig-storage-local-static-provisioner/blob/master/helm/provisioner/values.yaml
      classes:
        - name: local-storage
          hostDir: /mnt/disks
          volumeMode: Filesystem
          fsType: xfs
          blockCleanerCommand:
            - /scripts/quick_reset.sh
          storageClass: true
      daemonset:
        initContainers:
          - name: disk-mounter
            image: busybox:1.28
            command: ["/bin/sh", "-c"]
            # TODO: move list of disks to the inventory
            args:
              - for disk in ${join(" ", var.local_storage_disks)}; do
                [ -L /mnt/disks/$disk ] || ln -s /dev/$disk /mnt/disks/$disk;
                done
            volumeMounts:
              - name: provisioner-dev
                mountPath: /dev
              - name: local-storage
                mountPath: /mnt/disks
                mountPropagation: HostToContainer
    EOT
  ]
}
