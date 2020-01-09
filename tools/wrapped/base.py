import os
import sys

import helpers


class WrappedCommandBase(object):
    ENV_NAME = 'environment-var-name-with-search-path'
    BINARY_NAME = 'binary-to-search'
    SEARCH_LEVELS = 3

    @classmethod
    def validate_binary(cls, target_path):
        return True

    @classmethod
    def detect(cls, exit_on_absent):
        binary = helpers.detect_binary(cls.BINARY_NAME, cls.ENV_NAME,
                                       validator=cls.validate_binary)
        if binary:
            return binary

        binary = helpers.detect_binary(cls.BINARY_NAME,
                                       validator=cls.validate_binary)
        if binary:
            return binary

        helpers.verbose_print('Cannot find %s via environment, '
                              'searching up to %s levels up from "%s"' %
                              (cls.BINARY_NAME,
                               cls.SEARCH_LEVELS, helpers.ZYME_TOOLS_DIR))

        search_dirs = [os.path.join(helpers.ZYME_TOOLS_DIR,
                                    *[os.pardir] * level)
                       for level in range(cls.SEARCH_LEVELS + 1)]

        binary = helpers.detect_binary_in_list(cls.BINARY_NAME, search_dirs,
                                               validator=cls.validate_binary)
        if binary:
            helpers.verbose_print('Found "%s" as "%s"' %
                                  (cls.BINARY_NAME, binary))
            return binary

        if exit_on_absent:
            sys.exit('Cannot detect %s, try setting %s env variable' %
                     (cls.BINARY_NAME, cls.ENV_NAME))
        return None

    def __init__(self, exit_on_absent=True):
        self.target = self.detect(exit_on_absent)

    @classmethod
    def ensure(cls):
        raise NotImplementedError()

    def __call__(self):
        raise NotImplementedError
