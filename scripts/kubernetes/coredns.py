#!/usr/bin/env python3

"""Tool to configure CoreDNS in the existing Kubernetes cluster."""

import argparse
import hashlib
import logging
import socket
from typing import List

import pydantic
from kubernetes import client, config

logger = logging.getLogger(__name__)


class Host(pydantic.BaseModel):
    """Host record."""

    ip: str
    name: str


Hosts = List[Host]


class CorefileError(Exception):
    """CoreDNS configuration file error."""


def update_corefile(content: str, hosts: Hosts) -> str:
    """Updates CoreDNS configuration file.

    Inserts or updates "hosts { ... }" after "ready".
    """
    lines = []
    ready_line = -1
    hosts_start = -1
    hosts_end = -1
    for line in content.splitlines():
        stripped = line.strip()
        if stripped == 'ready':
            ready_line = len(lines)
        if stripped == 'hosts {':
            hosts_start = len(lines)
        if hosts_start >= 0 and hosts_end == -1 and stripped == '}':
            hosts_end = len(lines)
        lines.append(line)

    if ready_line < 0:
        raise CorefileError('Corefile: failed to find "ready" plugin')

    host_lines = [
        '    hosts {',
        *create_hosts(hosts),
        '       fallthrough',
        '    }',
    ]

    if hosts_start >= 0:
        if hosts_end < 0:
            raise CorefileError('Corefile: failed to parse configuration of "hosts" plugin')
        lines = lines[:hosts_start] + host_lines + lines[hosts_end + 1 :]
    else:
        lines = lines[:ready_line] + host_lines + lines[ready_line:]

    return '\n'.join(lines)


def create_hosts(hosts: Hosts) -> List[str]:
    """Create hosts entries similar to /etc/hosts.

    The form of the entries in the /etc/hosts file are based on IETF RFC 952 which was updated by
    IETF RFC 1123.
    """
    lines = []
    for host in hosts:
        lines.append(f'       {host.ip} {host.name}')
    return lines


def resolve(host: str) -> str:
    """Resolve host to IP address."""
    return socket.gethostbyname(host)


def main(ip: str, domain: str):
    """Main."""
    logger.debug('ip: %s', ip)

    config.load_kube_config()
    apps_v1 = client.AppsV1Api()
    core_v1 = client.CoreV1Api()
    response = core_v1.read_namespaced_config_map(namespace='kube-system', name='coredns')
    corefile = response.data['Corefile']
    logger.debug('Corefile before: %s', corefile)

    services = [
        'ceph',
        'dashboard',
        'dask',
        'grafana',
        'jupyter',
        'minio',
        'prefect',
        'ray',
        's3',
        'app.clearml',
        'api.clearml',
        'files.clearml',
    ]

    hosts = [Host(ip=ip, name=f'{service}.{domain}') for service in services]
    corefile = update_corefile(corefile, hosts)
    corefile_sha256 = hashlib.sha256(corefile.encode('utf-8')).hexdigest()
    logger.debug('Corefile after: %s', corefile)

    core_v1.patch_namespaced_config_map(
        namespace='kube-system',
        name='coredns',
        body={
            'data': {
                'Corefile': corefile,
            },
        },
    )

    response = apps_v1.read_namespaced_deployment(namespace='kube-system', name='coredns')
    annotations = response.spec.template.metadata.annotations
    if annotations is None:
        annotations = {}
    annotations['x1/corefile-sha256'] = corefile_sha256
    apps_v1.patch_namespaced_deployment(
        namespace='kube-system',
        name='coredns',
        body={
            'metadata': response.metadata,
            'spec': {
                'template': {
                    'metadata': {
                        'annotations': annotations,
                    }
                }
            },
        },
    )


if __name__ == '__main__':
    logging.basicConfig(level=logging.INFO)
    parser = argparse.ArgumentParser()
    parser.add_argument('host', help='host or IP address')
    parser.add_argument('domain', help='domain name')
    args = parser.parse_args()
    main(socket.gethostbyname(args.host), args.domain)
