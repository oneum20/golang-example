package sshstd

import (
	"fmt"
	"io"
	"log"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHBuf struct {
	Stdout io.Reader
	Stdin  io.WriteCloser
	Buffer string
	Data   chan []byte
}

type Config struct {
	ServerConfig *ssh.ClientConfig
	Protocol     string
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
		Protocol:     "tcp",
		Addr:         "localhost:2222",
	}

	return &config
}

func newConn(config *Config) *ssh.Client {
	conn, err := ssh.Dial(config.Protocol, config.Addr, config.ServerConfig)
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

func (s *SSHBuf) reader() {
	var data = make([]byte, 1024)

	for {
		n, err := s.Stdout.Read(data)

		if err != nil {
			fmt.Println(err.Error())
			return
		}
		s.Data <- data[:n]
	}
}

func (s *SSHBuf) getData() {
	t := time.NewTimer(time.Second)
	defer t.Stop()

	for {
		select {
		case d := <-s.Data:
			s.Buffer += string(d)
			fmt.Println(s.Buffer)
		case <-t.C:
			return
		}
	}
}

func ExecCase01() {
	config := getConfig()
	conn := newConn(config)
	defer func() {
		fmt.Println("Connection Closed")
		conn.Close()
	}()

	session := newSession(conn)
	defer func() {
		fmt.Println("Session Closed")
		session.Close()
	}()

	cmd := "ls -al"
	session.RequestPty(cmd, 80, 50, ssh.TerminalModes{})
	output, err := session.Output(cmd)

	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Println(string(output))
}

func ExecCase02() {
	s := &SSHBuf{Data: make(chan []byte, 1024)}
	config := getConfig()
	conn := newConn(config)
	defer func() {
		fmt.Println("Connection Closed")
		conn.Close()
	}()

	session := newSession(conn)
	defer func() {
		fmt.Println("Session Closed")
		session.Close()
	}()

	s.Stdout, _ = session.StdoutPipe()
	s.Stdin, _ = session.StdinPipe()

	err := session.Shell()
	if err != nil {
		log.Fatal(err.Error())
	}

	go s.reader()
	go s.getData()

	s.Stdin.Write([]byte("ls -al\n"))

	session.Wait()

	fmt.Println(s.Stdout)
	fmt.Println(s.Buffer)

}
