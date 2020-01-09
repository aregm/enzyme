variable login_address {}
variable user_name {}
variable key_name {}
variable worker_count {}
variable all_instance_ids {
    type = "list"
}
# NOTE: first entry in all_instance_ips MUST be login node, or stuff would break
variable all_instance_ips {
    type = "list"
}
variable cluster_cidr_block {}
variable login_extra_disk_id {}
variable pkey_file_path {}
variable postprocess_path {}


resource "null_resource" "cluster-node" {
  count = "${1 + var.worker_count}" // 1 for login node
  
  # changes to any node should require re-provisioning
  triggers = {
    nodes = "${join(",", var.all_instance_ids)}"
  }

  connection {
    type = "ssh"

    bastion_host = "${var.login_address}"
    
    host = "${var.all_instance_ips[count.index]}"
    user = "${var.user_name}"
    private_key = "${file("${var.pkey_file_path}")}"
  }
  
  # make sure we have passwordless SSH from login node to worker ones
  provisioner "file" {
    source = "${var.pkey_file_path}"
    destination = "~/.ssh/id_rsa"
  }
  provisioner "file" {
    content = "${var.all_instance_ips[0]}"
    destination = "~/.zyme-login-node"
  }
  provisioner "remote-exec" {
    inline = [
      "chmod 600 ~/.ssh/id_rsa",
      "mkdir ~/zyme-postprocess -p"
    ]
  }
  provisioner "file" {
    source = "${var.postprocess_path}"
    destination = "~/zyme-postprocess"
  }
  provisioner "remote-exec" {
    inline = [
      "find ~/zyme-postprocess -name '*.sh' -exec dos2unix {} \\;",
      "find ~/zyme-postprocess -name '*.sh' -exec chmod +x {} \\;",
      "ZYME_CLUSTER_CIDR=${var.cluster_cidr_block} ZYME_LOGIN_EXTRA_DISK_ID=${var.login_extra_disk_id} ~/zyme-postprocess/${count.index == 0 ? "bootstrap-head.sh" : "bootstrap-worker.sh"} ${join(" ", var.all_instance_ips)}"
    ]
  }
}

resource "null_resource" "mount_nfs_home" {
  depends_on = ["null_resource.cluster-node"]

  triggers = {
    nodes = "${join(",", var.all_instance_ids)}"
  }

  connection {
    type = "ssh"
    host = "${var.login_address}"
    user = "${var.user_name}"
    private_key = "${file("${var.pkey_file_path}")}"
  }

  provisioner "remote-exec" {
    inline = [
      "~/zyme-postprocess/finish-all-workers.sh ${join(" ", var.all_instance_ips)}"
    ]
  }
}