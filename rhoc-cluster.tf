module "ssh_manager" {
  source                = "git::https://github.com/cloudposse/terraform-tls-ssh-key-pair.git?ref=0.11/master"
  namespace             = "hpc"
  stage                 = "dev"
  name                  = "${var.key_name}"
  ssh_public_key_path   = "${path.module}/secrets"
  private_key_extension = ".pem"
  public_key_extension  = ".pub"
  chmod_command         = "${var.chmod_command}"
  version               = "0.1.1"
}

module "aws_provider" {
    source = "./tf_modules/providers/aws"
    general_path_module = "${path.module}"
    worker_count = "${var.worker_count}"
    image_name = "${var.image_name}"
    region = "${var.aws_region}"
    instance_type_login_node = "${var.aws_instance_type_login_node}"
    instance_type_worker_node = "${var.aws_instance_type_worker_node}"
    login_node_root_size = "${var.login_node_root_size}"
    key_name = "${module.ssh_manager.key_name}"
    user_name = "${var.user_name}"
    public_key = "${module.ssh_manager.public_key}"
    cluster_name = "${var.cluster_name}"
}

module "gcp_provider" {
    source = "./tf_modules/providers/gcp"
    general_path_module = "${path.module}"
    worker_count = "${var.worker_count}"
    image_name = "${var.image_name}"
    region = "${var.gcp_region}"
    zone = "${var.gcp_zone}"
    instance_type_login_node = "${var.gcp_instance_type_login_node}"
    instance_type_worker_node = "${var.gcp_instance_type_worker_node}"
    login_node_root_size = "${var.login_node_root_size}"
    key_name = "${module.ssh_manager.key_name}"
    user_name = "${var.user_name}"
    public_key = "${module.ssh_manager.public_key}"
    cluster_name = "${var.cluster_name}"
    project_name = "${var.project_name}"
}

output "aws_login_address" { 
    value = "${module.aws_provider.login_address}"
} 

output "gcp_login_address" { 
    value = "${module.gcp_provider.login_address}"
} 
  
output "aws_centos_image_id" { 
    value = "${module.aws_provider.centos_image_id}"
} 

output "gcp_centos_image_id" { 
    value = "${module.gcp_provider.centos_image_id}"
}

output "username" {
    value = "${var.user_name}"
}

output "pkey_file" {
    value = "${path.module}/secrets/hpc-dev-${var.key_name}.pem"
}

output "worker_count" {
    value = "${var.worker_count}"
}

output "aws_workers_private_ip" {
    value = "${module.aws_provider.workers_private_ip}"
}

output "gcp_workers_private_ip" {
    value = "${module.gcp_provider.workers_private_ip}"
}
