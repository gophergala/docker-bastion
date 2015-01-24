package sshd

import (
	"database/sql"
	"io/ioutil"
	"net"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/gophergala/docker-bastion/config"
	"golang.org/x/crypto/ssh"
)

type SSHD struct {
	addr        string
	errch       chan<- error
	sshconfig   *ssh.ServerConfig
	hostkeyPath string
	db          *sql.DB
}

func New(addr string, db *sql.DB, ch chan<- error) (*SSHD, error) {
	sshd := &SSHD{
		addr:        addr,
		errch:       ch,
		hostkeyPath: config.DATA_DIR + "/hostkey.rsa",
		db:          db,
	}

	if fp, err := os.Open(sshd.hostkeyPath); os.IsNotExist(err) {
		key, err := GenHostKey()
		if err != nil {
			return nil, err
		}
		err = ioutil.WriteFile(sshd.hostkeyPath, key, 0400)
		if err != nil {
			return nil, err
		}
	} else {
		fp.Close()
	}

	if err := sshd.initServerConfig(); err != nil {
		return nil, err
	}

	return sshd, nil
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
