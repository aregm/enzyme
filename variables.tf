variable worker_count {}

variable image_name {
    default = "zyme-worker-node"
}

variable aws_region {
    default = "us-west-2"
}

variable gcp_region {
    default = "us-east1"
}

variable gcp_zone {
    default = "us-east1-b"
}

variable aws_instance_type_login_node {
    default = "t2.micro"
}

variable aws_instance_type_worker_node {
    default = "t2.micro"
}

variable gcp_instance_type_login_node {
    default = "f1-micro"
}

variable gcp_instance_type_worker_node {
    default = "f1-micro"
}

variable login_node_root_size {
    default = "20"
}

variable chmod_command {}

variable key_name {
    default = "first"
}

variable "user_name" {
    default = "ec2-user"
}

variable cluster_name {
    default = "sample-cloud-cluster"
}

variable project_name {
    default = "zyme-cluster"
}