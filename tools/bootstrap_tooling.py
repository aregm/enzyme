"""
Placeholder.
"""

import subprocess
import os
import sys
import errno

try:
    from urllib.request import urlopen
except ImportError:
    from urllib2 import urlopen

import helpers
import wrapped


def relative_join(*paths):
    return os.path.join(os.path.dirname(os.path.abspath(__file__)), *paths)


TARGET_VENV = 'pylib'
TARGET_VENV = relative_join(TARGET_VENV)

if sys.platform == 'win32':
    PYTHON_EXE = os.path.join(TARGET_VENV, 'Scripts',
                              os.path.basename(sys.executable))
    PIP_EXE = os.path.join(TARGET_VENV, 'Scripts', 'pip.exe')
elif sys.platform in ['linux2']:
    PYTHON_EXE = os.path.join(TARGET_VENV, 'bin',
                              os.path.basename(sys.executable))
    PIP_EXE = os.path.join(TARGET_VENV, 'bin', 'pip')
else:
    raise Exception('Unsupported platform')


def check_access(url):
    print('Checking for access to "%s"' % url)

    try:
        page = urlopen(url, timeout=3)
    except IOError:
        print('Cannot access "%s"' % url)
        return False

    page.close()
    print('Access to "%s" verified' % url)
    return True


def ensure_venv():
    if os.path.exists(PYTHON_EXE) and os.path.exists(PIP_EXE):
        print('Target venv exists')
        return

    print('Searching for venv')
    venv_path = helpers.detect_binary('virtualenv')
    if not venv_path:
        sys.exit('Cannot find virtualenv')
    print('Found virtualenv: %s' % venv_path)

    print('Creating venv')
    try:
        subprocess.check_output([venv_path, TARGET_VENV],
                                stderr=subprocess.STDOUT)
    except subprocess.CalledProcessError as err:
        sys.exit('Cannot create venv:\n%s' % err.output)
    print('Created venv')


def ensure_python_pkgs():
    print('Ensuring required Python packages are installed')

    try:
        subprocess.check_output([PIP_EXE, 'install', '-r',
                                 relative_join('requirements.txt')],
                                stderr=subprocess.STDOUT)
    except subprocess.CalledProcessError as err:
        sys.exit('Cannot install PIP packages:\n%s' % err.output)

    print('Ensured packages')


def main():
    if '--verbose' in sys.argv:
        helpers.enable_verbose_print()

    if not check_access('https://pypi.python.org/'):
        sys.exit('Cannot reach PyPI, probably HTTPS proxy not set')

    ensure_venv()
    ensure_python_pkgs()

    print('Ensuring wrapped tools are present')
    wrapped.ensure_all_wrapped()

    if sys.platform == 'win32':
        with open(relative_join('python.cmd'), 'w') as out:
            out.write('@%s %%*\n' % subprocess.list2cmdline([PYTHON_EXE]))

        with open(relative_join('..', 'zyme.cmd'), 'w') as out:
            out.write('@%s %%*\n' % subprocess.list2cmdline(
                [PYTHON_EXE, relative_join('zyme.py')]))
    else:
        try:
            os.unlink(relative_join('python'))
        except OSError as err:
            if err.errno != errno.ENOENT:
                raise
        os.symlink(PYTHON_EXE, relative_join('python'))
        with open(relative_join('..', 'zyme'), 'w') as out:
            out.write('#!/bin/sh\n%s "$@"\n' % subprocess.list2cmdline(
                [PYTHON_EXE, relative_join('zyme.py')]))

        helpers.mark_as_executable(relative_join('..', 'zyme'))

    print("Bootstrapping succeeded")


if __name__ == '__main__':
    main()
