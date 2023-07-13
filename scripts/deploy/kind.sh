#!/usr/bin/env bash

# Deploys X1 cluster using kind, see https://kind.sigs.k8s.io/

set -e

# Default values that can be overriden by corresponding environment variables
: ${KIND_VERSION:="v0.19.0"}
: ${CLUSTER_NAME:="x1"}
: ${REDSOCKS_SKIP:="0.0.0.0/8 10.0.0.0/8 100.64.0.0/10 127.0.0.0/8 169.254.0.0/16 172.16.0.0/12 192.168.0.0/16 198.18.0.0/15 224.0.0.0/4 240.0.0.0/4"}
: ${INGRESS_DOMAIN:="localtest.me"}
: ${CONTROL_NODE_IMAGE:=pbchekin/ccn:0.0.1}

# https://stackoverflow.com/questions/59895/getting-the-source-directory-of-a-bash-script-from-within
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

source "$SCRIPT_DIR/functions.sh"

# Install tools, such as kind, to ~/bin/
# TODO: also support ~/.local/bin
export PATH="$HOME/bin:$PATH"

function install_kind() {
  if ! is_installed curl; then
    exit 1
  fi
  mkdir -p "$HOME/bin"
  curl -sSL -o "$HOME/bin/kind" "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-linux-amd64"
  chmod a+x "$HOME/bin/kind"
}

if ! is_installed docker; then
  echo "See https://docs.docker.com/engine/install/"
  exit 1
fi

if ! is_installed kind; then
  echo "See https://kind.sigs.k8s.io/docs/user/quick-start#installation"
  echo "Will attempt to install kind to $HOME/bin"
  install_kind
  if ! is_installed kind; then
    exit 1
  fi
else
  kind_version="$(kind --version)"
  if [[ "$kind_version" =~ ([0-9]+\.[0-9]+\.[0-9]+) ]]; then
    actual_kind_version="${BASH_REMATCH[1]}"
    if [[ "$KIND_VERSION" =~ ([0-9]+\.[0-9]+\.[0-9]+) ]]; then
      desired_kind_version="${BASH_REMATCH[1]}"
      if [[ $actual_kind_version != $desired_kind_version ]]; then
        # TODO: check if actual kind version is newer than desired
        warn "Kind version: $actual_kind_version, required: $desired_kind_version, will attempt to install kind to $HOME/bin"
        install_kind
      else
        pass "Kind version: $actual_kind_version"
      fi
    fi
  else
    fail "Failed to parse kind version: $kind_version"
    exit 1
  fi
fi

# TODO: make ports 80 and 443 configurable on host
function create_kind_cluster() {
  kind_config="\
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    image: kindest/node:v1.25.9
    # This works only for one node, see https://kind.sigs.k8s.io/docs/user/ingress/#ingress-nginx
    # With multiple nodes, a more granular control is needed where nginx pod is running.
    extraPortMappings:
      - containerPort: 80
        hostPort: 80
        protocol: TCP
      - containerPort: 443
        hostPort: 443
        protocol: TCP
      # Map Ray client port 10001 to the container port (see nodePort configuration for Ray).
      # Since it is a default Ray port you may need to change it if you have Ray running on the host.
      - containerPort: 30009
        hostPort: 10001
        protocol: TCP
"
  if [[ -v dockerhub_proxy ]]; then
    pass "DockerHub proxy: ${dockerhub_proxy}"
    kind_config="\
$kind_config
containerdConfigPatches:
  - |-
    [plugins.\"io.containerd.grpc.v1.cri\".registry]
    [plugins.\"io.containerd.grpc.v1.cri\".registry.mirrors]
    [plugins.\"io.containerd.grpc.v1.cri\".registry.mirrors.\"docker.io\"]
    endpoint = [\"${dockerhub_proxy}\"]
    [plugins.\"io.containerd.grpc.v1.cri\".registry.configs.\"${dockerhub_proxy}\".tls]
      insecure_skip_verify = true
"
  fi
  kind create cluster --name $CLUSTER_NAME --config=- <<< "$kind_config"
}

# execute command on kind cluster node
function cluster_node() {
  local docker_cmd=( )

  if [[ -t 0 ]]; then
    docker_cmd+=( --interactive )
  fi
  if [[ -t 1 ]]; then
    docker_cmd+=( --tty )
  fi

  docker_cmd+=( "$CLUSTER_NAME-control-plane" )

  if (( $# != 0 )); then
    docker_cmd+=( /bin/bash -c "$@" )
  fi

  docker exec "${docker_cmd[@]}"
}


function pull_images() {
  cat "$X1_ROOT/scripts/etc/kind/images.txt" | xargs -P4 -n1 docker pull -q
}

function load_images() {
  cat "$X1_ROOT/scripts/etc/kind/images.txt" | xargs -P4 -n1 kind --name $CLUSTER_NAME load docker-image
}

function with_proxy() {
  local proxy_url=""
  if [[ -v https_proxy ]]; then
    pass "Using https_proxy: $https_proxy"
    proxy_url="$https_proxy"
  elif [[ -v http_proxy ]]; then
    pass "Using http_proxy: $http_proxy"
    proxy_url="$http_proxy"
  else
    fail "http_proxy or https_proxy must be set"
    exit 1
  fi

  cluster_node "export DEBIAN_FRONTEND=noninteractive; apt-get update -y; apt-get install -y --no-install-recommends redsocks"

  if [[ $proxy_url =~ (https?:\/\/)?([^:]+):([^:]+) ]]; then
    proxy_host="${BASH_REMATCH[2]}"
    proxy_port="${BASH_REMATCH[3]}"
    pass "proxy_host: $proxy_host, proxy_port: $proxy_port"
  else
    fail "Unable to parse proxy URL $proxy_url"
  fi

  redsocks_config="\
base {
    log_debug = off;
    log_info = on;
    log = \"syslog:daemon\";
    daemon = on;
    redirector = iptables;
}

redsocks {
    local_ip = 0.0.0.0;
    local_port = 12345;
    ip = $proxy_host;
    port = $proxy_port;
    type = http-connect;
}
"
  cluster_node "echo '$redsocks_config' > /etc/redsocks.conf"
  cluster_node "/etc/init.d/redsocks restart"

  cluster_node "iptables -w 60 -t nat -N REDSOCKS"

  for nw in $REDSOCKS_SKIP; do
    cluster_node "iptables -w 60 -t nat -A REDSOCKS -d $nw -j RETURN"
  done

  cluster_node "iptables -w 60 -t nat -A REDSOCKS -p tcp --dport 80 -j REDIRECT --to-ports 12345"
  cluster_node "iptables -w 60 -t nat -A REDSOCKS -p tcp --dport 443 -j REDIRECT --to-ports 12345"

  cluster_node "iptables -w 60 -t nat -A PREROUTING -p tcp --dport 80 -j REDSOCKS"
  cluster_node "iptables -w 60 -t nat -A PREROUTING -p tcp --dport 443 -j REDSOCKS"
}

# Update CoreDNS configuration file to resolve external endpoints in cluster correctly
function with_corefile() {
  CONTROl_PLANE_IP=$(docker inspect --format '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$CLUSTER_NAME-control-plane")
  pass "Cluster IP: $CONTROl_PLANE_IP"
  control_node "python -m scripts.kubernetes.coredns $CONTROl_PLANE_IP $INGRESS_DOMAIN"
}

if [[ " $@ " =~ " --help " ]]; then
  cat <<EOF
Usage: $(basename $0) [option]

Options:
  --help              Show this help
  --console           Start control node console
  --check             Run a quick check for Docker and network connectivity
  --list-images       List images in the existing $CLUSTER_NAME cluster to kind/images.txt
  --pull-images       Pull images listed in kind/images.txt
  --load-images       Load images listed in kind/images.txt to the existing $CLUSTER_NAME cluster
  --with-images       Create a cluster and load images (requires '--pull-images' first)
  --with-clearml      Deploy a cluster with ClearML
  --with-dask         Deploy a cluster with Dask
  --with-cert-manager Deploy a cluster with cert-manager
  --delete            Delete cluster $CLUSTER_NAME
EOF
  exit 0
fi

if [[ " $@ " =~ " --console " ]]; then
  control_node
  exit 0
fi

if [[ " $@ " =~ " --check " ]]; then
  control_node "curl https://ipinfo.io/"
  exit 0
fi

if [[ " $@ " =~ " --list-images " ]]; then
  kubectl get pods --all-namespaces -o json | jq -r '.items[].spec.containers[].image' | sort | uniq | tee "$X1_ROOT/scripts/etc/kind/images.txt"
  exit 0
fi

if [[ " $@ " =~ " --pull-images " ]]; then
  pull_images
  exit 0
fi

if [[ " $@ " =~ " --load-images " ]]; then
  load_images
  exit 0
fi

if [[ " $@ " =~ " --delete " ]]; then
  kind delete cluster --name $CLUSTER_NAME
  exit 0
fi

if [[ " $@ " =~ " --with-proxy " ]]; then
  with_proxy
  exit 0
fi

if [[ " $@ " =~ " --with-corefile " ]]; then
  with_corefile
  exit 0
fi

if kind get clusters | grep -qE "^${CLUSTER_NAME}\$" &> /dev/null; then
  pass "Cluster $CLUSTER_NAME is up"
else
  pass "Cluster $CLUSTER_NAME is not up, will attempt to create a new cluster"
  create_kind_cluster
  if [[ " $@ " =~ " --with-images " ]]; then
    load_images
  fi
  if [[ -v http_proxy || -v https_proxy ]]; then
    with_proxy
  fi
  # Clean up Terraform state files from the previous run
  rm -f $X1_ROOT/terraform/terraform.tfstate*
fi

terraform_extra_args=(
  -var local_path_enabled=false         # Kind cluster has local-path-provisioner, another provisioner is not required
  -var default_storage_class="standard" # Kind cluster has local-path-provisioner, it defines "standard" StorageClass
  -var prometheus_enabled=false         # Disable prometheus stack to make footprint smaller
  -var ingress_domain="$INGRESS_DOMAIN"
)

if [[ " $@ " =~ " --with-clearml " ]]; then
  terraform_extra_args+=( -var clearml_enabled=true )
fi

if [[ " $@ " =~ " --with-dask " ]]; then
  terraform_extra_args+=( -var dask_enabled=true )
fi

if [[ " $@ " =~ " --with-cert-manager " ]]; then
  terraform_extra_args+=( -var cert_manager_enabled=true )
fi

with_corefile
control_node "terraform -chdir=x1/terraform init -upgrade -input=false"
control_node "terraform -chdir=x1/terraform apply -input=false -auto-approve ${terraform_extra_args[*]}"

echo "To delete the cluster run '$0 --delete'"
