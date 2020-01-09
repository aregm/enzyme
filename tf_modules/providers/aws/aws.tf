provider "aws" {
  region     = "${var.region}"
  shared_credentials_file = "${path.module}/credentials"
  version    = "~> 1.8" # restrict update of aws provider to limit chances of breaking changes
}

provider "external" {
  version = "~> 1.0"
}

provider "null" {
  version = "~> 1.0"
}

variable "cidr_host_start" {
    default = 10
}

data "aws_ami" "centos_ami" {
  most_recent      = true
  filter {
    name   = "name"
    values = ["${var.image_name}"]
  }
  filter {
    name   = "architecture"
    values = ["x86_64"]
  }
  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

resource "aws_key_pair" "generated" {
  key_name   = "${var.key_name}"
  public_key = "${var.public_key}"
}

resource "aws_vpc" "cluster" {
  cidr_block = "10.10.0.0/16"
  tags {
    Name = "${var.cluster_name}"
  }
}

resource "aws_subnet" "cluster_subnet" {
  vpc_id = "${aws_vpc.cluster.id}"
  cidr_block = "10.10.10.0/24"
  tags = "${aws_vpc.cluster.tags}"
}

resource "aws_internet_gateway" "gw" {
  vpc_id = "${aws_vpc.cluster.id}"
  tags = "${aws_vpc.cluster.tags}"
}

resource "aws_security_group" "allow_incoming" {
  name = "allow_incoming"
  tags {
    Name = "${var.cluster_name}.allow_incoming_rule"
    description = "Allow all inbound traffic"
  }
  vpc_id = "${aws_vpc.cluster.id}"
  
  ingress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
  
  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["${aws_vpc.cluster.cidr_block}"]
  }
  
  revoke_rules_on_delete = true
}

resource "aws_security_group" "allow_interconnect" {
  name = "allow_interconnect"
  tags {
    Name = "${var.cluster_name}.allow_interconnect_rule"
    description = "Allow interconnect in cluster"
  }
  vpc_id = "${aws_vpc.cluster.id}"
  
  ingress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["${aws_vpc.cluster.cidr_block}"]
  }
  
  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["${aws_vpc.cluster.cidr_block}"]
  }
  
  revoke_rules_on_delete = true
}

resource "aws_route_table" "routes" {
  vpc_id = "${aws_vpc.cluster.id}"
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.gw.id}"
  }
  tags {
    Name = "${var.cluster_name}.default_gateway_route"
  }
}

resource "aws_route_table_association" "routes_assoc" {
  subnet_id = "${aws_subnet.cluster_subnet.id}"
  route_table_id = "${aws_route_table.routes.id}"
}

resource "aws_network_interface" "cluster_interconnect" {
  count = "${var.worker_count}"
  subnet_id = "${aws_subnet.cluster_subnet.id}"
  tags = "${aws_vpc.cluster.tags}"
  private_ips = ["${cidrhost(aws_subnet.cluster_subnet.cidr_block, count.index + var.cidr_host_start + 1)}"] // 1 for login node
  security_groups = ["${aws_security_group.allow_interconnect.id}"]
}

resource "aws_network_interface" "cluster_inbound" {
  subnet_id = "${aws_subnet.cluster_subnet.id}"
  tags {
    Name = "${var.cluster_name}.inbound_interface"
  }
  private_ips = ["${cidrhost(aws_subnet.cluster_subnet.cidr_block, var.cidr_host_start)}"]
  security_groups = ["${aws_security_group.allow_incoming.id}"]
}

resource "aws_instance" "worker" {
  count         = "${aws_network_interface.cluster_interconnect.count}"
  ami           = "${data.aws_ami.centos_ami.id}"
  instance_type = "${var.instance_type_worker_node}"
  network_interface {
    network_interface_id = "${aws_network_interface.cluster_interconnect.*.id[count.index]}"
    device_index = 0
  }
  tags = {
    Name = "${var.cluster_name}.worker-${count.index}"
  }
  key_name = "${var.key_name}"
}

resource "aws_instance" "login" {
  # login node, open to external access
  ami           = "${data.aws_ami.centos_ami.id}"
  instance_type = "${var.instance_type_login_node}"
  network_interface {
    network_interface_id = "${aws_network_interface.cluster_inbound.id}"
    device_index = 0
  }
  tags = {
    Name = "${var.cluster_name}.login"
  }
  key_name = "${var.key_name}"
  root_block_device {
    volume_size = "${var.login_node_root_size}"
    delete_on_termination = true
  }
}

resource "aws_eip" "external_access" {
  depends_on = ["aws_internet_gateway.gw"]
  instance = "${aws_instance.login.id}"
  tags = "${aws_vpc.cluster.tags}"
  vpc = true
}

module "provision" {
    source = "../../provision"
    general_path_module = "${var.general_path_module}"
    login_address = "${aws_eip.external_access.public_ip}"
    user_name = "${var.user_name}"
    key_name = "${var.key_name}"
    worker_count = "${var.worker_count}"
    all_instance_ids = "${concat(aws_instance.worker.*.id, list(aws_instance.login.id))}"
    all_instance_ips = "${concat(list(aws_instance.login.private_ip), aws_instance.worker.*.private_ip)}"
    #login_extra_disk_id = "${lookup(aws_instance.login.ebs_block_device[0], "volume_id")}"
    login_extra_disk_id = ""
    cluster_cidr_block = "${aws_vpc.cluster.cidr_block}"
}

output "login_address" {
  value = "${aws_eip.external_access.public_ip}"
}

output "centos_image_id" {
  value = "${data.aws_ami.centos_ami.id}"
}

output "workers_private_ip" {
  value = "${aws_instance.worker.*.private_ip}"
}
