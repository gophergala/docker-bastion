package sshd

// The SSH server code is inspired from Jaime Pillora's gist
// at https://gist.github.com/jpillora/b480fde82bff51a06238.
// Thank you Jaime Pillora for your detailed example.

import (
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	log "github.com/Sirupsen/logrus"
	"github.com/kr/pty"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"
)

var (
	ErrInvalidContainerName = errors.New("login name must has the format of USERNAME.CONTAINER_NAME, where CONTAINER_NAME is the container that you want to login")
	ErrInvalidPassword      = errors.New("username or password is incorrect")
	ErrAccessDenied         = errors.New("you don't have the privileges to access the container")
)

func (sshd *SSHD) initServerConfig() error {
	keybytes, err := ioutil.ReadFile(sshd.hostkeyPath)
	if err != nil {
		return err
	}
	cfg := &ssh.ServerConfig{}
	key, err := ssh.ParsePrivateKey(keybytes)
	if err != nil {
		return err
	}
	cfg.AddHostKey(key)
	cfg.PasswordCallback = sshd.passwordCallback
	sshd.sshconfig = cfg
	return nil
}

func (sshd *SSHD) passwordCallback(meta ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
	user := sshd.getUsername(meta.User())
	container := sshd.getContainerId(meta.User())
	if len(container) != 12 {
		return nil, ErrInvalidContainerName
	}

	stmt, err := sshd.db.Prepare("select id, password from users where name = ?")
	if err != nil {
		return nil, err
	}

	var (
		savedPass    string
		uid          int
		hasContainer int
	)
	err = stmt.QueryRow(user).Scan(&uid, &savedPass)
	stmt.Close()
	if err != nil {
		log.Error("Scan: ", err)
		return nil, ErrInvalidPassword
	}

	err = bcrypt.CompareHashAndPassword([]byte(savedPass), pass)
	if err != nil {
		return nil, ErrInvalidPassword
	}

	stmt, err = sshd.db.Prepare("select 1 from containers where user_id = ? and cid like ?")
	if err != nil {
		return nil, err
	}
	stmt.QueryRow(uid, container+"%").Scan(&hasContainer)
	stmt.Close()
	if hasContainer != 1 {
		return nil, ErrAccessDenied
	}

	return nil, nil
}

// The username of the SSH connection containers 2 fields separated by a dot.
// The first field is the user's account name,
// the second field is the first 12 characters of the container id to login.
func (sshd *SSHD) getUsername(in string) string {
	segs := strings.SplitN(in, ".", 2)
	return segs[0]
}
func (sshd *SSHD) getContainerId(in string) string {
	segs := strings.SplitN(in, ".", 2)
	if len(segs) != 2 {
		return ""
	}
	return segs[1]
}

func (sshd *SSHD) serve(conn net.Conn) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, sshd.sshconfig)
	if err != nil {
		log.Error("ssh.NewServerConn: ", err)
		conn.Write([]byte(err.Error()))
		conn.Close()
		return
	}
	container := sshd.getContainerId(sshConn.User())

	log.Infof("new connection %s/%s", sshConn.RemoteAddr(), string(sshConn.ClientVersion()))
	go sshd.handleRequests(reqs)
	go sshd.handleChannels(chans, container)
}

func (sshd *SSHD) handleRequests(requests <-chan *ssh.Request) {
	for req := range requests {
		log.Info("out-of-band request: %s", req.Type)
	}
}

func (sshd *SSHD) handleChannels(chans <-chan ssh.NewChannel, container string) {
	for ch := range chans {
		if ch.ChannelType() != "session" {
			ch.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, requests, err := ch.Accept()
		if err != nil {
			log.Error(err)
			continue
		}
		sshd.handleChannel(channel, requests, container)
	}
}

func (sshd *SSHD) handleChannel(channel ssh.Channel, requests <-chan *ssh.Request, container string) {
	cmd := exec.Command("docker", "exec", "-i", "-t", container, "/bin/bash")
	closeChannel := func() {
		channel.Close()
		_, err := cmd.Process.Wait()
		if err != nil {
			log.Errorf("failed to exit docker exec (%s)", err)
		}
	}

	fp, err := pty.Start(cmd)
	if err != nil {
		log.Error("pty.Start: ", err)
		closeChannel()
		return
	}

	go func() {
		for req := range requests {
			log.Debugf("new request: %s", req.Type)
			switch req.Type {
			case "shell":
				if len(req.Payload) == 0 {
					req.Reply(true, nil)
				}
			case "pty-req":
				termLen := req.Payload[3]
				w, h := sshd.parseDims(req.Payload[termLen+4:])
				sshd.setWinsize(fp.Fd(), w, h)
				req.Reply(true, nil)
			case "window-change":
				w, h := sshd.parseDims(req.Payload)
				sshd.setWinsize(fp.Fd(), w, h)
			case "env":
			}
		}
	}()

	var once sync.Once
	cp := func(dst io.Writer, src io.Reader) {
		io.Copy(dst, src)
		once.Do(closeChannel)
	}
	go cp(channel, fp)
	go cp(fp, channel)
}

// parseDims extracts terminal dimensions (width x height) from the provided buffer.
func (sshd *SSHD) parseDims(b []byte) (uint32, uint32) {
	w := binary.BigEndian.Uint32(b)
	h := binary.BigEndian.Uint32(b[4:])
	return w, h
}

// Winsize stores the Height and Width of a terminal.
type Winsize struct {
	Height uint16
	Width  uint16
	x      uint16 // unused
	y      uint16 // unused
}

// SetWinsize sets the size of the given pty.
func (sshd *SSHD) setWinsize(fd uintptr, w, h uint32) {
	ws := &Winsize{Width: uint16(w), Height: uint16(h)}
	syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
}
