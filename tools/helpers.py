from __future__ import print_function

import os
import sys
try:
    from urllib.parse import urlsplit
except ImportError:
    from urlparse import urlsplit
import socket
import json
import time

PROXY_CACHE_REFRESH = 60 * 60  # force refresh if cache is older than 1 hour

ZYME_TOOLS_DIR = os.path.abspath(os.path.expanduser(os.path.dirname(__file__)))


if sys.platform == 'win32':
    EXE_EXTS = ['.exe', '.cmd', '.bat']
else:
    EXE_EXTS = ['']


def verbose_print(*args, **kw):
    pass
verbose_print.is_verbose = False


def enable_verbose_print():
    global verbose_print

    def do_print(*args, **kw):
        print(*args, **kw)
        sys.stdout.flush()

    verbose_print = do_print
    verbose_print.is_verbose = True


DEFAULT_SCHEME_PORTS = {
    'http': 80,
    'https': 443,
}


def split_address(addr, default_port=None):
    if '//' not in addr:
        addr = '//' + addr
    splitted = urlsplit(addr)

    try:
        socket.gethostbyname(splitted.hostname)
    except socket.error:
        raise ValueError('Invalid address: hostname "%s" does not exist' %
                         splitted.hostname)

    port = splitted.port or default_port or \
        DEFAULT_SCHEME_PORTS.get(splitted.scheme, 80)

    return splitted.hostname, int(port)


def __always_true(target_path):
    return True


def detect_binary_in_list(name, pathlist, validator=None):
    if not validator:
        validator = __always_true

    for path_entry in pathlist:
        path_entry = os.path.abspath(os.path.expanduser(path_entry))

        for ext in EXE_EXTS:
            target_path = os.path.join(path_entry, name + ext)
            if os.path.exists(target_path) and validator(target_path):
                return target_path

    return None


def detect_binary(name, env_name='PATH', validator=None):
    verbose_print('Searching for "%s" in %s' % (name, env_name))

    try:
        pathlist = os.environ[env_name].split(os.pathsep)
    except KeyError:
        verbose_print('"%s" environment variable does not exist, '
                      'cannot search for "%s"' % (env_name, name))
        return None

    target_path = detect_binary_in_list(name, pathlist, validator)
    if target_path:
        verbose_print('Found "%s" in %s as "%s"' % (name, env_name,
                                                    target_path))
    else:
        verbose_print('"%s" not found in %s' % (name, env_name))

    return target_path


PROXY_PROTOS = ('http', 'https')


def _detect_proxies_by_pac(target_host):
    import pypac
    result = {}
    verbose_print('Detecting proxy via PAC')
    pac = pypac.get_pac()
    if pac:
        for proto in PROXY_PROTOS:
            proxy = pypac.parser.proxy_url(
                pac.find_proxy_for_url('%s://%s' % (proto, target_host),
                                       target_host))
            if proxy:
                verbose_print('Found "%s" proxy for "%s" proto' % (proxy,
                                                                   proto))
                result[proto] = proxy
    else:
        verbose_print('No proxy found')
    return result


def _apply_proxies_to_env(proxies):
    for proto in PROXY_PROTOS:
        os.environ['%s_proxy' % proto] = proxies.get(proto, '')


def detect_proxies_and_update_env(target_host, proxy_cache=None, force_update=False):
    proxies, cache_data, force_cache_store = None, {}, False
    if proxy_cache and not force_update:
        try:
            with open(proxy_cache) as cache:
                cache_data = json.loads(cache.read())
        except (ValueError, IOError):
            verbose_print('Cache file "%s" is invalid' % proxy_cache)
        else:
            if cache_data.get('timestamp', 0) + PROXY_CACHE_REFRESH >= time.time():
                proxies = cache_data.get('%s-proxies' % target_host, None)
                if proxies is not None:
                    verbose_print('Read proxies from cache successfully')
                    for proxy in proxies.values():
                        try:
                            split_address(proxy)
                        except ValueError:
                            verbose_print('Proxy "%s" does not exist, refreshing' % proxy)
                            proxies = None
                            break
            else:
                verbose_print("Refreshing proxy cache file as it's too old")
                force_cache_store = True

    cached_proxies = {} if proxies is None else dict(proxies)
    if proxies is None:
        proxies = _detect_proxies_by_pac(target_host)
        verbose_print('Checking that proxy hosts exist')
        for proto, proxy in list(proxies.items()):
            try:
                split_address(proxy)
            except ValueError:
                verbose_print('Proxy host "%s" found but does not exist' % proxy)
                del proxies[proto]

    if proxy_cache and (cached_proxies != proxies or force_cache_store):
        cache_data['timestamp'] = time.time()
        cache_data['%s-proxies' % target_host] = proxies
        try:
            with open(proxy_cache, 'w') as cache:
                cache.write(json.dumps(cache_data))
        except IOError as err:
            verbose_print('Cannot update "%s" proxy cache: %s' % (proxy_cache, err))

    _apply_proxies_to_env(proxies)
    return proxies


def mark_as_executable(path):
    mode = os.stat(path).st_mode
    mode |= (mode & 0o444) >> 2  # copy R bits to X
    os.chmod(path, mode)
