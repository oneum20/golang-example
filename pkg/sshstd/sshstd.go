package sshstd

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
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
		fmt.Println(">> reading...")
		n, err := s.Stdout.Read(data)
		fmt.Println(">> n: ", n)

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
			fmt.Print(s.Buffer)
		case <-t.C:
			// return
		}
	}
}

func MuxShell(w io.Writer, r io.Reader) (chan<- string, <-chan string) {
	in := make(chan string, 1)
	out := make(chan string, 1)
	var wg sync.WaitGroup
	wg.Add(1) //for the shell itself
	go func() {
		for cmd := range in {
			wg.Add(1)
			w.Write([]byte(cmd + "\n"))
			wg.Wait()
		}
	}()
	go func() {
		var (
			buf [65 * 1024]byte
			t   int
		)
		for {
			n, err := r.Read(buf[t:])
			if err != nil {
				fmt.Println(err)
				close(in)
				close(out)
				return
			}
			t += n
			if buf[t-2] == '$' { //assuming the $PS1 == 'sh-4.3$ '
				out <- string(buf[:t])
				t = 0
				wg.Done()
			}
		}
	}()
	return in, out
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
	s.Stdin.Write([]byte("ls \n"))

	s.Stdin.Write([]byte("pwd \n"))
	session.Wait()

}

func ExecCase03() {
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

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		log.Fatal(err)
	}

	// 입출력 스트림 설정
	s.Stdin, _ = session.StdinPipe()
	s.Stdout, _ = session.StdoutPipe()

	// TTY를 통해 원격 서버로부터 데이터 읽기
	go func() {
		io.Copy(os.Stdout, s.Stdout)
	}()

	// 사용자 입력을 TTY로 전송
	go func() {
		io.Copy(s.Stdin, os.Stdin)
	}()

	session.Shell()
	session.Wait()

}

func ExecCase04() {
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

	if err := session.RequestPty("xterm", 80, 40, ssh.TerminalModes{}); err != nil {
		log.Fatal(err.Error())
	}

	s.Stdin, _ = session.StdinPipe()
	s.Stdout, _ = session.StdoutPipe()

	go s.reader()
	go s.getData()

	go func() {
		io.Copy(s.Stdin, os.Stdin)
	}()

	session.Shell()
	session.Wait()

}
