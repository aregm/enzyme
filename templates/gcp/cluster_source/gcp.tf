variable worker_count {}
variable image_name {}
variable region {}
variable instance_type_login_node {}
variable instance_type_worker_node {}
variable user_name {}
variable public_key {}
variable cluster_name {}
variable login_node_root_size {}
variable zone {}
variable project_name {}
variable credential_path {}

/*TODO
resource "google_project" "my_project" {
  name = "My Project"
  project_id = "your-project-id"
  org_id     = "1234567"
}
*/

provider "google" {
  credentials = "${file("${var.credential_path}")}"
  project     = "${var.project_name}"
  region      = "${var.region}"
  version = "~> 2.5"
}

provider "external" {
  version = "~> 1.0"
}

provider "null" {
  version = "~> 2.1"
}

variable "network_ip_range" {
    default = "10.10.0.0/16"
}

variable "cidr_host_start" {
    default = 10
}

data "google_compute_image" "centos_image" {
  name    = "${var.image_name}"
  project = "${var.project_name}"
}

resource "google_compute_network" "cluster" {
  routing_mode = "GLOBAL"
  auto_create_subnetworks = "false"
  name = "${var.cluster_name}"
}

resource "google_compute_address" "login_public" {
  name = "${var.cluster_name}-login-node-public"
}

resource "google_compute_subnetwork" "cluster_subnet" {
  name          = "${google_compute_network.cluster.name}"
  ip_cidr_range = "10.10.10.0/24"
  network       = "${google_compute_network.cluster.self_link}"
}

resource "google_compute_firewall" "allow_incoming_ingress_rule" {
  name    = "${var.cluster_name}-allow-incoming-ingress-rule"
  description = "Allow all inbound traffic"
  
  network = "${google_compute_network.cluster.name}"
  
  direction = "INGRESS"
  source_ranges = ["0.0.0.0/0"]
  
  allow {   
    protocol = "all"
  }
  target_tags = ["login"]
}

resource "google_compute_firewall" "egress_rule" {
  name    = "${var.cluster_name}-egress-rule"
  description = "Allow outbound traffic between login node and worker nodes"
  
  network = "${google_compute_network.cluster.name}"
  
  direction = "EGRESS"
  destination_ranges = ["${var.network_ip_range}"]
  
  allow {   
    protocol = "all"
  } 
  target_tags = ["login", "workers"]
}

resource "google_compute_firewall" "allow_interconnect_ingress_rule" {
  name    = "${var.cluster_name}-allow-interconnect-ingress-rule"
  description = "Allow interconnect"
  
  network = "${google_compute_network.cluster.name}"
  
  direction = "INGRESS"
  source_ranges = ["${var.network_ip_range}"]
  
  allow {   
    protocol = "all"
  } 
  target_tags = ["workers"]
}

resource "google_compute_instance" "worker" {
  count        = "${var.worker_count}" 
  name         = "${var.cluster_name}-worker-${count.index}"
  machine_type = "${var.instance_type_worker_node}"
  zone      = "${var.zone}"
  
  tags = ["workers"]

  boot_disk {
    initialize_params {
      image = "${data.google_compute_image.centos_image.self_link}"
    }
  }
  network_interface {
    subnetwork = "${google_compute_subnetwork.cluster_subnet.name}"
    network_ip = "${cidrhost(google_compute_subnetwork.cluster_subnet.ip_cidr_range, count.index + var.cidr_host_start + 1)}" // 1 for login node
  }
  metadata = {
    "sshKeys" = "${var.user_name}:${var.public_key}"
  }
}

resource "google_compute_instance" "login" {
  # login node, open to external access
  name = "${var.cluster_name}-login"
  machine_type = "${var.instance_type_login_node}"
  zone      = "${var.zone}"
  can_ip_forward = "true"
  tags = ["login"]
  
  boot_disk {
    initialize_params {
      image = "${data.google_compute_image.centos_image.self_link}"
    }
  }
  network_interface {
    subnetwork = "${google_compute_subnetwork.cluster_subnet.name}"
    network_ip = "${cidrhost(google_compute_subnetwork.cluster_subnet.ip_cidr_range, var.cidr_host_start)}"
    
     access_config {
      nat_ip = "${google_compute_address.login_public.address}"
    }
  }
  metadata = {
    "sshKeys" = "${var.user_name}:${var.public_key}"
  }
}

output "login_address" {
  value = "${google_compute_instance.login.network_interface.0.access_config.0.nat_ip}"
}

output "centos_image_id" {
  value = "${data.google_compute_image.centos_image.self_link}"
}

output "workers_private_ip" {
  value = "${google_compute_instance.worker.*.network_interface.0.network_ip}"
}

output "all_instance_ids" {
  value = "${concat(google_compute_instance.worker.*.instance_id, list(google_compute_instance.login.instance_id))}"
}

# NOTE: first entry in all_instance_ips MUST be login node, or stuff would break
output "all_instance_ips" {
  value = "${concat(list(google_compute_instance.login.network_interface.0.network_ip), google_compute_instance.worker.*.network_interface.0.network_ip)}"
}

output "network_ip_range" {
  value = "${var.network_ip_range}"
}

output "network_cluster_id" {
  value ="${google_compute_network.cluster.id}"
}

output "subnetwork_cluster_subnet_id" {
  value = "${google_compute_subnetwork.cluster_subnet.id}"
}