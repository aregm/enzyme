import json
import os
import sys


class UserFileError(Exception):
    pass


class FileReadingError(UserFileError):
    pass


class FileWritingError(UserFileError):
    pass


class InvalidConfigError(UserFileError):
    pass


class Config(object):
    TARGET_CONFIG_NAME = None

    READ_FILE = os.path.abspath('user_defined.json')

    CREDENTIALS_FILES = {
        'aws': os.path.abspath('tf_modules/providers/aws/credentials'),
        'gcp': os.path.abspath('tf_modules/providers/gcp/credentials.json')
    }

    PROVIDERS = sorted(CREDENTIALS_FILES.keys())

    EXAMPLE_CONFIG = ('{\n'
                      '    "provider": "gcp",\n'
                      '    "worker_count": "2"\n'
                      '}\n')

    def __init__(self):
        self._config_contents = ''
        self._current_provider = 'none'

    def _check_aws_access_keys(self, access_keys):
        """
        Checks format keys.

        Parameters
        ----------
        access_keys: list

        """
        if len(access_keys) != 2:
            raise InvalidConfigError(
                '%s does not contain access key id or secret access key' %
                (self.CREDENTIALS_FILES[self._current_provider]))

        # the access key ID must be 20 characters long
        if len(access_keys[0]) != 20:
            raise InvalidConfigError(
                '%s contains wrong access key ID' %
                (self.CREDENTIALS_FILES[self._current_provider]))

        # the secret access key must be 40 characters long
        if len(access_keys[1]) != 40:
            raise InvalidConfigError(
                '%s contains wrong secret access key' %
                (self.CREDENTIALS_FILES[self._current_provider]))

    def transform_aws_credentials(self, contents):
        """
        Converts aws credentials string to supported format
        for Packer and Terraform and writes it to credential file.

        Parameters
        ----------
        contents: str

        Returns
        -------
        aws_credentials: str

        Notes
        -----
        Expected format of AWS credential file:
        User name,Password,Access key ID,Secret access key,Console login link
        X,X,X,X,X

        """
        value_line = contents.split('\n')[1]
        access_keys = value_line.split(',')[2:4]

        self._check_aws_access_keys(access_keys)

        aws_credentials = ('[default]\n'
                           'aws_access_key_id = %s\n'
                           'aws_secret_access_key = %s\n'
                           ) % (access_keys[0], access_keys[1])

        return aws_credentials

    def check_provider(self, data):
        """
        Checks that `data` containts supported provider from `self.PROVIDERS`.

        Parameters
        ----------
        data: dict

        Returns
        -------
        provider: str

        """
        try:
            if data['provider'] not in self.PROVIDERS:
                raise InvalidConfigError(
                    'Invalid file: %s, provider is not defined - choose one '
                    'of: %s' % (self.READ_FILE, ', '.join(self.PROVIDERS)))
        except KeyError:
            raise InvalidConfigError(
                'Invalid file: %s, lacks <"provider": "XXX"> entry. In place '
                'of XXX choose one of: %s' % (self.READ_FILE,
                                              ', '.join(self.PROVIDERS)))
        return data['provider']

    def _read(self, file_name):
        """
        Reads content from 'file_name'.

        Parameters
        ----------
        file_name: str

        Returns
        -------
        content: str

        """
        try:
            with open(file_name, "r") as input_file:
                content = input_file.read()
        except IOError:
            return None
        return content

    def write_config(self, file_name=None, str_config=None):
        """
        Writes `str_config` to `file_name`.

        Parameters
        ----------
        file_name: str, default None
        str_config: str, default None

        Notes
        -----
        Recording only happens if `file_name` content is different from
        `str_config`.

        """

        if file_name is None:
            file_name = self.TARGET_CONFIG_NAME

        if str_config is None:
            str_config = self._config_contents

        try:
            content = self._read(file_name)
            if content != str_config or content is None:
                with open(file_name, "w") as output_file:
                    output_file.write(str_config)
        except IOError:
            raise FileWritingError('Error writing to file: %s' % (file_name))

    def _read_json(self, conf_file=None):
        """
        Reads json file `conf_file` to dict.

        Parameters
        ----------
        conf_file: str, default None

        Returns
        -------
        data: dict

        Notes
        -----
        By default read from `self.READ_FILE`.
        If `conf_file` isn't None, than it would be saved in `self.READ_FILE`.

        """
        json_file = self.READ_FILE if conf_file is None else conf_file

        # This variable contains information about the read file.
        self.READ_FILE = json_file

        try:
            with open(json_file) as json_data:
                data = json.load(json_data)

        except IOError:
            raise FileReadingError("No such file or directory: %s" % json_file)

        except ValueError as ex:
            raise FileReadingError(
                'File %s has incorrect JSON format: %s' % (json_file, ex))

        return data

    def check_worker_count(self, data):
        """
        Checks that `data[worker_count]` >= 2.

        Parameters
        ----------
        data: dict

        Returns
        -------
        worker_count: int

        """

        try:
            worker_count = int(data["worker_count"])
            if worker_count < 2:
                raise InvalidConfigError(
                    'Invalid file: %s, worker_count should be >= 2' %
                    self.READ_FILE)

        except KeyError:
            raise InvalidConfigError(
                'Invalid file: %s, worker_count field was not found' %
                (self.READ_FILE))

        except ValueError:
            raise InvalidConfigError(
                'Invalid file: %s, worker_count should be a number' %
                self.READ_FILE)

        return worker_count

    def prepare_credentials(self):
        """
        Prepares provider's credentials for using by Packer and Terraform.

        """
        try:
            with open(self.CREDENTIALS_FILES[self._current_provider]) as _file:
                contents = _file.read()

            if self._current_provider == 'aws':
                if not contents.startswith('[default]'):
                    aws_credentials = self.transform_aws_credentials(contents)
                    self.write_config(
                        self.CREDENTIALS_FILES[self._current_provider],
                        aws_credentials)

                os.environ['AWS_SHARED_CREDENTIALS_FILE'] = os.path.abspath(
                    self.CREDENTIALS_FILES[self._current_provider])

        except IOError:
            raise FileReadingError(
                'missing file %s' %
                (self.CREDENTIALS_FILES[self._current_provider]))

    def read_config(self, conf_file=None):
        """
        Reads json file `conf_file` to dict with provider and worker_count
        checks.

        Parameters
        ----------
        conf_file: str, default None

        Returns
        -------
        data: dict

        Notes
        -----
        Setup `self._current_provider`.

        """
        data = self._read_json(conf_file)
        self._current_provider = self.check_provider(data)
        data["worker_count"] = str(self.check_worker_count(data))

        return data

    @property
    def current_provider(self):
        return self._current_provider

    @property
    def content(self):
        return self._config_contents


class PackerConfig(Config):
    TARGET_CONFIG_NAME = os.path.abspath('variables.json')

    PACKER_KEYS = ['image_name', 'project_name', 'aws_region', 'gcp_zone']
    PACKER_BUILDERS = {'aws': 'amazon-ebs', 'gcp': 'googlecompute'}

    def __init__(self, conf_file=None):
        """
        Reads information from `conf_file` with user's variables and prepares
        it for writing to files that will override default variables
        in configs for Packer.

        Parameters
        ----------
        conf_file: str, default None

        """
        super(PackerConfig, self).__init__()
        data = self.read_config(conf_file)
        self._current_builder = self.PACKER_BUILDERS[self.current_provider]
        self._config_contents = self.generate_json_file_content(data)

    @property
    def current_builder(self):
        return self._current_builder

    @classmethod
    def generate_json_file_content(cls, data):
        """
        Creates string, that can be written to `.json` file.

        Parameters
        ----------
        data: dict

        Returns
        -------
        content: str

        Notes
        -----
        example of return string:
        {
            "key_name": "test.pem",
            "worker_count": "4"
        }

        Variables are sorted in lexicographical order.

        """
        config = {}
        for key, value in data.items():
            if key in cls.PACKER_KEYS:
                config[key] = str(value)

        return json.dumps(config, indent=4, sort_keys=True)

    @staticmethod
    def prepare_packer_config():
        conf = PackerConfig()
        conf.prepare_credentials()
        conf.write_config()
        return conf


class TerraformConfig(Config):
    TARGET_CONFIG_NAME = os.path.abspath('variables.tfvars')

    TERRAFORM_KEYS = [
        'worker_count', 'image_name', 'aws_region',
        'gcp_region', 'gcp_zone',
        'aws_instance_type_login_node',
        'aws_instance_type_worker_node',
        'gcp_instance_type_login_node',
        'gcp_instance_type_worker_node',
        'key_name', 'cluster_name',
        'project_name', 'login_node_root_size'
    ]

    CHMOD_COMMAND = 'chmod_command = ""' if sys.platform == 'win32' \
        else 'chmod_command = "chmod 600 %v"'

    def __init__(self, conf_file=None):
        """
        Reads information from `conf_file` with user's variables and prepares
        it for writing to files that will override default variables
        in configs for Terraform.

        Parameters
        ----------
        conf_file: str, default None

        """
        super(TerraformConfig, self).__init__()
        data = self.read_config(conf_file)
        self._config_contents = self.generate_tfvars_file_content(data)

    @classmethod
    def generate_tfvars_file_content(cls, data):
        """
        Creates string, that can be written to `.tfvars` file.

        Parameters
        ----------
        data: dict

        Returns
        -------
        content: str

        Notes
        -----
        example `.tfvars` format:
            key_name = "test.pem"
            worker_count = "4"

        Variables are sorted in lexicographical order.

        """
        content = cls.CHMOD_COMMAND + '\n'
        for key in sorted(data):
            if key in cls.TERRAFORM_KEYS:
                content += '%s = "%s"\n' % (key, data[key])
        return content

    @staticmethod
    def prepare_terraform_config():
        conf = TerraformConfig()
        conf.prepare_credentials()
        conf.write_config()
        return conf
