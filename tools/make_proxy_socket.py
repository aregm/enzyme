import os

from helpers import split_address

SOCKS_PROXY = 'socks-proxy.txt'


def read_socks_proxy():
    try:
        with open(os.path.join(os.path.dirname(__file__), SOCKS_PROXY)) as cfg:
            hostname = cfg.readline().strip()
    except IOError:
        return None, None
    if not hostname:
        # empty socks-proxy.txt
        return None, None
    # check if socks proxy host exists
    try:
        return split_address(hostname, 1080)
    except ValueError:
        return None, None


def proxy_socket():
    socks_proxy, socks_port = read_socks_proxy()
    if not socks_proxy:
        # not behind a proxy, use direct connection
        return None

    import socks
    sock = socks.socksocket()
    sock.set_proxy(socks.SOCKS5, socks_proxy, socks_port)
    return sock


if __name__ == '__main__':
    print(read_socks_proxy())
