package manager

import (
	"database/sql"
	"time"

	log "github.com/Sirupsen/logrus"
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
	err = mgr.addDefaultUser()
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

func (mgr *Manager) addDefaultUser() error {
	exists := 0
	err := mgr.db.QueryRow("select count(*) from admins").Scan(&exists)
	if err != nil {
		log.Error(err)
		return err
	}
	if exists == 0 {
		// add a default user, name: admin, password: password
		_, err := mgr.db.Exec("insert into admins(name, password) values(?,?)", "admin", "$2a$11$6BDpJ3NbRgDruOqY15Nsy.7vx3a32p.JdQjy1NxOWoshKDaRflUti")
		if err != nil {
			log.Error(err)
			return err
		}
		log.Info("A default user is created. username: admin, password: password")
	}
	return nil
}
