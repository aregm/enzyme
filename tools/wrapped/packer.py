from __future__ import absolute_import

import os
import subprocess

import helpers
try:
    from make_proxy_socket import read_socks_proxy
except ImportError:
    def read_socks_proxy():
        return None, None

from .hashicorp import HashicorpWrappedCommand


class PackerWrapper(HashicorpWrappedCommand):
    ENV_NAME = 'ZYME_PACKER_PATH'
    BINARY_NAME = 'packer'
    SEARCH_LEVELS = 3
    RELEASES_PAGE = 'https://releases.hashicorp.com/packer/'

    # packer 1.2.1 has a bug on AWS credentials,
    # see https://github.com/hashicorp/packer/issues/5986
    BAD_RELEASES = ['1.2.1-1.2.1']

    def __call__(self, packer_args, current_builder, no_socks_proxy=False):
        # In case the user enters their own template
        possible_template = packer_args.pop()
        check_template = possible_template + '.json'

        if possible_template.startswith('-') or \
                os.path.exists(possible_template):
            packer_args.append(possible_template)

        elif os.path.exists(check_template):
            packer_args.append(check_template)
        else:
            packer_args.append(possible_template)

        cli = [self.target]

        cli.append(packer_args[0])
        if packer_args[0] in ('build', 'validate', 'push'):
            cli.extend(['-var-file=' + os.path.abspath('variables.json')])
            if packer_args[0] == 'build':
                cli.extend(['-only=%s' % (current_builder)])
                socks_host, socks_port = (None, None) if no_socks_proxy \
                    else read_socks_proxy()

                if socks_host:
                    cli.extend(['-var', 'ssh_socks_proxy_host=%s' %
                               socks_host])
                    if socks_port:
                        cli.extend(['-var', 'ssh_socks_proxy_port=%d' %
                                   socks_port])

        cli.extend(packer_args[1:])
        helpers.verbose_print('Packer command: %s' %
                              subprocess.list2cmdline(cli))
        subprocess.call(cli)
