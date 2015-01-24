package manager

import (
	"database/sql"
	"time"

	"github.com/go-martini/martini"
	"github.com/mountkin/dockerclient"
)

type Manager struct {
	addr   string
	errch  chan<- error
	db     *sql.DB
	m      *martini.ClassicMartini
	client dockerclient.Client
}

func New(addr string, db *sql.DB, ch chan<- error) (*Manager, error) {
	mgr := &Manager{
		addr:  addr,
		errch: ch,
		db:    db,
	}
	err := mgr.InitMartini()
	if err != nil {
		return nil, err
	}

	// connect to docker remote API
	mgr.client, err = dockerclient.NewDockerClientTimeout("unix:///var/run/docker.sock", nil, 3*time.Second)
	if err != nil {
		return nil, err
	}
	return mgr, nil
}

func (mgr *Manager) Start() {
	go func() {
		mgr.m.RunOnAddr(mgr.addr)
		mgr.errch <- nil
	}()
}
