package sshd

import (
	"net"

	log "github.com/Sirupsen/logrus"
)

type SSHD struct {
	addr  string
	errch chan<- error
}

func New(addr string, ch chan<- error) (*SSHD, error) {
	return &SSHD{
		addr:  addr,
		errch: ch,
	}, nil
}

func (sshd *SSHD) Start() {
	go func() {
		l, err := net.Listen("tcp", sshd.addr)
		if err != nil {
			sshd.errch <- err
			return
		}

		for {
			conn, err := l.Accept()
			if err != nil {
				log.Error("Accept:", err)
				continue
			}
			go sshd.serve(conn)
		}
	}()
}

func (sshd *SSHD) serve(conn net.Conn) {
	conn.Write([]byte("Hello world"))
	conn.Close()
}
