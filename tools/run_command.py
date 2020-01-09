from __future__ import print_function
import paramiko
import sys
import time
import json
import make_proxy_socket
import wrapped
import subprocess
import errno
import os
import socks
import helpers


MPIRUN = '/opt/intel/compilers_and_libraries_2018/linux/mpi/bin64/mpirun'


class ConnectionError(Exception):
    pass


def get_credentials_from_terraform(hostname=None,
                                   username=None, pkey_file=None):
    """
    Gets hostname for cluster's login node, username and private key
    in OpenSSH format for authentication; overrides input variables by
    values from the terraform in case they are undefined.

    Parameters
    ----------
    hostname: str, default None
    username: str, default None
        using for authentication
    pkey_file: str, default None
        private key in OpenSSH format

    Returns
    -------
    (hostname, username, pkey_file): tuple

    Notes
    -----
    In case there are undefined variables, the ValueError exception is thrown.

    """
    terraform = wrapped.TerraformWrapper(exit_on_absent=True)
    output_tf = terraform._get_variable_output()
    try:
        data = json.loads(output_tf)
    except ValueError:
        data = None

    host_override = username_override = pkey_file_override = None

    if data:
        for key, value in data.items():
            # maybe prefix - {aws, gcp}
            if key.endswith('login_address'):
                if host_override is not None:
                    raise ValueError(
                        "RHOC doesn't support several providers")
                host_override = value['value']

        username_override = data.get('username', {}).get('value', None)
        pkey_file_override = data.get('pkey_file', {}).get('value', None)

        # attemp to override undefined variables
        hostname = hostname or host_override
        username = username or username_override
        pkey_file = pkey_file or pkey_file_override

    if not all((hostname, username, pkey_file)):
        raise ValueError('some of the variables {hostname, username, '
                         'pkey_file} are undefined')

    return (hostname, username, pkey_file)


class ZymeClient:

    def __init__(self, hostname, username,
                 pkey_file, use_proxy=True, add2known_hosts=True):
        """
        Create SSH client.

        Parameters
        ----------
        hostname: str
        username: str
            using for authentication
        pkey_file: str
            private key in OpenSSH format
        use_proxy: bool, default True
            creates socksocket with SOCKS5 protocol to use by paramiko connect
        add2known_hosts: bool, default True
            automatically adding the hostname and new host key to the local
            known_hosts file

        """
        self.hostname = hostname
        self.username = username
        self.pkey_file = pkey_file
        self.use_proxy = use_proxy
        self.add2known_hosts = add2known_hosts
        self.connected = False

    def __enter__(self):
        """
        Connect to host.

        Returns
        -------
        client: ZymeClient

        """
        helpers.verbose_print("Connecting to: %s" % self.hostname)

        if self.use_proxy:
            self.sock = make_proxy_socket.proxy_socket()

            try:
                self.sock.connect((self.hostname.encode('utf-8'), 22))
            except socks.SOCKS5Error as err:
                raise ConnectionError("%s; check hostname" % err)

        else:
            self.sock = None

        self.client = paramiko.SSHClient()
        if self.add2known_hosts:
            self.client.set_missing_host_key_policy(paramiko.AutoAddPolicy())

        try:
            self.client.connect(hostname=self.hostname, username=self.username,
                                key_filename=self.pkey_file,
                                port=22, sock=self.sock)
        except IOError as err:
            raise ConnectionError(err)
        except paramiko.ssh_exception.SSHException:
            raise ConnectionError("server refused key; check credentials "
                                  "(username, private key) - (%s, %s)" %
                                  (self.username, self.pkey_file))

        self.connected = True
        helpers.verbose_print("Connected")

        return self

    def __exit__(self, type, value, traceback):
        self.client.close()
        if self.use_proxy:
            self.sock.close()

        self.connected = False

    def verify_connected(self):
        assert self.connected, 'ZymeClient is not connected'

    @property
    def home(self):
        self.verify_connected()

        stdin, stdout, stderr = self.client.exec_command('echo $HOME')
        return stdout.read().rstrip('\n')

    def exec_command(self, command, get_output=True,
                     stdout_log_file='stdout_log.txt',
                     stderr_log_file='stderr_log.txt', chunksize=1024):
        """
        Execute the command on remote host by using connected SSH client.

        Parameters
        ----------
        command:          str
        get_output:       bool, default True
            display information from stdout and stderr streams on remote host
            localy
        stdout_log_file:  str, default 'stdout_log.txt'
        stderr_log_file:  str, default 'stderr_log.txt'
        chunksize:        int, default 1024

        Returns
        -------
        retcode: int
            result of the executed command

        """
        self.verify_connected()
        stdin, stdout, stderr = self.client.exec_command(command, bufsize=1)

        stdout.channel.setblocking(0)
        stderr.channel.setblocking(0)

        while True:

            while stdout.channel.recv_ready():
                outdata = stdout.channel.recv(chunksize)
                if get_output:
                    sys.stdout.write(outdata)
                if stdout_log_file:
                    with open(stdout_log_file, 'a') as out:
                        out.write(outdata)

            while stderr.channel.recv_stderr_ready():
                errdata = stderr.channel.recv_stderr(chunksize)
                if get_output:
                    sys.stderr.write(errdata)
                if stderr_log_file:
                    with open(stderr_log_file, 'a') as out:
                        out.write(errdata)

            if stdout.channel.exit_status_ready() and \
                    stderr.channel.exit_status_ready():  # If completed
                break
            time.sleep(0.5)

        return stdout.channel.recv_exit_status()

    def _get_sftp_client(self):
        self.verify_connected()
        transport = self.client.get_transport()
        sftp_client = paramiko.SFTPClient.from_transport(transport)
        return sftp_client

    def _make_dir(self, remotepath):
        self.verify_connected()
        dir_path, file_name = os.path.split(remotepath)
        stdin, stdout, stderr = self.client.exec_command(
            'mkdir -p "%s"' % dir_path)
        res = stdout.read().rstrip('\n') + stderr.read().rstrip('\n')

        if res:
            raise OSError(-1, "Can't create file on remote host (%s); "
                          "result - %s" % (remotepath, res), remotepath)

    def put_file(self, localpath, remotepath,
                 overwrite=False, chunksize=1024**2):
        """
        Copy a local file (localpath) to the SFTP server as remotepath by using
        connected SSH client.

        Parameters
        ----------
        local_path: str
        remotepath: str
        chunksize: int, default 1024

        Returns
        -------
        file_attr: paramiko.sftp_attr.SFTPAttributes

        """
        def file_content_generator():
            with open(localpath, mode='rb') as f:
                while True:
                    data = f.read(chunksize)
                    if not data:
                        break
                    yield data

        return self._put_string(file_content_generator,
                                remotepath, overwrite, 'w')

    def put_string(self, file_content, remotepath,
                   overwrite=False, mode='w'):
        """
        Copy a file_content string to the SFTP server as remotepath by using
        connected SSH client.

        Parameters
        ----------
        file_content: str
        remotepath: str
        mode: str, default 'w'
            mode for open function

        Returns
        -------
        file_attr: paramiko.sftp_attr.SFTPAttributes

        """
        def string_generator():
            yield file_content

        return self._put_string(string_generator, remotepath, overwrite, mode)

    def _put_string(self, data_generator, remotepath, overwrite, mode):
        with self._get_sftp_client() as sftp_client:
            exist_remotepath = True
            try:
                sftp_client.stat(remotepath)
            except IOError:
                # remotepath not exist
                exist_remotepath = False

            if not exist_remotepath or overwrite:
                try:
                    with sftp_client.file(remotepath, mode=mode) as f:
                        for data in data_generator():
                            f.write(data)
                except IOError:
                    self._make_dir(remotepath)
                    self._put_string(data_generator,
                                     remotepath, overwrite, mode)
            else:
                raise OSError(errno.EEXIST,
                              '%s already exists' % remotepath, remotepath)

            return sftp_client.stat(remotepath)

    def expand_home_path(self, remotepath):
        if remotepath.startswith('~'):
            remotepath = remotepath.replace('~', self.home)
        return remotepath

    def ensure_remotepath(self, remotepath):
        if remotepath is None:
            remotepath = '~/run_script.sh'

        return self.expand_home_path(remotepath)


def check_script(file_path):
    """
    File having string shebang is treated as a script, otherwise
    it is treated as binary.

    Parameters
    ----------
    file_path: str

    Returns
    -------
    : bool
        True if script

    """
    with open(file_path) as f:
        return f.readline().startswith('#!')


def convert_newline_symbol(local_file):
    """
    Converts DOS/Windows newline (CRLF) to UNIX newline (LF)
    in local_file content.

    Parameters
    ----------
    local_file: str

    Returns
    -------
    unix_like_string: str

    """
    with open(local_file, 'rU') as f:
        return f.read()


def identify_script_position(args):
    skip_next_arg = False
    mpi_script_pos = -1

    for idx, arg in enumerate(args):
        if not skip_next_arg:
            if arg.startswith('-') or arg.startswith('+'):
                skip_next_arg = True
                continue
            mpi_script_pos = idx
            break
        skip_next_arg = False

    if mpi_script_pos == -1:
        raise ValueError("script not found in run command's arguments")

    return mpi_script_pos


def prepare_run_args(run_args, remotepath, mpirun_nodefile=None):
    if mpirun_nodefile:
        # mpirun case
        try:
            pos = run_args.index('-f')
        except ValueError:
            run_args = ['-f', mpirun_nodefile] + run_args
        else:
            if len(run_args) > pos + 1:
                old_nodefile = run_args[pos + 1]
                run_args[pos + 1] = mpirun_nodefile
                print('Warning: changed nodefile: %s -> %s for mpi task' %
                      (old_nodefile, mpirun_nodefile))
            else:
                raise ValueError('After flag "-f" should be nodefile '
                                 '(what was lost) - (%s)' %
                                 subprocess.list2cmdline(run_args))

    script_pos = identify_script_position(run_args)
    script = run_args[script_pos]
    run_args[script_pos] = remotepath

    return (script, subprocess.list2cmdline(run_args))


def zyme_run_command(run_args, hostname=None, username=None,
                     pkey_file=None, remotepath=None,
                     newline_conversion=True,
                     overwrite=False, no_remove=False):
    """
    Run script on remote host.

    Parameters
    ----------
    script_args: list
        first element of list is script path;
        other - args that will be passed into script
    host: str
    username: str, default None
        using for authentication
    pkey_file: str, default None
        private key in OpenSSH format
    remotepath: str, default None
        the place to which the file will be copied
    newline_conversion: bool, default True
        convert DOS/Windows newline (CRLF) to UNIX newline (LF) in script
    overwrite: bool, default False
    no_remove: bool, default False

    Returns
    -------
    retcode: int

    """
    hostname, username, pkey_file = get_credentials_from_terraform(
        hostname, username, pkey_file)

    try:
        with ZymeClient(hostname, username, pkey_file) as connected_client:
            remotepath = connected_client.ensure_remotepath(remotepath)

            script_path, script_args_str = prepare_run_args(
                run_args, remotepath)

            try:
                if newline_conversion and check_script(script_path):
                    unix_string = convert_newline_symbol(script_path)
                    connected_client.put_string(unix_string,
                                                remotepath, overwrite)
                else:
                    connected_client.put_file(script_path,
                                              remotepath, overwrite)
            except Exception as err:
                print('Error: %s' % err)
                # return bad_retcode
                return 1

            connected_client.exec_command('chmod +x "%s"' % (remotepath))
            print('Command execution "%s":\n' % script_args_str)
            retcode = connected_client.exec_command(script_args_str)
            print('\nExecution complete')

            if not no_remove:
                connected_client.exec_command('rm -f "%s"' % remotepath)

        return retcode

    except ConnectionError as err:
        print('Error: %s' % err)
        return -1


def ensure_count_worker_nodes(desired_count_worker_nodes):
    """
    Specifies the count of working nodes.
    If there are more than in the cluster returns the maximum available count.

    Parameters
    ----------
    desired_count_worker_nodes: int

    Returns
    -------
    count_worker_nodes: int
        cluster's worker count

    """
    terraform = wrapped.TerraformWrapper(exit_on_absent=True)
    output_tf = terraform._get_variable_output('worker_count')

    try:
        data = json.loads(output_tf)
    except ValueError:
        raise ValueError('Cluster not created')

    current_worker_count = int(data['value'])

    if desired_count_worker_nodes is not None:
        if desired_count_worker_nodes > current_worker_count:
            print('Warning: desired number of nodes (%s) is greater '
                  'than maximum; the maximum number of available worker '
                  'nodes (%s) will be used' % (desired_count_worker_nodes,
                                               current_worker_count),
                  file=sys.stderr)
            desired_count_worker_nodes = current_worker_count
    else:
        desired_count_worker_nodes = current_worker_count

    return desired_count_worker_nodes


def prepare_nodefile_mpirun(connected_client,
                            desired_count_worker_nodes, mpirun_nodefile):
    current_count_worker_nodes = ensure_count_worker_nodes(
        desired_count_worker_nodes)

    terraform = wrapped.TerraformWrapper(exit_on_absent=True)
    output_tf = terraform._get_variable_output()

    try:
        data = json.loads(output_tf)
    except ValueError:
        raise ValueError('Cluster not created')

    for key in data.keys():
        # maybe prefix - {aws, gcp}
        if 'workers_private_ip' in key:
            workers_ip = data[key]['value']
            workers_str = '\n'.join(workers_ip[:current_count_worker_nodes])
            connected_client.put_string(workers_str,
                                        mpirun_nodefile, overwrite=True)
            return

    raise ValueError('Cluster not created')


def zyme_mpirun_command(mpirun_args, count_worker_nodes=None,
                        remotepath=None,
                        newline_conversion=True,
                        overwrite=False, no_remove=False):
    """
    Run mpi script on the remote host.

    Parameters
    ----------
    mpirun_args: list, default None
    count_worker_nodes: int, default None
    remotepath: str, default None
        the place to which mpi script will be copied
    newline_conversion: bool, default True
        convert DOS/Windows newline (CRLF) to UNIX newline (LF) in script
    overwrite: bool, default False
    no_remove: bool, default False

    Returns
    -------
    retcode: int

    """
    hostname, username, pkey_file = get_credentials_from_terraform()

    mpirun_nodefile = '~/.RHOC/nodefile_mpirun'

    try:
        with ZymeClient(hostname, username, pkey_file) as connected_client:
            remotepath = connected_client.ensure_remotepath(remotepath)

            mpi_script, mpirun_args_str = prepare_run_args(
                mpirun_args, remotepath, mpirun_nodefile)

            try:
                if newline_conversion and check_script(mpi_script):
                    unix_string = convert_newline_symbol(mpi_script)
                    connected_client.put_string(unix_string,
                                                remotepath, overwrite)
                else:
                    connected_client.put_file(mpi_script,
                                              remotepath, overwrite)
            except Exception as err:
                print('Error: %s' % err)
                # return bad_retcode
                return 1

            mpirun_nodefile = connected_client.expand_home_path(
                mpirun_nodefile)
            prepare_nodefile_mpirun(connected_client,
                                    count_worker_nodes, mpirun_nodefile)

            connected_client.exec_command('chmod +x "%s"' % (remotepath))

            command_exec = '%s %s' % (MPIRUN, mpirun_args_str)
            print('Command execution "%s":\n' % command_exec)
            retcode = connected_client.exec_command(command_exec)
            print('\nExecution complete')

            if not no_remove:
                connected_client.exec_command('rm -f "%s"' % remotepath)

        return retcode

    except ConnectionError as err:
        print('Error: %s' % err)
        return -1
