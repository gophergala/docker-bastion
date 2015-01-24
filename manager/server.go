package manager

import (
	"net/http"
)

type Manager struct {
	addr  string
	errch chan<- error
}

func New(addr string, ch chan<- error) (*Manager, error) {
	return &Manager{
		addr:  addr,
		errch: ch,
	}, nil
}

func (mgr *Manager) Start() {
	http.ListenAndServe(mgr.addr, nil)
}
