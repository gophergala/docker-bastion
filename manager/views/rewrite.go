package views

import (
	"net/http"

	"github.com/martini-contrib/sessions"
)

var rewriteMap = map[string]string{
	"/signin":     "/signin.html",
	"/containers": "/dashboard.html",
	"/users":      "/dashboard.html",
}

func (v *Views) Rewrite(r *http.Request, ss sessions.Session) {
	if r.URL.Path == "/" {
		if ss.Get("uid") == nil {
			r.URL.Path = "/" + v.Index
		} else {
			r.URL.Path = "/containers"
		}
	}

	if t, ok := rewriteMap[r.URL.Path]; ok {
		r.URL.Path = t
	}
}
