// +build dev

package views

import (
	"net/http"

	"github.com/martini-contrib/sessions"
)

type Views struct {
	Index string
	s     http.Handler
}

func New(fallback string) *Views {
	v := &Views{
		Index: fallback,
		s:     http.FileServer(http.Dir("manager/views/assets")),
	}
	return v
}

func (v *Views) ServeHTTP(w http.ResponseWriter, r *http.Request, ss sessions.Session) {
	v.Rewrite(r, ss)
	v.s.ServeHTTP(w, r)
}
