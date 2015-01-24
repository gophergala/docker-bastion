package manager

import (
	"database/sql"

	"github.com/go-martini/martini"
)

type Manager struct {
	addr  string
	errch chan<- error
	db    *sql.DB
	m     *martini.ClassicMartini
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
	return mgr, nil
}

func (mgr *Manager) Start() {
	go func() {
		mgr.m.RunOnAddr(mgr.addr)
		mgr.errch <- nil
	}()
}
