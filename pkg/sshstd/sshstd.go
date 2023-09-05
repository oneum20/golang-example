package sshstd

import (
	"fmt"
	"log"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	ServerConfig *ssh.ClientConfig
	Protocal     string
	Addr         string
}

func getConfig() *Config {
	sshConfig := &ssh.ClientConfig{
		User: "vagrant",
		Auth: []ssh.AuthMethod{
			ssh.Password("vagrant"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	config := Config{
		ServerConfig: sshConfig,
		Protocal:     "tcp",
		Addr:         "localhost:2222",
	}

	return &config
}

func newConn(config *Config) *ssh.Client {
	conn, err := ssh.Dial(config.Protocal, config.Addr, config.ServerConfig)
	if err != nil {
		log.Fatal(err.Error())
	}

	return conn
}

func newSession(conn *ssh.Client) *ssh.Session {
	session, err := conn.NewSession()
	if err != nil {
		log.Fatal(err.Error())
	}

	return session
}

func ExecCase01() {
	config := getConfig()
	conn := newConn(config)
	defer conn.Close()

	session := newSession(conn)
	defer session.Close()

	cmd := "ls -al"
	session.RequestPty(cmd, 80, 50, ssh.TerminalModes{})
	output, err := session.Output(cmd)

	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Println(string(output))
}
