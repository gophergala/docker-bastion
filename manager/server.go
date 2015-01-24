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
	go func() {
		mgr.errch <- http.ListenAndServe(mgr.addr, nil)
	}()
}
