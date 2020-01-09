from __future__ import absolute_import

from .terraform import TerraformWrapper
from .packer import PackerWrapper

ALL_WRAPPED = [TerraformWrapper, PackerWrapper]

VERSION_INFO = (1, 0, 6)
VERSION = '.'.join(str(c) for c in VERSION_INFO)


def ensure_all_wrapped():
    import helpers

    for wrapped in ALL_WRAPPED:
        if not wrapped.detect(exit_on_absent=False):
            wrapped.ensure()
        helpers.verbose_print('Ensured %s' % wrapped.__name__)
