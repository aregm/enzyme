from __future__ import absolute_import

import contextlib
import re
import sys
import os
import zipfile
import subprocess

try:
    from html.parser import HTMLParser
except ImportError:
    from HTMLParser import HTMLParser
try:
    from urllib.request import urlopen
except ImportError:
    from urllib2 import urlopen
try:
    from urllib.parse import urljoin
except ImportError:
    from urlparse import urljoin

import helpers

from .base import WrappedCommandBase


class ReleasePageParser(HTMLParser):
    def __init__(self, name):
        HTMLParser.__init__(self)
        self.name = name
        self.links = []
        self.current_link = None

    def handle_starttag(self, tag, attrs):
        if tag == 'a' and not self.current_link:
            self.current_link = [attrs]

    def handle_endtag(self, tag):
        if tag == 'a':
            try:
                attrs, data = self.current_link
            except ValueError:
                pass
            else:
                attrs = dict(attrs)
                if self.name in data and 'href' in attrs:
                    helpers.verbose_print('Found potential release: %s' % data)
                    self.links.append(attrs['href'])
        self.current_link = []

    def handle_data(self, data):
        if self.current_link:
            self.current_link.append(data)

    @classmethod
    def parse_url(cls, url, name):
        helpers.verbose_print('Querying "%s" for releases' % url)
        with contextlib.closing(urlopen(url)) as page:
            data = page.read()
        parser = cls(name)
        parser.feed(data)
        return parser.links


PYPLATFORM_TO_HASHICORP = {
    'win32': 'windows',
    'linux2': 'linux',
}
try:
    TARGET_PLATFORM = PYPLATFORM_TO_HASHICORP[sys.platform]
except KeyError:
    raise Exception('Unsupported platform "%s"' % sys.platform)
TARGET_ARCH = 'amd64' if sys.maxsize > 2**32 else '386'


class HashicorpWrappedCommand(WrappedCommandBase):
    RELEASES_PAGE = 'url-to-releases-page'
    MAX_VERSION = '99999'
    BAD_RELEASES = []

    @staticmethod
    def __parse_version(version_str):
        try:
            substr = re.search(r'(\d+(?:\.\d+)*)', version_str).group(1)
        except AttributeError:
            return []
        return [int(x) for x in re.findall(r'\d+', substr)]

    @staticmethod
    def __format_version(version_iter):
        return '.'.join(str(x) for x in version_iter)

    @classmethod
    def is_version_okay(cls, version):
        allowed_max_version = cls.__parse_version(cls.MAX_VERSION)
        if not version or version > allowed_max_version:
            helpers.verbose_print('Version %s of %s is excluded' % (
                cls.__format_version(version),
                cls.BINARY_NAME))
            return False
        for bad_release in cls.BAD_RELEASES:
            bad_start, bad_end = [cls.__parse_version(x)
                                  for x in bad_release.split('-', 1)]
            if version >= bad_start and version <= bad_end:
                helpers.verbose_print('Version %s of %s is excluded' % (
                    cls.__format_version(version), cls.BINARY_NAME))
                return False
        return True

    @classmethod
    def __get_max_version(cls):
        links = ReleasePageParser.parse_url(cls.RELEASES_PAGE, cls.BINARY_NAME)
        versions = []
        for link in links:
            try:
                version = re.search(r'/(\d+(?:\.\d+)*)/', link).group(1)
            except AttributeError:
                # bad url, e.g. a release candidate
                continue
            version = cls.__parse_version(version)
            if cls.is_version_okay(version):
                versions.append((version, link))
        if not versions:
            sys.exit('Cannot find any version of "%s"' % cls.BINARY_NAME)
        version, link = max(versions, key=lambda entry: entry[0])
        helpers.verbose_print('Found max version: %s' %
                              '.'.join(str(x) for x in version))

        return urljoin(cls.RELEASES_PAGE, link)

    @staticmethod
    def __download_link(link):
        fname = os.path.join(helpers.ZYME_TOOLS_DIR, link.rsplit('/', 1)[1])
        helpers.verbose_print('Downloading %s: ' % link, end='')
        with contextlib.closing(urlopen(link)) as src:
            with open(fname, 'wb') as target:
                while True:
                    chunk = src.read(1024*1024)
                    if not chunk:
                        break
                    target.write(chunk)
                    helpers.verbose_print('.', end='', sep='')
        helpers.verbose_print(' done')
        return fname

    @classmethod
    def __unpack_archive(cls, fname):
        with zipfile.ZipFile(fname, 'r') as zf:
            for ext in helpers.EXE_EXTS:
                try:
                    info = zf.getinfo(cls.BINARY_NAME + ext)
                except KeyError:
                    # no such file in archive
                    continue
                helpers.verbose_print('Extracting %s to %s' %
                                      (info.filename, helpers.ZYME_TOOLS_DIR))

                target_fname = os.path.join(helpers.ZYME_TOOLS_DIR,
                                            os.path.basename(info.filename))

                with open(target_fname, 'wb') as target:
                    with zf.open(info.filename, 'r') as src:
                        while True:
                            chunk = src.read(1024 * 1024)
                            if not chunk:
                                break
                            target.write(chunk)
                helpers.mark_as_executable(target_fname)
                return
        sys.exit('Cannot find "%s" in %s' % (cls.BINARY_NAME, link))

    @classmethod
    def ensure(cls):
        maxRelease = cls.__get_max_version()
        links = ReleasePageParser.parse_url(maxRelease, cls.BINARY_NAME)
        links = [l for l in links if TARGET_ARCH in l and TARGET_PLATFORM in l]
        if not links:
            sys.exit('Cannot find a version for current platform (%s %s)' %
                     (TARGET_PLATFORM, TARGET_ARCH))

        link = urljoin(maxRelease, links[0])
        if not link.endswith('.zip'):
            sys.exit('Do not know how to unpack %s' % link)
        fname = cls.__download_link(link)
        cls.__unpack_archive(fname)
        os.unlink(fname)

    @classmethod
    def validate_binary(cls, target_path):
        try:
            output = subprocess.check_output([target_path, '--version'],
                                             stderr=subprocess.STDOUT)
        except subprocess.CalledProcessError as err:
            helpers.verbose_print('Cannot check version of %s: %s' %
                                  (target_path, err.output))
            return False

        tool_version = cls.__parse_version(output.decode('utf-8'))
        helpers.verbose_print('Detected %s version as %s' %
                              (target_path,
                               cls.__format_version(tool_version)))

        return cls.is_version_okay(tool_version)
