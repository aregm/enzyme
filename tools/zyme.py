import sys
import argparse

import helpers
import wrapped

from wrapped.make_variables_files import (
    PackerConfig, TerraformConfig, UserFileError)

from run_command import zyme_mpirun_command, zyme_run_command


def make_terraform_arg_subparser(zyme_subparsers):
    """
    Creates Terraform's command parser.

    Parameters
    ----------
    zyme_subparsers: argparse._SubParsersAction

    Returns
    -------
    tf_parser: argparse.ArgumentParser

    """

    tf_parser = zyme_subparsers.add_parser(
        'tf', help='Call wrapped terraform', prefix_chars='+')

    tf_parser.add_argument(
        'tf_args', metavar='TERRAFORM ARGS', nargs='+',
        help='Arguments to pass to underlying terraform')

    tf_parser.add_argument(
        '++no+ssh+tunnel', dest='no_ssh_tunnel',
        action='store_true', default=False,
        help='Disable auto-start SSH tunneling server')

    return tf_parser


def make_packer_arg_subparser(zyme_subparsers):
    """
    Creates Packer's command parser.

    Parameters
    ----------
    zyme_subparsers: argparse._SubParsersAction

    Returns
    -------
    pk_parser: argparse.ArgumentParser

    """

    pk_parser = zyme_subparsers.add_parser(
        'pk', help='Call wrapped packer', prefix_chars='+')

    pk_parser.add_argument(
        'pk_args', metavar='PACKER ARGS', nargs='+',
        help='Arguments to pass to underlying packer')

    pk_parser.add_argument(
        '++no+socks+proxy', dest='no_socks_proxy',
        action='store_true', default=False,
        help='Disable auto-patching socks proxy in packer json configs')

    return pk_parser


def make_run_arg_subparser(zyme_subparsers):
    """
    Creates RHOC's Run command parser.

    Parameters
    ----------
    zyme_subparsers: argparse._SubParsersAction

    Returns
    -------
    run_parser: argparse.ArgumentParser

    """
    run_parser = zyme_subparsers.add_parser(
        'run', help='Run script', prefix_chars='+')

    run_parser.add_argument(
        '++host', default=None, help='host that will execute script')

    run_parser.add_argument(
        '++username', default=None, help='name for authentication')

    run_parser.add_argument(
        '++pkey_file', default=None,
        help='file with private key that will be used for authentication')

    run_parser.add_argument(
        '++remotepath', default=None,
        help='script name in remote host')

    run_parser.add_argument(
        '++no_newline_conversion', action='store_true',
        help='disable convertion of DOS/Windows newlines to UNIX newlines'
    )

    run_parser.add_argument(
        '++overwrite', action='store_true',
        help=('overwrite the content of the remote file '
              'with the content of the local file')
    )

    run_parser.add_argument(
        '++no_remove', action='store_true',
        help="don't remove file uploaded to remote"
    )

    run_parser.add_argument(
        'run_args', metavar='SCRIPT AND ARGS',
        nargs='+', help='full script path to remote run at host and args '
                        'that will be passed to script')

    return run_parser


def make_mpirun_arg_subparser(zyme_subparsers):
    """
    Creates RHOC's MPI Run command parser.

    Parameters
    ----------
    zyme_subparsers: argparse._SubParsersAction

    Returns
    -------
    run_parser: argparse.ArgumentParser

    """
    mpirun_parser = zyme_subparsers.add_parser(
        'mpirun', help='mpirun script', prefix_chars='+')

    mpirun_parser.add_argument(
        'mpirun_args', metavar='MPIRUN ARGS', nargs='+',
        help='Arguments to pass to remote mpirun command on login node')

    mpirun_parser.add_argument(
        '++remotepath', default=None,
        help='script name in remote host')

    mpirun_parser.add_argument(
        '++count_worker_nodes', default=None,
        type=int, help='count nodes for mpi task'
    )

    mpirun_parser.add_argument(
        '++no_newline_conversion', action='store_true',
        help='disable convertion of DOS/Windows newlines to UNIX newlines'
    )

    mpirun_parser.add_argument(
        '++overwrite', action='store_true',
        help=('overwrite the content of the remote file '
              'with the content of the local file')
    )

    mpirun_parser.add_argument(
        '++no_remove', action='store_true',
        help="don't remove file uploaded to remote"
    )

    return mpirun_parser


def make_zyme_arg_parser():
    """
    Creates Zyme's command parser.

    Returns
    -------
    : argparse.ArgumentParser

    """
    parser = argparse.ArgumentParser(
        description='RHOC CLI tool - managing a cloud cluster', prog='zyme')

    parser.add_argument('--verbose', action='store_true', default=False)

    parser.add_argument(
        '--no-proxy', '-np', action='store_true', default=False,
        help='Disable automatic proxy discovery via PAC')

    parser.add_argument(
        '--refresh-proxy', action='store_true', default=False,
        help='Force proxy settings update (ignore proxy settings cache)')

    subparsers = parser.add_subparsers(
        dest='command', help='Commands %(prog)s can perform')

    make_terraform_arg_subparser(subparsers)
    make_packer_arg_subparser(subparsers)
    make_run_arg_subparser(subparsers)
    make_mpirun_arg_subparser(subparsers)

    return parser


def main():
    parser = make_zyme_arg_parser()

    args = parser.parse_args()
    if args.verbose:
        helpers.enable_verbose_print()

    if not args.no_proxy:
        helpers.detect_proxies_and_update_env('amazon.com',
                                              proxy_cache='.zyme-proxy-cache',
                                              force_update=args.refresh_proxy)

    if args.command == 'run':
        zyme_run_command(args.run_args, args.host, args.username,
                         args.pkey_file, args.remotepath,
                         not args.no_newline_conversion,
                         args.overwrite, args.no_remove)

    if args.command == 'mpirun':
        zyme_mpirun_command(args.mpirun_args, args.count_worker_nodes,
                            args.remotepath,
                            not args.no_newline_conversion,
                            args.overwrite, args.no_remove)

    if args.command == 'tf':
        try:
            t_conf = TerraformConfig.prepare_terraform_config()
        except UserFileError as e:
            sys.exit(e)

        terraform = wrapped.TerraformWrapper(exit_on_absent=True)
        terraform(args.tf_args, t_conf.current_provider, args.no_ssh_tunnel)
    elif args.command == 'pk':
        try:
            p_conf = PackerConfig.prepare_packer_config()
        except UserFileError as e:
            sys.exit(e)

        packer = wrapped.PackerWrapper(exit_on_absent=True)
        packer(args.pk_args, p_conf.current_builder, args.no_socks_proxy)


if __name__ == '__main__':
    main()
