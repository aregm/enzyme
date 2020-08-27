# RHOC - Rapid HPC Orchestration in the Cloud

1. [Introduction](#introduction)
   1. [Overview](#overview)
   2. [Intro to HPC](#Intro-to-the-Intel-HPC-Platform-Specification)
      1. [Motivations for enabling HPC in the cloud](#Motivations-for-enabling-HPC-in-the-cloud)
   3.  [How the parameterization works](#how-the-parameterization-works)
   4. [Supported providers](#Currently-supported-providers)
2.  [How to use RHOC](#How-to-use-RHOC) 
   1. [Preparation steps](#preparation-steps)
      1. [Cloud system](#cloud-system)
      2. [Required software](#Required-software)
   2. [Installing RHOC](#installing-RHOC)
   3. [Use cases](#use-cases)
      1. [Workload launching](#Launching-workloads)
      2. [Persistent cluster](#Persistent-clusters)
      3. [Workload launching with storage](#Launching-workloads-with-storage)
      4. [Destroying](#Destroying-clusters)
      5. [Additional commands and options](#additional-commands-and-options)
   4. [Options and parameters](#options-and-parameters)
   5. [Examples](#examples)
      1. [LINPACK-based](#run-high-performance-linpack-benchmark-on-cloud-cluster)
      2. [LAMMPS-based](#run-lammps-molecular-dynamics-simulator-on-login-node)
      3. [OpenFOAM-based](#run-openfoam-benchmark-on-login-node)

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

## How to use RHOC

### Preparation steps

#### Cloud system

First, you need to have an account with a supported provider:

[Google Cloud Platform](https://accounts.google.com/signup/v2/webcreateaccount?service=cloudconsole&continue=https%3A%2F%2Fconsole.cloud.google.com%2F%3F_ga%3D2.221590619.-23985963.1522764483%26ref%3Dhttps%3A%2F%2Fcloud.google.com%2F&flowName=GlifWebSignIn&flowEntry=SignUp&nogm=true) 

Then you need credentials:

* For [Google Cloud Platform](https://cloud.google.com/iam/docs/creating-managing-service-account-keys#creating_service_account_keys)
* For [Amazon Web Services](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html#Using_CreateAccessKey) 

and put it into any folder (e.g. `user_credentials/gcp/gcp_credentials.json`). 

#### Required software

You need to install:

- [Go](https://golang.org/doc/install) 
- *make*: [for Windows](http://gnuwin32.sourceforge.net/packages/make.htm), [for Linux](https://www.gnu.org/software/make/)

### Installing RHOC

-  Clone RHOC repository from [Github](https://github.com/intel-go/RHOC)

- Build project by issuing *make* in the root project folder

  ```
  make GOOS=windows
  ```

  *GOOS* parameter defines binaries for which platform (`windows`, `linux`) you want to get them. (*default:* `"linux"`)

  You should get a `package-{GOOS}-amd64`  folder with built binaries/executables. Also, the whole package is archived  into `package-{GOOS}-amd64-{version}-{hash}.tar.gz`.

### Use cases

- First, create a configuration file with your [customized parameters](#options-and-parameters). For example, see `examples/linpack/linpack-cluster.json`

#####  Launching workloads

```
RHOC run task.sh --parameters path/to/parameters.json
```

This command will instantiate a cloud-based cluster and run the specified task. On first use, the machine image will be automatically created. After the task completes, the cluster will be destroyed, but the machine image will be left intact for future use.

##### Persistent clusters  

```
rhoc run task.sh --parameters path/to/parameters.json --keep-cluster
```

This command will instantiate the requested cluster and storage for the specified task. The required images will be created on first use. Using `--use-storage` option allows you to access data living on the storage node. **NOTICE**: *make sure you don't change parameters in configuration except `storage_disk_size`, otherwise, a new storage will be created after parameters are changed. Currently changing `storage_disk_size` has no effect and the disk keep its previous size, to force it to resize destroy the storage node and delete the disk in cloud provider interface.*

You can create a persistent cluster without running a task. For this, just use the [create cluster command](#create-cluster).

##### Launching workloads with storage

```
rhoc run task.sh --parameters path/to/parameters.json --use-storage
```

This command will instantiate the requested cluster and storage and then run the specified task. As before, the required images will be created on first use. Using `--use-storage` option allows you to access to storage data. **NOTICE**: *make sure you didn't change parameters in configuration except `storage_disk_size`, otherwise, a new storage will be created after parameters are changed. `storage_disk_size` changing is ignored, disk keep the previous size.*

You can create storage without running a task. For this, just use the [create storage command](#create-storage).

##### Destroying clusters

```
rhoc destroy destroyObjectID
```

You can destroy a cluster or storage by *destroyObjectID*, which can be found by [checking state](#check-states).

**NOTICE**: *The disk is kept when the storage is destroyed. Only the VM instances will be removed, and the "storage" RHOC entity will change its status from XXXX to configured. You can delete a disk manually through a selected provider if you want to.*

##### Additional commands and options

###### Create image

```
rhoc create image --parameters path/to/parameters.json
```

This command tells RHOC to create a VM image from a single configuration file. You can check for created images in cloud provider interface if you want to.

###### Create cluster

```
rhoc create cluster --parameters path/to/parameters.json
```

This command tells RHOC to spawn VM instances and form a cluster. It also creates the needed image if it doesn't yet exist.

###### Create storage

```
rhoc create storage --parameters path/to/parameters.json
```

This command tells RHOC to create VM instance based on a disk that holds your data. You can use storage to organize your data and control access to it. Storage locates in `/storage` folder on VM instance. It also creates the needed image if it doesn't exist yet.

Uploading data into the storage is outside the scope of RHOC. RHOC only provides information allowing you to connect to the storage using `rhoc state` [state command](#check-states).

###### Check states

```
rhoc state
```

This command enumerate all manageable entities (images, clusters, storages etc.) and their respective status. For cluster and storage entities, additional information about SSH/SCP connection (user name, address, and security keys) is provided, in order to facilitate access to these resources.

###### Check version

```
rhoc version 
```

###### Check parameters that user can set

Use this command with one of the additional arguments: *image, cluster, task*.

```
rhoc print-vars image
```

You can use `--provider` flag to check parameters specific for certain provider (*default:* GCP)

###### Help

```
rhoc help
```

This command prints a short help summary. Also, each RHOC command has a `--help` switch for providing command-related help.

###### Verbose info

Use `-v` or `--verbose` flag with any command to get extended info.

###### Simulation

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

###### parameters

A task combines parameters from all entities it might need to create. For individual entities see:

- [Image parameters](#image)
- [Cluster parameters](#cluster)

###### options

- `--keep-cluster` keep the cluster running after script is done
- `--use-storage` allow accessing to storage data
- `--newline-conversion` enable conversion of DOS/Windows newlines to UNIX newlines for the uploaded script (useful if you're running RHOC on Windows)
- `--overwrite` overwrite the content of the remote file with the content of the local file
- `--remote-path` name for the uploaded script on the remote machine (*default:* `"./RHOC-script"`)
- `--upload-files` files for copying into the cluster (into `~/RHOC-upload` folder with the same names)
- `--download-files` files for copying from the cluster (into `./RHOC-download` folder with the same names)

#### Image

###### parameters

- `project_name` (*default:* `"zyme-cluster"`)
- `user_name` user name for ssh access (*default:* `"ec2-user"`)
- `image_name` name of the image of the machine being created (*default:* `"zyme-worker-node"`)
- `disk_size` size of image boot disk, in GB (*default:* `"20"`)

#### Cluster

###### parameters

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

###### parameters

- `project_name` (*default:* `"zyme-cluster"`)
- `user_name` user name for ssh access (*default:* `"ec2-user"`)
- `storage_name` name of the storage being created (*default:* `"zyme-storage"`)
- `image_name` name of the image which will be used (*default:* `"zyme-worker-node"`)
- `storage_disk_size`  size of permanent disk, in GB (*default:* `"50"`)
- `storage_instance_type` machine type of storage node (*default:* `"f1-micro"` for GCP)
- `ssh_key_pair_path` (*default:* `"private_keys"`)
- `storage_key_name` (*default:* `"hello-storage"`)

### Examples

#### Examples preparation steps

Let's create your first own cluster. 

- Take the first two steps from [How to use RHOC](#how-to-use-RHOC) ([1](#preparation-steps) and [2](#installing-RHOC)) if you haven't completed them yet.
- Acquire GCP credentials file and save it as `user_credentials/credentials.json`.

#### Run High-Performance LINPACK benchmark on cloud cluster

- Complete the preparation steps from [Example preparation steps](#examples-preparation-steps)
- Run the LINPACK benchmark:
   ```
   RHOC run examples/linpack/linpack-cluster.sh --parameters examples/linpack/linpack-cluster.json --upload-files examples/linpack/HPL.dat
   ```
- Your end of output should look like this:
   ```
   *Finished        1 tests with the following results:*

                                           *1 tests completed and passed residual checks,*

                                            *0 tests completed and failed residual checks,*

                                           *0 tests skipped because of illegal input values.*

   --------------------------------------------------------------------------------

   *End of Tests.*
   ```
- This is it! You have just successfully ran LINPACK on the cloud.

#### Run LAMMPS Molecular Dynamics Simulator on login node

- Complete the preparation steps from [Example preparation steps](#examples-preparation-steps)
- Create storage:
   ```
   ./RHOC create storage --parameters=examples/lammps/lammps-single-node.json
   ```
- Consult `RHOC state` for connection details to the storage node, SSH into it using provided private key and IP address
- Prepare `/storage/lammps/` folder for upload data:
   ```
   sudo mkdir /storage/lammps/
   chown lammps-user /storage/lammps/
   ```
- Upload `lammps.avx512.simg` container into `/storage/lammps/`, e.g. by `scp -i path/to/private_key.pem path/to/lammps.avx512.simg lammps-user@storage-address:/storage/lammps/`
- Run LAMMPS benchmark:
   ```
   ./RHOC run examples/lammps/lammps-single-node.sh --parameters=examples/lammps/lammps-single-node.json --use-storage --download-files=lammps.log
   ```
- Your content of `RHOC-download/lammps.log` file should look like this (*Note:* this was received by running on 4 cores):
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
- This is it! You have just successfully ran LAMMPS on the cloud.
- Don't forget to destroy storage.

#### Run OpenFOAM Benchmark on login node

- Complete the preparation steps from [Example preparation steps](#examples-preparation-steps)
- Run OpenFOAM benchmark, where *7* is the `endTime` of computing benchmark:
   ```
   ./RHOC run -r us-east1 -z b --parameters examples/openfoam/openfoam-single-node.json --download-files DrivAer/log.simpleFoam --overwrite examples/openfoam/openfoam-single-node.sh 7
   ```
- Full log of running OpenFOAM should be available as `RHOC-download/log.simpleFoam`
- This is it! You have just successfully ran OpenFOAM on the cloud.
