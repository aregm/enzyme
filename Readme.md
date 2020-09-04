# Rapid HPC Orchestration in the Cloud (RHOC)

1. [Introduction](#introduction)
   1. [Overview](#overview)
   2. [Motivations for enabling HPC in the cloud](#Motivations-for-enabling-HPC-in-the-cloud)
   3. [Intro to the Intel HPC Platform Specification](#Intro-to-the-Intel-HPC-Platform-Specification)
   4. [Currently Supported providers](#Currently-supported-providers)
2. [Installing RHOC](#installing-RHOC)
   1. [Required software](#Required-software)
   2. [Cloan the RHOC respository](#clone-the-RHOC-repository)
   3. [Build RHOC](#build-RHOC)
3. [Getting Started with RHOC](#Getting-Started-with-RHOC)
   1. [User Credentials File](#User-Credentials-File)
   2. [Cloud Provider Templates](#cloud-provider-templates)
   3. [Test Run](#test-run)   
4. [RHOC User Guide](#RHOC-user-guide)
      1. [Launching workloads](#Launching-workloads)
      2. [Persistent clusters](#Persistent-clusters)
      3. [Launching Workloads with storage](#Launching-workloads-with-storage)
      4. [Destroying Clusters](#Destroying-clusters)
      5. [Create image](#create-image)
      6. [Create storage](#create-storage)
      7. [Check Status](#check-status)
      8. [Check version](#check-version)
      9. [Check user defined parameters](#check-user-defined-parameters)
      10. [Help](#help)
      11. [Set Verbosity](#set-verbosity)
      12. [Simulate](#simulate)
      13. [Options and parameters](#options-and-parameters)
5. [Additional Examples](#additional-examples)
      1. [LAMMPS](#lammps)
      2. [OpenFOAM](#openfoam)
6. [Cloud Provider Quick Reference](#cloud-provider-quick-reference)
   1. [Amazon Web Services](#amazon-web-services)
   2. [Google Cloud Platform](#google-cloud-platform)

## Introduction

* This is a relatively new project and should be considered Alpha level software

### Overview

RHOC is a software tool that helps provide an acclerated path to spinning up and utilizing high performance compute clusters in the cloud. RHOC provides a simple command line mechanism for users to launch workloads pointing to templates that abstract the orchestration and Operating System image generation of the HPC cluster. It creates operating system images that follow the Intel® HPC Platform Specification to provide a standard base solution that enables a wide range of popular HPC applications. RHOC aims to accelerate the path for users wanting to migrate to a public cloud by abstracting the learning curve of a supported cloud provider. This allows users a "rapid" path to start using cloud resources, and it allows the RHOC community to collaborate to provide optimal envinronments for the underlying HPC solutions. 

### Motivations for enabling HPC in the cloud

There are many reasons for running HPC and compute intensive workloads in a cloud environment. The following are some of the top motivators behind RHOC, but the list is not exhaustive.
- Local HPC cluster resource capacity is typically fixed while demand is variable. Cloud resources provide augmentation to local resources that help meet spikes in resource needs on demand.
- Cloud-based HPC clusters can simplify and accelerate access for new HPC users and new businesses, resulting in faster time to results. 
- Cloud provides a means to access massive resources or specialized resource for short periods, to address temporary or intermittent business needs.
- Cloud provides access to the newest technologies, allowing evaluation and use ahead of long-term ownership
- Datasets may already exist in the cloud, and utilizing cloud resouces may be the best option for performance and/or cost. 

### Intro to the Intel® HPC Platform Specification

The Intel HPC Platform Specification captures industry best practices, optimized Intel runtime requirements, and broad application compatability needs. These requirements form a foundation for high performance computing solutions to provide enhanced compatibility and performance across a range of popular HPC workloads. Intel developed this specification by collaborating with many industry partners, incorporating feedback, and curating the specification since 2007.


### Currently supported providers

- [Google Cloud Platform](https://cloud.google.com/)
- [Amazon Web Services](https://aws.amazon.com/)

###### Planned supported providers in the nearest future

- [Microsoft Azure](https://azure.microsoft.com/en-us/)

## Installing RHOC

### Required software

You need to install:

- [Go](https://golang.org/doc/install) 
- *make*: [for Windows](http://gnuwin32.sourceforge.net/packages/make.htm), [for Linux](https://www.gnu.org/software/make/)

### Clone the RHOC repository

Clone RHOC repository from [Github](https://github.com/intel-go/RHOC)

RHOC uses open source tools from [Hashicorp](https://www.hashicorp.com), and those tools are included as sub-modules in the RHOC git repository. To ensure all the required source is cloned, it is suggested to use the following:

  ```
  git clone --recurse-submodules https://github.com/intel-go/RHOC
  ```

If needed, the sub-modules can be downloaded after cloning by using 

  ```
  git submodule --init
  git submodule update 
  ```

Note: some firewall configurations can impact access to git repositories. 

### Build RHOC

RHOC uses *make* to build the binary from go source code. Build RHOC by specifying the make command and optionally including the target OS platform using the `GOOS` command line option. Options are currently are `windows` or `linux`. If no OS is specificed, the *default* build assumes `linux`


Note: the *make* file does not currently support building for Windows under a Windows cmd shell. To build to run RHOC from a Windows platform, use a Windows Bash implementation and run the following make command:
  ```
  make GOOS=windows
  ```

If the build completes successfully, the RHOC build will create a sub-directory called `package-{GOOS}-amd64` that includes the binaries and supporting directory structures for executing RHOC. In addition, the sub-directory package is archived into `package-{GOOS}-amd64-{version}-{hash}.tar.gz` for easy distribution.

The binary package name for Linux is `Rhoc`, and the binary package name for Windows is `Rhoc.exe`. The command line examples in this guide all use the Linux binary name. For use from a Windows system, substitute the `Rhoc` command with `Rhoc.exe`.

## Getting Started with RHOC

RHOC takes a number of input parameters that provide user credentials for a target cloud account, templates for the cloud provider, and templates for the desired image to run on top of in the cloud. These JSON inputs are combined into a single structure to drive the Hashicorp tools, terraform and packer, to create machine images and the spin up the cluster.  The combined structure is saved in the `.RHOC/` of the RHOC package directory.

### User Credentials File

RHOC requires an active account for the desired cloud provider. Access to that user account is utilized by providing access keys and account information in a credentials JSON file. Cloud providers typically offer mechanisms to create this credentials file. See the appropriate [Cloud Provider Quick Reference](#cloud-provider-quick-reference) section for referencing how to create a user credentials file for a specific provider. Please note, that these provider specific mechanisms may change.

The user credentials file needs to be copied to the user's host system where RHOC will execute. To use RHOC without specifying a full path to the desired user credentials file, copy the cloud provider crendtials file to `./user_credentials/credentials.json` in the RHOC binary directory. RHOC uses this file as the default to access the desired cloud provider account. RHOC does provide a command line option to use a different path and filename for credentials if desired. For example, a user may have more than one accounts and thus have multiple user credentials files that are specified by the command line option with each run.  

### Cloud Provider Templates

RHOC uses template files to direct how to build a cluster and how to build the compute node operating system to run workloads on the desired type of instance within the desired cloud provider. These templates are JSON files that provide variables that control how RHOC uses the Hashicorp tools. These templates may be curated and expanded to provide additional user options and customizations.

Cloud provider templates are provided under the `.\templates` directory and are typically named after the cloud service provider. The templates `cluster_template.json` and `image_template.json` under a given cloud provider directory control instance and image creation, respectively.

For example, a hypothetical cloud provider called MyCloud would have:
`./templates/mycloud/cluster_template.json`
`./templates/mycloud/image_template.json`

A user specifies which cloud provider templates to use with the`-p` or `--provider` command line parameter. RHOC currently defaults to using the Google Cloud Provider templates. To use the hypothetical MyCloud providers then, a user includes `-p mycloud` or `--provider mycloud` on the command line.

### Test Run

Now let's execute a real job as a "Hello, World" test that RHOC is working. To do this, we'll use the well-known High-Performance LINPACK benchmark that is included in the `./examples` folder. This example will use the default cloud provider. To test a different or multiple cloud providers, insert the `-p` option with the name of the directory of the desired provider in the example below. 

To execute a workload through RHOC, the user specifies the job to launch (or typically a launch script) and points to a RHOC parameter file and any potential input data files. The RHOC parameter file fills out and replaces default values used in execution. This allows a user to modify some aspects of execution without needing to modify the cloud provider or image templates themselves. An important parameter is the project name associated with the user account. This must be set correctly in the project parameter file.

With this in mind, three steps are all that are requried to test execution using HP-LINPACK.

1. Copy the user credentials file to `./user_credentials/credentials.json`. This is the default credentials file RHOC uses.

2. Modify the `./examples/linpack/linpack-cluster.json` file to set the `project_name` value to the actual name of the cloud project. For example, if the cloud project name is My-Hpc-Cloud-Cluster, modify the key-value pair in the JSON file to be
   ```
   project_name: "My-Hpc-Cloud-Cluster",
   ```

3. Execute the command to run HP-LINPACK through RHOC on the default cloud provider. The following commmand uses both the default cloud provider as well as the default user credentials file (from Step 1).
   ```
   Rhoc run examples/linpack/linpack-cluster.sh --parameters examples/linpack/linpack-cluster.json --upload-files examples/linpack/HPL.dat
   ```

RHOC will begin building a compute node operating system and installing on the desired instance types in the cloud provider. If that is successful, RHOC will launch HP-LINPACK on the cluster. RHOC reports progress along the way, so there should be periodic output displayed on console. 

If the end of output should looks like this:
   ```
   *Finished        1 tests with the following results:*

                                           *1 tests completed and passed residual checks,*

                                            *0 tests completed and failed residual checks,*

                                           *0 tests skipped because of illegal input values.*

   --------------------------------------------------------------------------------

   *End of Tests.*
   ```
then HP-LINPACK successfully executed in the cloud. Congratulations!

Unfortunately, if there is an issue, RHOC does not have a well-documented debug section yet. That is a work in progress! Stay tuned.
Troubleshooting areas to check:
- TerraForm and Packer executables exist under the `./tools` directory. If not, there was a problem building those tools during the RHOC build.

- A cluster does not appear in the cloud provider dashboard while running RHOC. Potential problems could be a problem with the user account permissions, incorrect user credentials file, or incorrect project name identified in the `./examples/linpack/linpack-cluster.json` file.

## RHOC User Guide

###  Launching workloads

```
Rhoc run task.sh --parameters path/to/parameters.json
```

This command will instantiate a cloud-based cluster and run the specified task. On first use, the machine image will be automatically created. After the task completes, the cluster will be destroyed, but the machine image will be left intact for future use.

### Persistent clusters  

```
Rhoc run task.sh --parameters path/to/parameters.json --keep-cluster
```

This command will instantiate the requested cluster and storage for the specified task. The required images will be created on first use. Using `--use-storage` option allows you to access data living on the storage node. **NOTICE**: *make sure you don't change parameters in configuration except `storage_disk_size`, otherwise, a new storage will be created after parameters are changed. Currently changing `storage_disk_size` has no effect and the disk keep its previous size, to force it to resize destroy the storage node and delete the disk in cloud provider interface.*

You can create a persistent cluster without running a task. For this, just use the [create cluster command](#create-cluster).

### Launching workloads with storage

```
Rhoc run task.sh --parameters path/to/parameters.json --use-storage
```

This command will instantiate the requested cluster and storage and then run the specified task. As before, the required images will be created on first use. Using `--use-storage` option allows you to access to storage data. **NOTICE**: *make sure you didn't change parameters in configuration except `storage_disk_size`, otherwise, a new storage will be created after parameters are changed. `storage_disk_size` changing is ignored, disk keep the previous size.*

You can create storage without running a task. For this, just use the [create storage command](#create-storage).

### Destroying clusters

```
Rhoc destroy destroyObjectID
```

You can destroy a cluster or storage by *destroyObjectID*, which can be found by [checking state](#check-states).

**NOTICE**: *The disk is kept when the storage is destroyed. Only the VM instances will be removed, and the "storage" RHOC entity will change its status from XXXX to configured. You can delete a disk manually through a selected provider if you want to.*

### Create image

```
Rhoc create image --parameters path/to/parameters.json
```

This command tells RHOC to create a VM image from a single configuration file. You can check for created images in cloud provider interface if you want to.

### Create cluster

```
Rhoc create cluster --parameters path/to/parameters.json
```

This command tells RHOC to spawn VM instances and form a cluster. It also creates the needed image if it doesn't yet exist.

### Create storage

```
Rhoc create storage --parameters path/to/parameters.json
```

This command tells RHOC to create VM instance based on a disk that holds your data. You can use storage to organize your data and control access to it. Storage locates in `/storage` folder on VM instance. It also creates the needed image if it doesn't exist yet.

Uploading data into the storage is outside the scope of RHOC. RHOC only provides information allowing you to connect to the storage using `rhoc state` [state command](#check-states).

### Check status

```
Rhoc state
```

This command enumerate all manageable entities (images, clusters, storages etc.) and their respective status. For cluster and storage entities, additional information about SSH/SCP connection (user name, address, and security keys) is provided, in order to facilitate access to these resources.

### Check version

```
Rhoc version 
```

### Check user defined parameters

Use this command with one of the additional arguments: *image, cluster, task*.

```
Rhoc print-vars image
```

You can use `--provider` flag to check parameters specific for certain provider (*default:* GCP)

### Help

```
Rhoc help
```

This command prints a short help summary. Also, each RHOC command has a `--help` switch for providing command-related help.

### Set Verbosity

Use `-v` or `--verbose` flag with any command to get extended info.

### Simulate

Use `-s` or `--simulate` flag with any command to simulate running the execution without actually running any commands that can modify anything in the cloud or locally. Useful for checking what RHOC would perform without actually performing it.

### Options and parameters

#### Common parameters

- `-p, --provider` select provider (default: `gcp`)
   `gcp` - Google Cloud Platform
   `aws` - Amazon Web Services

- `-c, --credentials` path to credentials file (default: `user_credentials/credentials.json`)

- `-r, --region` location of your cluster for selected provider (default: `us-central1`)

- `-z, --zone` location of your cluster for selected provider (default: `a`)

- `--parameters` path to file with user parameters

You can define the above parameters only via command line.



Parameters presented below can be used in configuration file and command line. When specified in command line they override parameters from configuration file.

For applying them by command line use

-  `--vars` list of user's variables (*example:* `"image_name=RHOC,disk_size=30"`)

#### Task 

##### parameters

A task combines parameters from all entities it might need to create. For individual entities see:

- [Image parameters](#image)
- [Cluster parameters](#cluster)

##### options

- `--keep-cluster` keep the cluster running after script is done
- `--use-storage` allow accessing to storage data
- `--newline-conversion` enable conversion of DOS/Windows newlines to UNIX newlines for the uploaded script (useful if you're running RHOC on Windows)
- `--overwrite` overwrite the content of the remote file with the content of the local file
- `--remote-path` name for the uploaded script on the remote machine (*default:* `"./RHOC-script"`)
- `--upload-files` files for copying into the cluster (into `~/RHOC-upload` folder with the same names)
- `--download-files` files for copying from the cluster (into `./RHOC-download` folder with the same names)

#### Image

##### parameters

- `project_name` (*default:* `"zyme-cluster"`)
- `user_name` user name for ssh access (*default:* `"ec2-user"`)
- `image_name` name of the image of the machine being created (*default:* `"zyme-worker-node"`)
- `disk_size` size of image boot disk, in GB (*default:* `"20"`)

#### Cluster

##### parameters

- `project_name` (*default:* `"zyme-cluster"`) 

- `user_name` user name for ssh access (*default:* `"ec2-user"`)

- `cluster_name` name of the cluster being created (*default:* `"sample-cloud-cluster"`)

- `image_name` name of the image which will be used (*default:* `"zyme-worker-node"`)

- `worker_count` count of worker nodes (*default:* `"2"`)

         **NOTICE**: *Must be greater than 1*

-  `login_node_root_size` boot disk size for login node, in GB (*default:* `"20"`)

         **NOTICE**: *Must be no less than `disk_size`*

- `instance_type_login_node` machine type of root node (*default:* `"f1-micro"` for GCP)

- `instance_type_worker_node` machine type of worker nodes (*default:* `"f1-micro"` for GCP)

-  `ssh_key_pair_path` (*default:* `"private_keys"`)

-  `key_name` (*default:* `"hello"`)

#### Storage

##### parameters

- `project_name` (*default:* `"zyme-cluster"`)
- `user_name` user name for ssh access (*default:* `"ec2-user"`)
- `storage_name` name of the storage being created (*default:* `"zyme-storage"`)
- `image_name` name of the image which will be used (*default:* `"zyme-worker-node"`)
- `storage_disk_size`  size of permanent disk, in GB (*default:* `"50"`)
- `storage_instance_type` machine type of storage node (*default:* `"f1-micro"` for GCP)
- `ssh_key_pair_path` (*default:* `"private_keys"`)
- `storage_key_name` (*default:* `"hello-storage"`)

## Additional Examples
The included examples in this section all assume correct build of RHOC and correct set up of user credentials. The examples will use the default cloud provider and the default user credentials file.

### LAMMPS 
LAMMPS is a molecular dynamics simulation application. The included workload will launch a container to execute LAMMPS on a single compute node. This requires use the the `storage` capabilities of RHOC.

1. Create storage for the LAMPPS workload
   ```
   ./Rhoc create storage --parameters=examples/lammps/lammps-single-node.json
   ```
2. Use information from `./RHOC state` to get connection details to the storage node created in step 1. SSH into the storage nodeusing provided private key and IP address and execute the following commands:
   ```
   sudo mkdir /storage/lammps/
   chown lammps-user /storage/lammps/
   ```
   Then log out of the storage node.

3. Upload `lammps.avx512.simg` container into `/storage/lammps/`, e.g. by `scp -i path/to/private_key.pem path/to/lammps.avx512.simg lammps-user@storage-address:/storage/lammps/`

4. Execute the LAMMPS benchmark through RHOC
   ```
   ./Rhoc run examples/lammps/lammps-single-node.sh --parameters=examples/lammps/lammps-single-node.json --use-storage --download-files=lammps.log
   ```
If successful, the content of `RHOC-download/lammps.log` file should look like this (*Note:* this was received by running on 4 cores):
   ```
   args: 2
   OMP_NUM_THREADS=1
   NUMCORES=4
   mpiexec.hydra -np 4 ./lmp_intel_cpu_intelmpi -in WORKLOAD -log none -pk intel 0 omp 1 -sf intel -v m 0.2 -screen
   Running: airebo Performance: 1.208 timesteps/sec
   Running: dpd Performance: 9.963 timesteps/sec
   Running: eam Performance: 9.378 timesteps/sec
   Running: lc Performance: 1.678 timesteps/sec
   Running: lj Performance: 19.073 timesteps/sec
   Running: rhodo Performance: 1.559 timesteps/sec
   Running: sw Performance: 14.928 timesteps/sec
   Running: tersoff Performance: 7.026 timesteps/sec
   Running: water Performance: 7.432 timesteps/sec
   Output file lammps-cluster-login_lammps_2019_11_17.results and all the logs for each workload lammps-cluster-login_lammps_2019_11_17 ... are located at /home/lammps-user/lammps
   ```
5. *Important* Destroy storage using the `./RHOC destroy` command with the storage ID to avoid unintended storage fees with the cloud provider.

### OpenFOAM 

OpenFOAM is a computation fluid dynamics application.

1. Run OpenFOAM benchmark, where *7* is the `endTime` of computing benchmark:
   ```
   ./Rhoc run -r us-east1 -z b --parameters examples/openfoam/openfoam-single-node.json --download-files DrivAer/log.simpleFoam --overwrite examples/openfoam/openfoam-single-node.sh 7
   ```
Full log of running OpenFOAM should be available as `RHOC-download/log.simpleFoam`

## Cloud Provider Quick Reference
This section is intended to provide easy references to cloud providers relative to RHOC setup.

### Amazon Web Services

[Help generating the user credentials for Amazon Web Services](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html#Using_CreateAccessKey) 

### Google Cloud Platform 

[Google Cloud Platform](https://accounts.google.com/signup/v2/webcreateaccount?service=cloudconsole&continue=https%3A%2F%2Fconsole.cloud.google.com%2F%3F_ga%3D2.221590619.-23985963.1522764483%26ref%3Dhttps%3A%2F%2Fcloud.google.com%2F&flowName=GlifWebSignIn&flowEntry=SignUp&nogm=true) Account Information

[Help generating the user credentials for Google Cloud Platform](https://cloud.google.com/iam/docs/creating-managing-service-account-keys#creating_service_account_keys)

