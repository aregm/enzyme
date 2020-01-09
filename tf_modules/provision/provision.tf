data "external" "ensure_ssh_bastion" {
  program = ["${var.general_path_module}/tools/python",
             "${var.general_path_module}/tools/proxy_ssh.py",

             #TODO: pass at least "proxy_ssh.state" as a variable from zyme.py
             "${var.general_path_module}/.zyme-ssh-tunnel.state",
             "tunnel",
             "--public_address=${var.login_address}",

             "--username=${var.user_name}",
             "--pkey=${var.general_path_module}/secrets/${var.key_name}.pem",
             "--remote_bind=${var.all_instance_ips[0]}",
             "--local_bind=127.0.0.1:10022"]
}

resource "null_resource" "cluster-node" {
  count = "${1 + var.worker_count}" // 1 for login node
  
  # changes to any node should require re-provisioning
  triggers {
    nodes = "${join(",", var.all_instance_ids)}"
  }

  connection {
    type = "ssh"
    
    bastion_host = "${data.external.ensure_ssh_bastion.result.bastion_host}"
    bastion_port = "${data.external.ensure_ssh_bastion.result.bastion_port}"
    
    host = "${var.all_instance_ips[count.index]}"
    user = "${var.user_name}"
    private_key = "${file("${var.general_path_module}/secrets/${var.key_name}.pem")}"
  }
  
  # make sure we have passwordless SSH from login node to worker ones
  provisioner "file" {
    source = "${var.general_path_module}/secrets/${var.key_name}.pem"
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
    source = "${var.general_path_module}/postprocess/"
    destination = "~/zyme-postprocess"
  }
  provisioner "remote-exec" {
    inline = [
      "dos2unix ~/zyme-postprocess/*.sh",
      "chmod +x ~/zyme-postprocess/*.sh",
      "ZYME_CLUSTER_CIDR=${var.cluster_cidr_block} ZYME_LOGIN_EXTRA_DISK_ID=${var.login_extra_disk_id} ~/zyme-postprocess/${count.index == 0 ? "bootstrap-head.sh" : "bootstrap-worker.sh"} ${join(" ", var.all_instance_ips)}"
    ]
  }
}