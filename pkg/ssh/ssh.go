package ssh

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

// ZymeClient is base interface for ssh package.
type ZymeClient interface {
	Close()
	PutFile(localPath, remotePath string, newlineConversion, overwrite bool, makeExecutable bool) error
	GetFile(localPath, remotePath string, overwrite bool) error
	ExecuteCommand(command string, getOutput bool) error
	Equals(other ZymeClient) bool
	Split(path string) (dir, file string)
}

// MakeZymeClient establishes SSH connection between server and client;
// hostName, userName and pkeyFile arguments determine parameters of SSH connection;
// If usage of proxy is unnecessary, proxyHost argument should be set as empty string.
func MakeZymeClient(hostName, userName, pkeyFile, proxyHost string) (ZymeClient, error) {
	var client *ssh.Client

	network := "tcp"
	sshPort := "22"

	log.WithFields(log.Fields{
		"hostName":  hostName,
		"userName":  userName,
		"pkeyFile":  pkeyFile,
		"proxyHost": proxyHost,
	}).Info("MakeZymeClient ...")

	pkeyFileContent, err := ioutil.ReadFile(pkeyFile)
	if err != nil {
		log.WithFields(log.Fields{
			"pkeyFile": pkeyFile,
		}).Errorf("MakeZymeClient: %s", err)

		return nil, err
	}

	pkey, err := ssh.ParsePrivateKey(pkeyFileContent)
	if err != nil {
		log.Errorf("MakeZymeClient: %s", err)
		return nil, err
	}

	config := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(pkey),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if proxyHost != "" {
		sockProxyDialer, err := proxy.SOCKS5(network, proxyHost, nil, proxy.Direct)
		if err != nil {
			log.WithFields(log.Fields{
				"network":   network,
				"proxyHost": proxyHost,
			}).Errorf("MakeZymeClient: %s", err)

			return nil, err
		}

		sockConn, err := sockProxyDialer.Dial(network, net.JoinHostPort(hostName, sshPort))
		if err != nil {
			log.WithFields(log.Fields{
				"network":  network,
				"hostName": hostName,
				"hostPort": sshPort,
			}).Errorf("MakeZymeClient: %s", err)

			return nil, err
		}

		sshConn, newChanel, request, err := ssh.NewClientConn(sockConn, net.JoinHostPort(hostName, sshPort), config)
		if err != nil {
			log.WithFields(log.Fields{
				"sockConn":     sockConn,
				"hostName":     hostName,
				"hostPort":     sshPort,
				"clientConfig": config,
			}).Errorf("MakeZymeClient: %s", err)

			return nil, err
		}

		client = ssh.NewClient(sshConn, newChanel, request)
	} else {
		client, err = ssh.Dial(network, net.JoinHostPort(hostName, sshPort), config)
		if err != nil {
			log.WithFields(log.Fields{
				"network":      network,
				"hostName":     hostName,
				"hostPort":     sshPort,
				"clientConfig": config,
			}).Errorf("MakeZymeClient: %s", err)

			return nil, err
		}
	}

	return &zymeClient{
		internalClient: client,
		comp: comparison{
			hostname:   hostName,
			username:   userName,
			privateKey: pkeyFile,
		},
	}, nil
}

type comparison struct {
	hostname   string
	username   string
	privateKey string
}
type zymeClient struct {
	internalClient *ssh.Client
	comp           comparison
}

func (client *zymeClient) Equals(other ZymeClient) bool {
	casted, ok := other.(*zymeClient)
	if !ok {
		return false
	}

	return client.comp == casted.comp
}

func (client *zymeClient) Close() {
	client.internalClient.Close()
}

func (client *zymeClient) PutFile(localPath, remotePath string, newlineConversion, overwrite bool,
	makeExecutable bool) error {
	// open an SFTP session over an existing ssh connection.
	sftpClient, err := sftp.NewClient(client.internalClient)
	if err != nil {
		log.WithFields(log.Fields{
			"client": client.internalClient,
		}).Errorf("zymeClient.PutFile: %s", err)

		return err
	}

	defer sftpClient.Close()

	_, err = sftpClient.Lstat(remotePath)
	if err == nil {
		if !overwrite {
			err := fmt.Errorf("remote file %s already exist; overwrite prohibited", remotePath)
			log.WithFields(log.Fields{
				"remotePath": remotePath,
				"overwrite":  overwrite,
			}).Errorf("zymeClient.PutFile: %s", err)

			return err
		}

		log.WithFields(log.Fields{
			"remotePath": remotePath,
		}).Warn("zymeClient.PutFile: overwriting ...")
	}

	dir, _ := sftp.Split(remotePath)
	if dir != "" {
		err = sftpClient.MkdirAll(dir)
		if err != nil {
			log.WithFields(log.Fields{
				"remote-dir": dir,
			}).Errorf("zymeClient.PutFile: can't create remote directory: %s", err)

			return err
		}
	}

	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		log.WithFields(log.Fields{
			"remotePath": remotePath,
		}).Errorf("zymeClient.PutFile: %s", err)

		return err
	}
	defer remoteFile.Close()

	localFile, err := os.Open(localPath)
	if err != nil {
		log.WithFields(log.Fields{
			"localPath": localPath,
		}).Errorf("zymeClient.PutFile: %s", err)

		return err
	}
	defer localFile.Close()

	if !newlineConversion {
		if _, err := io.Copy(remoteFile, localFile); err != nil {
			log.WithFields(log.Fields{
				"remotePath": remotePath,
				"localPath":  localPath,
			}).Errorf("zymeClient.PutFile: %s", err)

			return err
		}
	} else {
		chunksize := 1024 * 1024
		readBuffer := make([]byte, chunksize)

		for {
			countReadBuffer, err := localFile.Read(readBuffer)
			if err == io.EOF {
				break
			} else if err != nil {
				log.WithFields(log.Fields{
					"readBuffer": string(readBuffer),
				}).Errorf("zymeClient.PutFile: %s", err)

				return err
			}

			writeBuffer := bytes.ReplaceAll(readBuffer[:countReadBuffer], []byte("\r"), []byte(""))

			if _, err = remoteFile.Write(writeBuffer); err != nil {
				log.WithFields(log.Fields{
					"writeBuffer": string(writeBuffer),
				}).Errorf("zymeClient.PutFile: %s", err)

				return err
			}
		}
	}

	if makeExecutable {
		makeFileExecutable(*client, remotePath)
	}

	return nil
}

func (client *zymeClient) GetFile(remotePath, localPath string, overwrite bool) error {
	// open an SFTP session over an existing ssh connection.
	sftpClient, err := sftp.NewClient(client.internalClient)
	if err != nil {
		log.WithFields(log.Fields{
			"client": client.internalClient,
		}).Errorf("zymeClient.GetFile: %s", err)

		return err
	}
	defer sftpClient.Close()

	_, err = os.Stat(localPath)
	if os.IsExist(err) {
		if !overwrite {
			err := fmt.Errorf("local file %s already exist; overwrite prohibited", localPath)
			log.WithFields(log.Fields{
				"localPath": localPath,
				"overwrite": overwrite,
			}).Errorf("zymeClient.GetFile: %s", err)

			return err
		}

		log.WithFields(log.Fields{
			"localPath": localPath,
		}).Warn("zymeClient.GetFile: overwriting ...")
	}

	dir, _ := filepath.Split(localPath)
	if dir != "" {
		err = os.MkdirAll(dir, 0750)
		if err != nil {
			log.WithFields(log.Fields{
				"local-dir": dir,
			}).Errorf("zymeClient.GetFile: can't create local directory: %s", err)

			return err
		}
	}

	localFile, err := os.Create(localPath)
	if err != nil {
		log.WithFields(log.Fields{
			"localPath": localPath,
		}).Errorf("zymeClient.GetFile: %s", err)

		return err
	}
	defer localFile.Close()

	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		log.WithFields(log.Fields{
			"remotePath": remotePath,
		}).Errorf("zymeClient.GetFile: %s", err)

		return err
	}
	defer remoteFile.Close()

	if _, err := io.Copy(localFile, remoteFile); err != nil {
		log.WithFields(log.Fields{
			"remotePath": remotePath,
			"localPath":  localPath,
		}).Errorf("zymeClient.GetFile: %s", err)

		return err
	}

	return nil
}

func (client *zymeClient) ExecuteCommand(command string, getOutput bool) error {
	session, err := client.internalClient.NewSession()
	if err != nil {
		log.Errorf("zymeClient.ExecuteCommand: %s", err)
		return err
	}

	defer session.Close()

	/*stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		//log
		return err
	}

	stderrPipe, err := session.StderrPipe()
	if err != nil {
		//log
		return err
	}*/

	if getOutput {
		session.Stdout = os.Stdout
		session.Stderr = os.Stderr
	}

	return session.Run(command)
}

func (client *zymeClient) Split(path string) (dir, file string) {
	return sftp.Split(path)
}

func makeFileExecutable(client zymeClient, path string) error {
	sftpClient, err := sftp.NewClient(client.internalClient)
	if err != nil {
		log.WithFields(log.Fields{
			"client": client.internalClient,
		}).Errorf("ssh.makeFileExecutable: %s", err)

		return err
	}
	defer sftpClient.Close()

	fileInfo, err := sftpClient.Stat(path)
	if err != nil {
		log.WithFields(log.Fields{
			"sftpClient": sftpClient,
			"path":       path,
		}).Errorf("ssh.makeFileExecutable: %s", err)

		return err
	}

	fileModeX := fileInfo.Mode() | 0111
	if err := sftpClient.Chmod(path, fileModeX); err != nil {
		log.WithFields(log.Fields{
			"sftpClient": sftpClient,
			"path":       path,
			"mode":       fileModeX,
		}).Errorf("ssh.makeFileExecutable: %s", err)

		return err
	}

	return nil
}
