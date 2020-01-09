from __future__ import absolute_import

import subprocess
import os

import helpers

from .hashicorp import HashicorpWrappedCommand


class TerraformWrapper(HashicorpWrappedCommand):
    ENV_NAME = 'ZYME_TERRAFORM_PATH'
    BINARY_NAME = 'terraform'
    SEARCH_LEVELS = 3
    RELEASES_PAGE = 'https://releases.hashicorp.com/terraform/'
    MAX_VERSION = '0.11.14'

    def __call__(self, tf_args, current_provider, disable_ssh_tunnel):
        if not any(noarg in tf_args for noarg in ('output', 'show')):
            tf_args.extend(
                ['-target=module.ssh_manager',
                 '-target=module.%s_provider' % (current_provider),
                 '-var-file=' + os.path.abspath('variables.tfvars')])

            if not disable_ssh_tunnel:
                helpers.verbose_print('Starting SSH tunneling server')
                subprocess.Popen(
                    [os.path.join(helpers.ZYME_TOOLS_DIR, 'python'),
                     os.path.join(helpers.ZYME_TOOLS_DIR, 'proxy_ssh.py'),
                     os.path.abspath('.zyme-ssh-tunnel.state'),
                     'server'] +
                    (['--silent'] if not helpers.verbose_print.is_verbose
                        else []),
                    close_fds=True, shell=True)

        subprocess.call([self.target] + list(tf_args))

    def _get_variable_output(self, variable_name=None):
        return subprocess.check_output(
            [self.target, 'output', '-json'] +
            ([variable_name] if variable_name else []))
