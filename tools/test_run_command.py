import pytest
from mock import mock_open, patch, MagicMock, Mock
from run_command import (
    ZymeClient, check_script,
    get_credentials_from_terraform, ensure_count_worker_nodes)
import paramiko
from make_proxy_socket import proxy_socket  # noqa
import wrapped


class TestRunCommand:

    TERRAFORM = wrapped.TerraformWrapper(exit_on_absent=True)

    def create_zyme_client(self, use_proxy):
        return ZymeClient(hostname='test_host', username='test_user',
                          pkey_file='test_pkey.pem', use_proxy=use_proxy)

    @pytest.mark.parametrize('use_proxy', [True, False])
    def test_create_connected_client(self, use_proxy):
        proxy_mock = Mock()
        kwargs = {'hostname': 'test_host', 'key_filename': 'test_pkey.pem',
                  'port': 22, 'username': 'test_user'}

        with patch('paramiko.SSHClient', return_value=Mock()):
            with patch('make_proxy_socket.proxy_socket',
                       return_value=proxy_mock):
                with self.create_zyme_client(use_proxy) as zyme_patched_client:
                    zyme_patched_client.client.connect.assert_called_with(
                        sock=proxy_mock if use_proxy else None, **kwargs)
                    if use_proxy:
                        proxy_mock.connect.assert_called_with(
                            ('test_host', 22))

    def test_put_file_with_no_connection(self):
        msg = 'ZymeClient is not connected'
        test_client = self.create_zyme_client(False)

        with patch('__builtin__.open',
                   mock_open(read_data='file content\n')):
            with pytest.raises(AssertionError, match=msg):
                test_client.put_file('./test_local_file', './test_remote_file')

    '''
    def test_put_string(self):
        sftp_mock = MagicMock(spec=paramiko.sftp_client.SFTPClient)
        mock = Mock()
        file_content = 'file content\n'
        remotepath = './test_remote_file'

        zyme_client = self.create_zyme_client(False)

        with patch.object(sftp_mock, 'stat', side_effect=IOError):
            with patch.object(zyme_client, '_get_sftp_client',
                              return_value=sftp_mock):
                with patch.object(zyme_client, '_put_string', new=mock):
                    zyme_client.put_string(file_content, remotepath, 'w')

        mock.assert_called_with(sftp_mock, file_content, remotepath, 'w')
        sftp_mock.close.assert_called_with()
    '''

    @pytest.mark.parametrize('read_data, expected', [
        ('#!/bin/bash -i\necho "test"', True),
        ('import os\n print "test"', False)
    ])
    def test_check_script(self, read_data, expected):
        with patch('__builtin__.open', mock_open(read_data=read_data)):
            res = check_script('test_file')

        assert expected == res

    @pytest.mark.parametrize('output,expected_tuple', [
        ('There is nothing to output.',
         (None, None, None)),

        ('{\n"gcp_login_address": {\n"value": "10.10.10.10"\n}\n}',
         ('10.10.10.10', None, None)),

        ('{\n"username": {\n"value": "ec2-user"\n}\n}',
         (None, 'ec2-user', None)),

        ('{\n"pkey_file": {\n"value": "~/pkey.pem"\n}\n}',
         (None, None, '~/pkey.pem'))
    ])
    def test_get_credentials_from_terraform_errors_without_input(
            self, output, expected_tuple):

        msg = ('some of the variables {hostname, username, '
               'pkey_file} are undefined')

        with patch('wrapped.terraform.TerraformWrapper._get_variable_output',
                   return_value=output):
            with patch('wrapped.TerraformWrapper',
                       return_value=self.TERRAFORM):
                with pytest.raises(ValueError, match=msg):
                    get_credentials_from_terraform()

    @pytest.mark.parametrize('output,input_tuple,expected_tuple', [
        ('There is nothing to output.',
         ('10.10.10.10', 'ec2-user', '~/pkey.pem'),
         ('10.10.10.10', 'ec2-user', '~/pkey.pem')),

        ('{\n"gcp_login_address": {\n"value": "10.10.10.10"\n}\n}',
         (None, 'ec2-user', '~/pkey.pem'),
         ('10.10.10.10', 'ec2-user', '~/pkey.pem')),

        ('{\n"aws_login_address": {\n"value": "10.10.10.10"\n}\n}',
         (None, 'ec2-user', '~/pkey.pem'),
         ('10.10.10.10', 'ec2-user', '~/pkey.pem')),

        ('{\n"username": {\n"value": "ec2-user"\n}\n}',
         ('10.10.10.10', None, '~/pkey.pem'),
         ('10.10.10.10', 'ec2-user', '~/pkey.pem')),

        ('{\n"pkey_file": {\n"value": "~/pkey.pem"\n}\n}',
         ('10.10.10.10', 'ec2-user', None),
         ('10.10.10.10', 'ec2-user', '~/pkey.pem')),

        ('{\n"gcp_login_address": {\n"value": "10.10.10.10"\n},'
         '\n"username": {\n"value": "ec2-user"\n},'
         '\n"pkey_file": {\n"value": "~/pkey.pem"\n}\n}',
         (None, None, None),
         ('10.10.10.10', 'ec2-user', '~/pkey.pem')),

        ('{\n"gcp_login_address": {\n"value": "10.10.10.10"\n},'
         '\n"username": {\n"value": "ec2-user"\n},'
         '\n"pkey_file": {\n"value": "~/pkey.pem"\n}\n}',
         ('10.10.10.11', 'gcp-user', '~/private_key.pem'),
         ('10.10.10.11', 'gcp-user', '~/private_key.pem')),
    ])
    def test_get_credentials_from_terraform(self, output,
                                            input_tuple, expected_tuple):
        with patch('wrapped.terraform.TerraformWrapper._get_variable_output',
                   return_value=output):
            with patch('wrapped.TerraformWrapper',
                       return_value=self.TERRAFORM):
                cred_tuple = get_credentials_from_terraform(*input_tuple)

        assert cred_tuple == expected_tuple

    def test_ensure_count_worker_nodes_error(self):
        output = 'There is nothing to output.'
        with patch('wrapped.terraform.TerraformWrapper._get_variable_output',
                   return_value=output):
            with patch('wrapped.TerraformWrapper',
                       return_value=self.TERRAFORM):
                with pytest.raises(ValueError, match='Cluster not created'):
                    ensure_count_worker_nodes(2)

    @pytest.mark.parametrize('output,desired_count,expected_count', [
        ('{\n"value": "2"\n}', 3, 2),
        ('{\n"value": "2"\n}', 1, 1),
        ('{\n"value": "2"\n}', None, 2)
    ])
    def test_ensure_count_worker_nodes(self, output,
                                       desired_count, expected_count):
        with patch('wrapped.terraform.TerraformWrapper._get_variable_output',
                   return_value=output):
            with patch('wrapped.TerraformWrapper',
                       return_value=self.TERRAFORM):
                count_workers = ensure_count_worker_nodes(desired_count)

        assert count_workers == expected_count
