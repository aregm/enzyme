# RHOC
A toolset for cloud-agnostic managing of Virtual Private Cluster over cloud provider infrastructure (e.g. for running HPC workloads)

# How to use *RHOC*

You need to complete all steps to use RHOC successfully on your machine:

## Installing *Zyme*
1. Unzip *Zyme*
2. Take `socks-proxy.txt` from the attached files and put in the `tools` directory.
3. Then set up proxy on your machine. (Based on your network settings, this may or may not apply) Paste to the console (on Windows; on Linux/MacOS use `export` instead of `set`):
```
set http_proxy=http://[your proxy address]
set https_proxy=http://[your proxy address]
```
4. You need [Python 2.7](https://www.python.org/downloads/release/python-2714/) to be installed on your machine.
5. After installation run `pip install virtualenv` in your console.
6. Run  `python bootstrap_tooling.py` script from **tools** folder. It will install all the dependencies and create a virtual environment.
6. For windows make sure to use the Command Prompt, you can type `cmd` in your search bar to bring up the command prompt console.
7. Now you are ready to start using Zyme.

## How to create example cluster with *Zyme*


### Getting and preparing Intel software
**NOTICE**: You need the following distributions: Intel® Cluster Checker(version: 2019 initial release) and Intel® Parallel Studio XE Cluster Edition for Fortran and C++ Linux*(version: 2018 update 3) at http://softwareproducts.intel.com/ILC/. 

1. Get them with the licenses, you can find the license file ending with .lic in the directions you get from the email after you have requested the cluster checker and Parallel Studio XE Cluster Edition and put in the `distrib/psxe_cluster_edition` folder.
1. Put the Cluster Checker distribution into the `distrib/clck` folder.
1. Put the PSXE Cluster Edition distribution with license into the `distrib/psxe_cluster_edition` folder.
1. The names of distributions should end with `.tgz`. If they do not, rename them.

### Configuring the cluster
First of all, you need to register with the supported provider. You can use Enzyme with GCP, and Azure. Get the credentials file from your CSP, and put it into the provider's folder(e.g. `enzyme/providers/gcp`). <br> 
**NOTE:** The name file must be `credentials`.
1. Create the `user_defined.json` file
1. Сonfigure the following settings:
   * `provider` - select from the [available providers] See all available parameters documented below.
   * `worker_count` - number of computational nodes (>1).

All available parameters for configurations: 

•	provider - (Required) available providers (aws - Amazon Web Services; gcp - Google Cloud Platform)
•	worker_count - (Required) number of computational nodes (>1)
•	image_name - the name of the image of the machine being created (default: "zyme-worker-node")
•	aws_region - location of your cluster for aws provider (default: "us-east-1")
•	gcp_region - location of your cluster for gcp provider (default: "us-east1")
•	gcp_zone - location of your cluster for gcp provider (default: "us-east1-b")
•	aws_instance_type_login_node - (default: "t2.micro")
•	aws_instance_type_worker_node - (default: "t2.micro")
•	gcp_instance_type_login_node - (default: "f1-micro")
•	gcp_instance_type_worker_node - (default: "f1-micro")
•	login_node_root_size - (default: "20")
•	chmod_command - assignment of rights to the file in which the generated ssh key is located (generated automatically)
•	key_name - the name of the generated key-pair (default: "first")
•	user_name - name for ssh access (default: "ec2-user")
•	cluster_name - (default: "sample_cloud_cluster")
•	project_name - (default: "zyme-cluster")


**NOTE**: Also, you can put scripts into the `distrib/your-scripts` folder that will be executed during the provisioning

### Setting up the cluster
1. Run `zyme pk build cluster-node.json` to prepare custom image (with Cluster Checker and its prerequisites installed)
1. Run `zyme tf init` to install Terraform requirements
1. Run `zyme tf plan` to see your initial configuration
1. Run `zyme tf apply` to deploy your cluster.

**NOTE**: Use `zyme tf destroy` to destroy your cluster.

Now you can login to the master node via ssh (login as: value of variable `user_name`)<br>
SSH private key can be found in `secrets` directory (**note**: if you're using PuTTY please convert `.pem` key to `.ppk` format via `PuTTYGen` tool that comes with PuTTY)

## Additionally

### How to get the ssf compatibility report from Cluster Checker
**Notice**: If you have just completed `zyme tf apply` then wait a couple of minutes. 

Run:
1. `~/clck/cluster_checker.sh` – this collects Cluster Checker data, analyzes Cluster Checker data and gives ssf compatibility report(`~/clck/clck_results.log`)

At the moment, the following checks don't pass:
* memory-minimum-required...(you must choose more powerful instance type for computes nodes)
* storage-ssf-compute(you must increase storage capacity to 80 GB and more)
* storage-ssf-head(you must increase storage capacity to 200 GB and more; for this - increase `login_node_root_size` variable)

**NOTE**: In this state, the cluster is operational

### Features
* We also install Intel® Cluster Checker to make sure that this configuration is viable for using. You may run it manually from your master node.
* Installing Zyme from `sdvis` branch you get pre-installed [Intel Rendering Framework(SDVis)](http://sdvis.org/)
