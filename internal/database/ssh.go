package database

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"

	"golang.org/x/crypto/ssh"
)

// Setup establishes an SSH tunnel and returns the modified connection string
func SetupTunnel(config Config) (string, func(), error) {
	// Read private key
	key, err := ioutil.ReadFile(config.SSHKey)
	if err != nil {
		return "", nil, fmt.Errorf("unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return "", nil, fmt.Errorf("unable to parse private key: %v", err)
	}

	// Setup SSH client config
	sshConfig := &ssh.ClientConfig{
		User: config.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to SSH server
	sshClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", config.SSHHost, config.SSHPort), sshConfig)
	if err != nil {
		return "", nil, fmt.Errorf("unable to connect to SSH server: %v", err)
	}

	// Setup local listener
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		sshClient.Close()
		return "", nil, fmt.Errorf("unable to setup local listener: %v", err)
	}

	localPort := listener.Addr().(*net.TCPAddr).Port

	// Start SSH tunnel
	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				log.Printf("Error accepting connection: %v", err)
				return
			}

			remoteConn, err := sshClient.Dial("tcp", fmt.Sprintf("%s:%d", config.Host, config.Port))
			if err != nil {
				log.Printf("Error dialing remote server: %v", err)
				localConn.Close()
				return
			}

			go copyConn(localConn, remoteConn)
			go copyConn(remoteConn, localConn)
		}
	}()

	// Build connection string using local port
	connStr := fmt.Sprintf(
		"host=localhost port=%d dbname=%s user=%s",
		localPort,
		config.Database,
		config.User,
	)

	if config.Password != "" {
		connStr += fmt.Sprintf(" password=%s", config.Password)
	}

	cleanup := func() {
		listener.Close()
		sshClient.Close()
	}

	return connStr, cleanup, nil
}

func copyConn(dst, src net.Conn) {
	defer dst.Close()
	defer src.Close()
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Printf("Error copying connection: %v", err)
	}
}
