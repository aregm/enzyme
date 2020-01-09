import os
import json
from mock import mock_open, patch, MagicMock
import pytest
from wrapped.make_variables_files import (
    Config, FileReadingError, FileWritingError,
    InvalidConfigError, PackerConfig, TerraformConfig)


class TestConfigAPI:

    # synthetic data
    AWS_VALID_CRED = ('User name,Password,Access key ID,Secret access key,'
                      'Console login link\n'
                      'ec2-user,,,'
                      ','
                      'https://aws.amazon.com/console\n')

    EXPECTED_TRANSFORMED_AWS_CRED = (
        '[default]\n'
        'aws_access_key_id = \n'
        'aws_secret_access_key = \n'
    )

    def create_example_config(self):
        with patch('__builtin__.open',
                   mock_open(read_data=TerraformConfig.EXAMPLE_CONFIG)):
            config = TerraformConfig()
        return config

    @pytest.mark.parametrize('data, message', [
        ({"without_provider": "None"},
         'Invalid file: .*, lacks <"provider": "XXX"> entry. In place of XXX '
         'choose one of: aws, gcp'),

        ({"provider": "qwerty"},
         'Invalid file: .* provider is not defined - choose one of: aws, gcp'),
    ])
    def test_check_provider_errors(self, data, message):
        config = self.create_example_config()
        with pytest.raises(InvalidConfigError, match=message):
            config.check_provider(data)

    def test_check_provider(self):
        config = self.create_example_config()
        assert 'gcp' == config.check_provider({"provider": "gcp"})

    @pytest.mark.parametrize('data, message', [
        ({"worker_count": "1"},
         'Invalid file: .* worker_count should be >= 2'),

        ({"worker_count": "XXX"},
         'Invalid file: .* worker_count should be a number'),

        ({"provider": "aws"},
         'Invalid file: .* worker_count field was not found'),
    ])
    def test_check_worker_count_errors(self, data, message):
        config = self.create_example_config()
        with pytest.raises(InvalidConfigError, match=message):
            config.check_worker_count(data)

    def test_check_worker_count(self):
        config = self.create_example_config()
        assert 2 == config.check_worker_count({"worker_count": "2"})

    def test_transform_aws_credentials_errors(self, bad_aws_cred, message):
        config = self.create_example_config()
        with pytest.raises(InvalidConfigError, match=message):
            config.transform_aws_credentials(bad_aws_cred)

    def test_transform_aws_credentials(self):
        config = self.create_example_config()
        transformed_aws_cred = config.transform_aws_credentials(
            self.AWS_VALID_CRED)
        assert transformed_aws_cred == self.EXPECTED_TRANSFORMED_AWS_CRED

    def test_prepare_credentials_errors(self):
        config = self.create_example_config()
        message = 'missing file .*'

        with pytest.raises(FileReadingError, match=message):
            with patch('__builtin__.open', side_effect=IOError):
                config.prepare_credentials()

    def test_prepare_credentials_aws_provider(self):
        aws_conf = ('{\n'
                    '    "provider": "aws",\n'
                    '    "worker_count": "2"\n'
                    '}\n')

        with patch('__builtin__.open', mock_open(read_data=aws_conf)):
            config = TerraformConfig()
        assert config.current_provider == 'aws'

        with patch('__builtin__.open',
                   mock_open(read_data=self.AWS_VALID_CRED)):
            with patch('wrapped.make_variables_files.Config.write_config',
                       return_value=None):
                config.prepare_credentials()

        # prepare_credentials must setuping
        # `AWS_SHARED_CREDENTIALS_FILE` environment variable
        cred_path = os.path.abspath(
            Config.CREDENTIALS_FILES[config.current_provider])
        assert os.environ['AWS_SHARED_CREDENTIALS_FILE'] == cred_path

    def test_write_config_errors(self):
        config = self.create_example_config()
        message = 'Error writing to file: .*RHOC.variables.tfvars'

        with pytest.raises(FileWritingError, match=message):
            with patch('__builtin__.open', side_effect=IOError):
                config.write_config()

    def test_write_config(self):
        config = self.create_example_config()
        content = 'Content from `_read` method; `test_write_config` test case'
        write_str = 'string for `file.write` method'
        mock = MagicMock(spec=file)

        with patch('__builtin__.open', mock_open(mock=mock)):
            with patch('wrapped.make_variables_files.Config._read',
                       return_value=content):
                config.write_config(file_name='check', str_config=write_str)

        file_handle = mock.return_value
        file_handle.write.assert_called_with(write_str)

    def test_read_errors(self):
        config = self.create_example_config()

        with patch('__builtin__.open', side_effect=IOError):
            assert config._read('example') is None

    def test_read(self):
        config = self.create_example_config()
        expected_return = 'Text for check'

        with patch('__builtin__.open', mock_open(read_data=expected_return)):
            assert expected_return == config._read('example')


class TestParsingConfig:

    EXAMPLE_USER_DEFINED_STR = ('{\n'
                                '    "provider": "gcp",\n'
                                '    "worker_count": "4",\n'
                                '    "key_name": "test.pem",\n'
                                '    "image_name": "CentOS 7",\n'
                                '    "project_name": "RHOC cluster"\n'
                                '}')

    def test_terraform_init(self):
        patch_str = ('wrapped.make_variables_files.'
                     'TerraformConfig.CHMOD_COMMAND')

        chmod_command = 'chmod_command = "dummy_value"'

        with patch(patch_str, new=chmod_command):
            with patch('__builtin__.open',
                       mock_open(read_data=self.EXAMPLE_USER_DEFINED_STR)):
                config = TerraformConfig()
                content = config.content

        expected_content = ('%s\n'
                            'image_name = "CentOS 7"\n'
                            'key_name = "test.pem"\n'
                            'project_name = "RHOC cluster"\n'
                            'worker_count = "4"\n') % (chmod_command)

        assert content == expected_content

    def test_packer_init(self):
        with patch('__builtin__.open',
                   mock_open(read_data=self.EXAMPLE_USER_DEFINED_STR)):
            pk_conf = PackerConfig()

        expected_json = json.loads('{\n'
                                   '    "image_name": "CentOS 7",\n'
                                   '    "project_name": "RHOC cluster"\n'
                                   '}')

        assert pk_conf.current_builder == PackerConfig.PACKER_BUILDERS['gcp']
        assert json.loads(pk_conf.content) == expected_json

    @pytest.mark.parametrize('config', [TerraformConfig, PackerConfig])
    def test_no_such_file_reading_error(self, config):
        message = 'No such file or directory: C:/config.json'

        with pytest.raises(FileReadingError, match=message):
            config('C:/config.json')

    @pytest.mark.parametrize('config', [TerraformConfig, PackerConfig])
    def test_incorrect_json_format_reading_error(self, config):
        message = ("File .* incorrect JSON format: Expecting ,"
                   " delimiter: line 3 column 2 .*")

        data = '{\n"provider": "gcp"\n "worker_count": "2"}'

        with patch('__builtin__.open', mock_open(read_data=data)):
            with pytest.raises(FileReadingError, match=message):
                config()

    def test_supported_providers(self):
        providers = sorted(Config.CREDENTIALS_FILES.keys())
        assert providers == sorted(PackerConfig.PACKER_BUILDERS.keys())
