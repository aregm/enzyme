import argparse
import errno
import json
import multiprocessing.connection
import threading
import time
import subprocess
import sys
import os
import socket
import sshtunnel

from helpers import split_address

try:
    from make_proxy_socket import proxy_socket
except ImportError:
    def proxy_socket():
        # not behind a proxy
        return None

CONN_FAMILY = 'AF_INET'
CONN_HOST = '127.0.0.1'


def find_empty_port(hostname, start_port):
    port = start_port
    while port < 65536:
        try:
            sock = socket.create_connection((hostname, port), 1)
        except socket.timeout:
            # free port found
            break
        except socket.error as err:
            if err.errno == errno.ECONNREFUSED:
                # connection refused - port is probably free
                break
            # some other reason - port busy
            pass
        else:
            sock.close()
        port += 1
    return port


def make_good_path(pkey):
    return os.path.normpath(os.path.abspath(os.path.expanduser(pkey)))


class Tunnel(object):
    def __init__(self, public_address,
                 username, pkey, remote_bind, local_bind):

        self.public_address = split_address(public_address, 22)
        self.username = username
        self.pkey = make_good_path(pkey)
        self.remote_bind = split_address(remote_bind, 22)
        self.local_bind = split_address(local_bind, 10022)
        self.tunnel = None

    def __eq__(self, other):
        if not isinstance(other, Tunnel):
            return NotImplemented
        for attr in 'public_address username pkey remote_bind'.split():
            if getattr(self, attr) != getattr(other, attr):
                return False
        # we're specifically ignoring local bind port
        # when comparing two tunnels
        return self.local_bind[0] == other.local_bind[0]

    @staticmethod
    def __flatten(*args):
        for arg in args:
            if isinstance(arg, (tuple, list)):
                for item in arg:
                    yield item
            else:
                yield arg

    def __str__(self):
        return '%s:%s => [%s@%s:%s] => %s:%s' % tuple(self.__flatten(
            self.local_bind, self.username,
            self.public_address, self.remote_bind))

    def start(self):
        self.__find_empty_port()
        self.tunnel = sshtunnel.SSHTunnelForwarder(
            self.public_address,
            ssh_username=self.username,
            ssh_pkey=self.pkey,
            remote_bind_address=self.remote_bind,
            local_bind_address=self.local_bind,
            ssh_proxy=proxy_socket(),
            set_keepalive=10)
        self.tunnel.start()

    def stop(self):
        self.tunnel.stop()

    def __find_empty_port(self):
        hostname, port = self.local_bind
        self.local_bind = hostname, find_empty_port(hostname, port)

    @property
    def is_active(self):
        return self.tunnel.is_active


class TunnelServer(object):
    MSG_ENSURE = 'ensure tunnel'
    MSG_STOP = 'stop server'
    MSG_STATUS = 'get status'
    BAD_REQUEST = 'bad request'
    PROCESSED = 'processed'

    def __init__(self, state_file, silent):
        conn_address = CONN_HOST, find_empty_port(CONN_HOST, 6000)
        try:
            self.listener = multiprocessing.connection.Listener(conn_address,
                                                                CONN_FAMILY)
        except Exception as err:
            try:
                os.remove(state_file)
            except OSError:
                pass
            raise err
        if not silent:
            print('[STATUS] started listener')
        with open(state_file, 'w') as state:
            state.write('%s %s %s' % ((CONN_FAMILY,) + (conn_address)))

        self.lock = threading.RLock()
        self.cleanup_lock = threading.RLock()
        self.tunnels = []
        self.stop_flag = False
        self.silent = silent

    def start_tunnel(self, public_address,
                     username, pkey, remote_bind, local_bind):
        try:
            tunnel = Tunnel(public_address,
                            username, pkey, remote_bind, local_bind)

        except ValueError as err:
            if not self.silent:
                print('Cannot start tunnel: %s' % err)
            return None

        with self.lock:
            for existing in self.tunnels:
                if existing.is_active and existing == tunnel:
                    return existing
            # no existing tunnel matches given, add current
            tunnel.start()
            self.tunnels.append(tunnel)
            return tunnel

    def talk_to_client(self, conn):
        try:
            msg = conn.recv()
            if not self.silent:
                print('[STATUS] got request with params: %s' % str(msg))
            if not isinstance(msg, (tuple, list)):
                conn.send([self.BAD_REQUEST, 'wrong message type'])
            elif msg[0] == self.MSG_ENSURE:
                try:
                    public_address, username, pkey, remote_bind, local_bind = msg[1:]
                except ValueError:
                    conn.send([self.BAD_REQUEST, 'invalid ensure params'])
                else:
                    tunnel = self.start_tunnel(public_address, username,
                                               pkey, remote_bind, local_bind)
                    if tunnel:
                        conn.send([self.PROCESSED, tunnel.local_bind])
                    else:
                        conn.send([self.BAD_REQUEST, 'Cannot start tunnel'])
            elif msg[0] == self.MSG_STOP:
                self.stop_flag = True
                conn.send([self.PROCESSED, 'stopped'])
                with self.cleanup_lock:
                    if self.listener:
                        # simulate a connection to make main loop
                        # break out of accept()
                        fake_conn = multiprocessing.connection.Client(
                            self.listener.address, CONN_FAMILY)

                        fake_conn.close()
            elif msg[0] == self.MSG_STATUS:
                result = [self.PROCESSED]
                with self.lock:
                    for tunnel in self.tunnels:
                        if tunnel.is_active:
                            result.append(str(tunnel))
                conn.send(result)
            else:
                conn.send([self.BAD_REQUEST, 'invalid message'])
        except EOFError:
            pass
        finally:
            conn.close()

    def cleanup(self):
        if not self.silent:
            print('[STATUS] cleaning up')
        with self.cleanup_lock:
            with self.lock:
                for tunnel in self.tunnels:
                    tunnel.stop()
            self.listener.close()
            self.listener = None

    def run(self):
        try:
            while not self.stop_flag:
                conn = self.listener.accept()
                thread = threading.Thread(target=self.talk_to_client,
                                          args=(conn,))
                thread.start()
        finally:
            self.cleanup()


CLI_MESSAGES = {
    'stop': TunnelServer.MSG_STOP,
    'status': TunnelServer.MSG_STATUS,
}


def setup_server(args):
    # server_exists = False
    args.message = 'status'
    try:
        if message_server(args) is not None:
            if not args.silent:
                print('Server already running')
            return
    except SystemExit:
        # no server running
        pass
    server = TunnelServer(args.state_file, args.silent)
    server.run()


def connect_and_exec(args, callback, tries=10, do_spawn=True):
    spawned = False
    for i in range(tries):
        try:
            with open(args.state_file) as state:
                connect_family, connect_host, connect_port = state.readline().strip().split(' ')
            connect_port = int(connect_port)
            try:
                sock = socket.create_connection((connect_host, connect_port), 1)
            except socket.timeout:
                raise IOError('Bad connection info')
            else:
                sock.close()
        except (IOError, ValueError):
            # no current state, spawn server if not already done so
            if not spawned and do_spawn:
                spawn_server(args)
                spawned = True
        else:
            try:
                client = multiprocessing.connection.Client((connect_host,
                                                            connect_port),
                                                           connect_family)
            except OSError:
                # invalid connection data, spawn server if needed
                if not spawned and do_spawn:
                    spawn_server(args)
                    spawned = True
            else:
                try:
                    result = callback(args, client)
                    if result is not None:
                        return result
                finally:
                    client.close()
        if i < tries - 1:
            time.sleep(5)
    sys.exit('Cannot connect to server')


def send_message_to_server(args, client):
    client.send([CLI_MESSAGES[args.message]])
    response = client.recv()
    if isinstance(response, (tuple, list)) and response[0] == TunnelServer.PROCESSED:
        return response[1:]
    return None


def message_server(args):
    return connect_and_exec(args, send_message_to_server, 1, do_spawn=False)


def spawn_server(args):
    # re-construct command line called
    cli = [sys.executable, sys.argv[0], args.state_file, 'server', '--silent']
    subprocess.Popen(cli, close_fds=True)


def refresh_server(args, client):
    client.send([TunnelServer.MSG_ENSURE, args.public_address, args.username,
                 make_good_path(args.pkey), args.remote_bind, args.local_bind])

    response = client.recv()
    if isinstance(response, (tuple, list)) and response[0] == TunnelServer.PROCESSED:
        hostname, port = response[1]
        return {'bastion_host': str(hostname), 'bastion_port': str(port)}
    return None


def setup_client(args, tries=10):
    return connect_and_exec(args, refresh_server, tries)


if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Seamless SSH tunneling')
    parser.add_argument('state_file', metavar='FILENAME',
                        help='Path to file holding current state')

    subparsers = parser.add_subparsers(dest='action',
                                       help='actions %(prog)s can perform')

    parser_server = subparsers.add_parser(
        'server', help='Set up a server for keeping tunnels up')

    parser_server.add_argument('--silent', action='store_true', default=False,
                               help='Suppress any output')

    parser_message = subparsers.add_parser(
        'message', help='Send a message to running server')

    parser_message.add_argument('message', choices=sorted(CLI_MESSAGES.keys()))

    parser_tunnel = subparsers.add_parser(
        'tunnel', help='Ensure a tunnel is maintaned by %(prog)s server')

    parser_tunnel.add_argument(
        '--public_address',
        help='Publicly accessible address of the bastion', required=True)

    parser_tunnel.add_argument('--username',
                               help='Username for bastion', required=True)

    parser_tunnel.add_argument('--pkey',
                               help='Path to private key for bastion',
                               required=True)

    parser_tunnel.add_argument(
        '--remote_bind',
        help='Remote bind address (egress end of tunnel)', required=True)

    parser_tunnel.add_argument(
        '--local_bind',
        help='Local bind address (ingress end of tunnel)', required=True)

    args = parser.parse_args()

    if args.action == 'tunnel':
        tunnel = setup_client(args)
        if tunnel:
            print(json.dumps(tunnel))
    elif args.action == 'server':
        setup_server(args)
    elif args.action == 'message':
        print('\n'.join(message_server(args)))
    else:
        sys.exit('How did you get here?!')
